package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/hooks"
	"github.com/steveyegge/gastown/internal/mail"
	"github.com/steveyegge/gastown/internal/tmux"
)

// MailOrchestratorConfig holds configuration for the mail orchestrator daemon.
type MailOrchestratorConfig struct {
	// PollInterval is how often to poll mail queues.
	PollInterval time.Duration `json:"poll_interval"`

	// MaxRetries is the maximum number of delivery retries.
	MaxRetries int `json:"max_retries"`

	// RetryDelay is the delay between retries.
	RetryDelay time.Duration `json:"retry_delay"`

	// DeadLetterThreshold is when to move to dead letter queue.
	DeadLetterThreshold int `json:"dead_letter_threshold"`

	// EnablePriorityProcessing enables priority-based message processing.
	EnablePriorityProcessing bool `json:"enable_priority_processing"`
}

// DefaultMailOrchestratorConfig returns default configuration.
func DefaultMailOrchestratorConfig() *MailOrchestratorConfig {
	return &MailOrchestratorConfig{
		PollInterval:             30 * time.Second,
		MaxRetries:               3,
		RetryDelay:               5 * time.Minute,
		DeadLetterThreshold:      5,
		EnablePriorityProcessing: true,
	}
}

// MailOrchestrator orchestrates mail delivery for async agent communication.
// It monitors mail queues, delivers messages based on priority and routing rules,
// and handles retries and dead letter queues.
type MailOrchestrator struct {
	townRoot string
	config   *MailOrchestratorConfig
	router   *mail.Router
	tmux     *tmux.Tmux
	logger   *log.Logger
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup

	// Queue tracking
	inboundMu      sync.Mutex
	inboundQueue   []*QueuedMessage
	outboundMu     sync.Mutex
	outboundQueue  []*QueuedMessage
	deadLetterMu   sync.Mutex
	deadLetterQueue []*QueuedMessage
}

// QueuedMessage represents a message in processing queue.
type QueuedMessage struct {
	Message     *mail.Message `json:"message"`
	Attempts    int           `json:"attempts"`
	LastAttempt time.Time     `json:"last_attempt"`
	QueuedAt    time.Time     `json:"queued_at"`
	Error       string        `json:"error,omitempty"`
}

// NewMailOrchestrator creates a new mail orchestrator daemon.
func NewMailOrchestrator(townRoot string, config *MailOrchestratorConfig, logger *log.Logger) *MailOrchestrator {
	if logger == nil {
		logger = log.New(os.Stderr, "[mail-orchestrator] ", log.LstdFlags)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &MailOrchestrator{
		townRoot:        townRoot,
		config:          config,
		router:          mail.NewRouterWithTownRoot(townRoot, townRoot),
		tmux:            tmux.NewTmux(),
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		inboundQueue:    make([]*QueuedMessage, 0),
		outboundQueue:   make([]*QueuedMessage, 0),
		deadLetterQueue: make([]*QueuedMessage, 0),
	}
}

// Start starts the mail orchestrator daemon.
func (mo *MailOrchestrator) Start() error {
	mo.logger.Println("Mail orchestrator starting...")

	// Load persistent queues from disk
	if err := mo.loadQueues(); err != nil {
		mo.logger.Printf("Warning: failed to load queues: %v", err)
	}

	// Start queue processors
	mo.wg.Add(3)
	go mo.processInboundQueue()
	go mo.processOutboundQueue()
	go mo.processRetryQueue()

	mo.logger.Println("Mail orchestrator started")
	return nil
}

// Stop stops the mail orchestrator daemon.
func (mo *MailOrchestrator) Stop() {
	mo.logger.Println("Mail orchestrator stopping...")
	mo.cancel()
	mo.wg.Wait()

	// Save persistent queues to disk
	if err := mo.saveQueues(); err != nil {
		mo.logger.Printf("Warning: failed to save queues: %v", err)
	}

	mo.logger.Println("Mail orchestrator stopped")
}

// processInboundQueue monitors beads for incoming messages and routes them.
func (mo *MailOrchestrator) processInboundQueue() {
	defer mo.wg.Done()

	ticker := time.NewTicker(mo.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mo.ctx.Done():
			return
		case <-ticker.C:
			mo.scanInboundMessages()
		}
	}
}

// scanInboundMessages scans beads for undelivered messages.
func (mo *MailOrchestrator) scanInboundMessages() {
	// Query beads for open messages with delivery=interrupt or priority=urgent
	beadsDir := filepath.Join(mo.townRoot, ".beads")
	if err := beads.EnsureCustomTypes(beadsDir); err != nil {
		mo.logger.Printf("Error ensuring custom types: %v", err)
		return
	}

	// Query for high priority messages that need immediate delivery
	messages, err := mo.queryPriorityMessages(beadsDir)
	if err != nil {
		mo.logger.Printf("Error querying priority messages: %v", err)
		return
	}

	// Queue messages for delivery
	mo.inboundMu.Lock()
	defer mo.inboundMu.Unlock()

	for _, msg := range messages {
		// Check if already queued
		if mo.isQueued(msg.ID) {
			continue
		}

		mo.inboundQueue = append(mo.inboundQueue, &QueuedMessage{
			Message:  msg,
			QueuedAt: time.Now(),
		})
		mo.logger.Printf("Queued inbound message %s (from: %s, to: %s, priority: %s)",
			msg.ID, msg.From, msg.To, msg.Priority)
	}

	// Sort by priority if enabled
	if mo.config.EnablePriorityProcessing {
		mo.sortByPriority(mo.inboundQueue)
	}
}

// queryPriorityMessages queries beads for high priority messages.
func (mo *MailOrchestrator) queryPriorityMessages(beadsDir string) ([]*mail.Message, error) {
	// Query for messages with:
	// - delivery=interrupt (explicit interrupt delivery)
	// - priority=urgent (implicit interrupt)
	// - priority=high (next priority tier)

	args := []string{"list",
		"--type", "message",
		"--status", "open",
		"--json",
	}

	stdout, err := mail.RunBdCommand(args, filepath.Dir(beadsDir), beadsDir)
	if err != nil {
		return nil, err
	}

	var beadsMsgs []mail.BeadsMessage
	if err := json.Unmarshal(stdout, &beadsMsgs); err != nil {
		if len(stdout) == 0 || string(stdout) == "null" {
			return nil, nil
		}
		return nil, err
	}

	// Filter for high priority or interrupt delivery
	var messages []*mail.Message
	for _, bm := range beadsMsgs {
		msg := bm.ToMessage()

		// Check if needs orchestrated delivery
		if mo.needsOrchestration(msg) {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// needsOrchestration returns true if message needs orchestrated delivery.
func (mo *MailOrchestrator) needsOrchestration(msg *mail.Message) bool {
	// Interrupt delivery always needs orchestration
	if msg.Delivery == mail.DeliveryInterrupt {
		return true
	}

	// Urgent priority always needs orchestration
	if msg.Priority == mail.PriorityUrgent {
		return true
	}

	// High priority messages get orchestration
	if msg.Priority == mail.PriorityHigh {
		return true
	}

	return false
}

// isQueued checks if message is already in any queue.
func (mo *MailOrchestrator) isQueued(msgID string) bool {
	// Check inbound
	for _, qm := range mo.inboundQueue {
		if qm.Message.ID == msgID {
			return true
		}
	}

	// Check outbound
	mo.outboundMu.Lock()
	defer mo.outboundMu.Unlock()
	for _, qm := range mo.outboundQueue {
		if qm.Message.ID == msgID {
			return true
		}
	}

	// Check dead letter
	mo.deadLetterMu.Lock()
	defer mo.deadLetterMu.Unlock()
	for _, qm := range mo.deadLetterQueue {
		if qm.Message.ID == msgID {
			return true
		}
	}

	return false
}

// sortByPriority sorts queued messages by priority.
func (mo *MailOrchestrator) sortByPriority(queue []*QueuedMessage) {
	sort.SliceStable(queue, func(i, j int) bool {
		// Priority order: urgent > high > normal > low
		pi := mo.priorityValue(queue[i].Message.Priority)
		pj := mo.priorityValue(queue[j].Message.Priority)

		if pi != pj {
			return pi > pj
		}

		// Same priority - older first
		return queue[i].QueuedAt.Before(queue[j].QueuedAt)
	})
}

// priorityValue converts priority to numeric value for sorting.
func (mo *MailOrchestrator) priorityValue(p mail.Priority) int {
	switch p {
	case mail.PriorityUrgent:
		return 3
	case mail.PriorityHigh:
		return 2
	case mail.PriorityNormal:
		return 1
	case mail.PriorityLow:
		return 0
	default:
		return 1
	}
}

// processOutboundQueue delivers queued messages.
func (mo *MailOrchestrator) processOutboundQueue() {
	defer mo.wg.Done()

	ticker := time.NewTicker(mo.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mo.ctx.Done():
			return
		case <-ticker.C:
			mo.deliverQueuedMessages()
		}
	}
}

// deliverQueuedMessages attempts delivery of queued messages.
func (mo *MailOrchestrator) deliverQueuedMessages() {
	mo.inboundMu.Lock()
	messages := make([]*QueuedMessage, len(mo.inboundQueue))
	copy(messages, mo.inboundQueue)
	mo.inboundQueue = nil
	mo.inboundMu.Unlock()

	for _, qm := range messages {
		if err := mo.deliverMessage(qm); err != nil {
			mo.logger.Printf("Failed to deliver message %s: %v", qm.Message.ID, err)
			mo.handleDeliveryFailure(qm, err)
		} else {
			mo.logger.Printf("Delivered message %s to %s", qm.Message.ID, qm.Message.To)
			mo.fireMailReceivedHook(qm.Message)
		}
	}
}

// deliverMessage delivers a single message based on routing rules.
func (mo *MailOrchestrator) deliverMessage(qm *QueuedMessage) error {
	msg := qm.Message

	// Update attempt tracking
	qm.Attempts++
	qm.LastAttempt = time.Now()

	// Interrupt delivery - inject into agent session
	if msg.Delivery == mail.DeliveryInterrupt || msg.Priority == mail.PriorityUrgent {
		return mo.deliverInterrupt(msg)
	}

	// Normal delivery - message already in mailbox via beads
	// Just trigger notification
	return mo.notifyRecipient(msg)
}

// deliverInterrupt delivers message via tmux interrupt.
func (mo *MailOrchestrator) deliverInterrupt(msg *mail.Message) error {
	// Resolve address to session ID
	sessionIDs := addressToSessionIDs(msg.To)
	if len(sessionIDs) == 0 {
		return fmt.Errorf("cannot resolve session for address: %s", msg.To)
	}

	// Try each possible session ID
	var lastErr error
	for _, sessionID := range sessionIDs {
		hasSession, err := mo.tmux.HasSession(sessionID)
		if err != nil || !hasSession {
			lastErr = fmt.Errorf("session %s not found", sessionID)
			continue
		}

		// Inject as system reminder
		notification := fmt.Sprintf("📬 URGENT MESSAGE from %s: %s\n\n%s\n\nRun 'gt mail inbox' to respond.",
			msg.From, msg.Subject, msg.Body)

		if err := mo.tmux.NudgeSession(sessionID, notification); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("no active session found for: %s", msg.To)
}

// notifyRecipient sends notification for queued message.
func (mo *MailOrchestrator) notifyRecipient(msg *mail.Message) error {
	sessionIDs := addressToSessionIDs(msg.To)
	if len(sessionIDs) == 0 {
		return nil // No session to notify
	}

	for _, sessionID := range sessionIDs {
		hasSession, err := mo.tmux.HasSession(sessionID)
		if err != nil || !hasSession {
			continue
		}

		notification := fmt.Sprintf("📬 New mail from %s: %s. Run 'gt mail inbox' to read.",
			msg.From, msg.Subject)
		return mo.tmux.NudgeSession(sessionID, notification)
	}

	return nil
}

// addressToSessionIDs converts mail address to tmux session IDs.
// This is a simplified version - full implementation would use mail.Router logic.
func addressToSessionIDs(address string) []string {
	// Town-level agents
	if address == "mayor/" || address == "mayor" {
		return []string{"gt-mayor"}
	}
	if address == "deacon/" || address == "deacon" {
		return []string{"gt-deacon"}
	}

	// For rig addresses, would need full parsing
	// This is placeholder - production would use mail.Router's addressToSessionIDs
	return nil
}

// handleDeliveryFailure handles failed delivery attempts.
func (mo *MailOrchestrator) handleDeliveryFailure(qm *QueuedMessage, err error) {
	qm.Error = err.Error()

	// Check if exceeded retry threshold
	if qm.Attempts >= mo.config.MaxRetries {
		mo.moveToDeadLetter(qm)
		return
	}

	// Requeue for retry
	mo.outboundMu.Lock()
	defer mo.outboundMu.Unlock()
	mo.outboundQueue = append(mo.outboundQueue, qm)
}

// moveToDeadLetter moves message to dead letter queue.
func (mo *MailOrchestrator) moveToDeadLetter(qm *QueuedMessage) {
	mo.deadLetterMu.Lock()
	defer mo.deadLetterMu.Unlock()

	mo.deadLetterQueue = append(mo.deadLetterQueue, qm)
	mo.logger.Printf("Moved message %s to dead letter queue after %d attempts: %s",
		qm.Message.ID, qm.Attempts, qm.Error)

	// Log to beads with dead-letter label
	if err := mo.markDeadLetter(qm.Message); err != nil {
		mo.logger.Printf("Error marking dead letter: %v", err)
	}
}

// markDeadLetter marks message as dead letter in beads.
func (mo *MailOrchestrator) markDeadLetter(msg *mail.Message) error {
	beadsDir := filepath.Join(mo.townRoot, ".beads")
	args := []string{"label", "add", msg.ID, "dead-letter"}
	_, err := mail.RunBdCommand(args, filepath.Dir(beadsDir), beadsDir)
	return err
}

// processRetryQueue processes messages in retry queue.
func (mo *MailOrchestrator) processRetryQueue() {
	defer mo.wg.Done()

	ticker := time.NewTicker(mo.config.RetryDelay)
	defer ticker.Stop()

	for {
		select {
		case <-mo.ctx.Done():
			return
		case <-ticker.C:
			mo.retryFailedMessages()
		}
	}
}

// retryFailedMessages attempts redelivery of failed messages.
func (mo *MailOrchestrator) retryFailedMessages() {
	mo.outboundMu.Lock()
	defer mo.outboundMu.Unlock()

	now := time.Now()
	var retry []*QueuedMessage
	var keep []*QueuedMessage

	for _, qm := range mo.outboundQueue {
		// Check if enough time has passed since last attempt
		if now.Sub(qm.LastAttempt) >= mo.config.RetryDelay {
			retry = append(retry, qm)
		} else {
			keep = append(keep, qm)
		}
	}

	mo.outboundQueue = keep

	// Move to inbound queue for retry
	if len(retry) > 0 {
		mo.inboundMu.Lock()
		mo.inboundQueue = append(mo.inboundQueue, retry...)
		mo.inboundMu.Unlock()

		mo.logger.Printf("Retrying %d failed messages", len(retry))
	}
}

// fireMailReceivedHook fires the mail-received hook.
func (mo *MailOrchestrator) fireMailReceivedHook(msg *mail.Message) {
	runner, err := hooks.NewHookRunner(mo.townRoot)
	if err != nil {
		return // No hooks configured
	}

	ctx := &hooks.HookContext{
		WorkingDir: mo.townRoot,
		Metadata: map[string]interface{}{
			"from":    msg.From,
			"to":      msg.To,
			"subject": msg.Subject,
		},
	}

	results := runner.Fire(hooks.EventMailReceived, ctx)
	for _, result := range results {
		if !result.Success {
			mo.logger.Printf("Hook execution failed: %v", result.Error)
		}
	}
}

// loadQueues loads persistent queues from disk.
func (mo *MailOrchestrator) loadQueues() error {
	queueDir := filepath.Join(mo.townRoot, "daemon", "mail-queues")
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		return err
	}

	// Load inbound queue
	if err := mo.loadQueue(filepath.Join(queueDir, "inbound.json"), &mo.inboundQueue); err != nil {
		mo.logger.Printf("Warning: failed to load inbound queue: %v", err)
	}

	// Load outbound queue
	if err := mo.loadQueue(filepath.Join(queueDir, "outbound.json"), &mo.outboundQueue); err != nil {
		mo.logger.Printf("Warning: failed to load outbound queue: %v", err)
	}

	// Load dead letter queue
	if err := mo.loadQueue(filepath.Join(queueDir, "dead-letter.json"), &mo.deadLetterQueue); err != nil {
		mo.logger.Printf("Warning: failed to load dead letter queue: %v", err)
	}

	return nil
}

// loadQueue loads a queue from JSON file.
func (mo *MailOrchestrator) loadQueue(path string, queue *[]*QueuedMessage) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, queue)
}

// saveQueues saves persistent queues to disk.
func (mo *MailOrchestrator) saveQueues() error {
	queueDir := filepath.Join(mo.townRoot, "daemon", "mail-queues")
	if err := os.MkdirAll(queueDir, 0755); err != nil {
		return err
	}

	// Save inbound queue
	mo.inboundMu.Lock()
	if err := mo.saveQueue(filepath.Join(queueDir, "inbound.json"), mo.inboundQueue); err != nil {
		mo.logger.Printf("Warning: failed to save inbound queue: %v", err)
	}
	mo.inboundMu.Unlock()

	// Save outbound queue
	mo.outboundMu.Lock()
	if err := mo.saveQueue(filepath.Join(queueDir, "outbound.json"), mo.outboundQueue); err != nil {
		mo.logger.Printf("Warning: failed to save outbound queue: %v", err)
	}
	mo.outboundMu.Unlock()

	// Save dead letter queue
	mo.deadLetterMu.Lock()
	if err := mo.saveQueue(filepath.Join(queueDir, "dead-letter.json"), mo.deadLetterQueue); err != nil {
		mo.logger.Printf("Warning: failed to save dead letter queue: %v", err)
	}
	mo.deadLetterMu.Unlock()

	return nil
}

// saveQueue saves a queue to JSON file.
func (mo *MailOrchestrator) saveQueue(path string, queue []*QueuedMessage) error {
	data, err := json.MarshalIndent(queue, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetStats returns orchestrator statistics.
func (mo *MailOrchestrator) GetStats() *MailOrchestratorStats {
	mo.inboundMu.Lock()
	inboundCount := len(mo.inboundQueue)
	mo.inboundMu.Unlock()

	mo.outboundMu.Lock()
	outboundCount := len(mo.outboundQueue)
	mo.outboundMu.Unlock()

	mo.deadLetterMu.Lock()
	deadLetterCount := len(mo.deadLetterQueue)
	mo.deadLetterMu.Unlock()

	return &MailOrchestratorStats{
		InboundQueueSize:    inboundCount,
		OutboundQueueSize:   outboundCount,
		DeadLetterQueueSize: deadLetterCount,
	}
}

// MailOrchestratorStats holds orchestrator statistics.
type MailOrchestratorStats struct {
	InboundQueueSize    int `json:"inbound_queue_size"`
	OutboundQueueSize   int `json:"outbound_queue_size"`
	DeadLetterQueueSize int `json:"dead_letter_queue_size"`
}
