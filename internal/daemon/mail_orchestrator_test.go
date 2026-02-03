package daemon

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/steveyegge/gastown/internal/mail"
)

func TestMailOrchestrator_QueueManagement(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	config.PollInterval = 100 * time.Millisecond

	mo := NewMailOrchestrator(tempDir, config, logger)

	// Test adding message to inbound queue
	msg := &mail.Message{
		ID:       "test-msg-1",
		From:     "mayor/",
		To:       "deacon/",
		Subject:  "Test Message",
		Body:     "Test body",
		Priority: mail.PriorityHigh,
		Delivery: mail.DeliveryInterrupt,
	}

	qm := &QueuedMessage{
		Message:  msg,
		QueuedAt: time.Now(),
	}

	mo.inboundMu.Lock()
	mo.inboundQueue = append(mo.inboundQueue, qm)
	mo.inboundMu.Unlock()

	// Check queue size
	stats := mo.GetStats()
	if stats.InboundQueueSize != 1 {
		t.Errorf("Expected inbound queue size 1, got %d", stats.InboundQueueSize)
	}
}

func TestMailOrchestrator_PrioritySort(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	mo := NewMailOrchestrator(tempDir, config, logger)

	// Create messages with different priorities
	msgs := []*QueuedMessage{
		{
			Message: &mail.Message{
				ID:       "msg-1",
				Priority: mail.PriorityNormal,
			},
			QueuedAt: time.Now(),
		},
		{
			Message: &mail.Message{
				ID:       "msg-2",
				Priority: mail.PriorityUrgent,
			},
			QueuedAt: time.Now().Add(1 * time.Second),
		},
		{
			Message: &mail.Message{
				ID:       "msg-3",
				Priority: mail.PriorityLow,
			},
			QueuedAt: time.Now(),
		},
		{
			Message: &mail.Message{
				ID:       "msg-4",
				Priority: mail.PriorityHigh,
			},
			QueuedAt: time.Now(),
		},
	}

	mo.sortByPriority(msgs)

	// Check order: urgent > high > normal > low
	expectedOrder := []string{"msg-2", "msg-4", "msg-1", "msg-3"}
	for i, qm := range msgs {
		if qm.Message.ID != expectedOrder[i] {
			t.Errorf("Position %d: expected %s, got %s", i, expectedOrder[i], qm.Message.ID)
		}
	}
}

func TestMailOrchestrator_NeedsOrchestration(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	mo := NewMailOrchestrator(tempDir, config, logger)

	tests := []struct {
		name     string
		msg      *mail.Message
		expected bool
	}{
		{
			name: "Interrupt delivery",
			msg: &mail.Message{
				Delivery: mail.DeliveryInterrupt,
				Priority: mail.PriorityNormal,
			},
			expected: true,
		},
		{
			name: "Urgent priority",
			msg: &mail.Message{
				Priority: mail.PriorityUrgent,
			},
			expected: true,
		},
		{
			name: "High priority",
			msg: &mail.Message{
				Priority: mail.PriorityHigh,
			},
			expected: true,
		},
		{
			name: "Normal priority, queue delivery",
			msg: &mail.Message{
				Priority: mail.PriorityNormal,
				Delivery: mail.DeliveryQueue,
			},
			expected: false,
		},
		{
			name: "Low priority",
			msg: &mail.Message{
				Priority: mail.PriorityLow,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mo.needsOrchestration(tt.msg)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMailOrchestrator_QueuePersistence(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	mo := NewMailOrchestrator(tempDir, config, logger)

	// Add messages to queues
	msg := &mail.Message{
		ID:       "persist-test-1",
		From:     "mayor/",
		To:       "deacon/",
		Subject:  "Persistence Test",
		Body:     "Test",
		Priority: mail.PriorityHigh,
	}

	qm := &QueuedMessage{
		Message:  msg,
		QueuedAt: time.Now(),
	}

	mo.inboundMu.Lock()
	mo.inboundQueue = append(mo.inboundQueue, qm)
	mo.inboundMu.Unlock()

	// Save queues
	if err := mo.saveQueues(); err != nil {
		t.Fatalf("Failed to save queues: %v", err)
	}

	// Create new orchestrator and load queues
	mo2 := NewMailOrchestrator(tempDir, config, logger)
	if err := mo2.loadQueues(); err != nil {
		t.Fatalf("Failed to load queues: %v", err)
	}

	// Check queue was loaded
	mo2.inboundMu.Lock()
	defer mo2.inboundMu.Unlock()

	if len(mo2.inboundQueue) != 1 {
		t.Errorf("Expected 1 message in inbound queue, got %d", len(mo2.inboundQueue))
	}

	if len(mo2.inboundQueue) > 0 && mo2.inboundQueue[0].Message.ID != "persist-test-1" {
		t.Errorf("Expected message ID 'persist-test-1', got %s", mo2.inboundQueue[0].Message.ID)
	}
}

func TestMailOrchestrator_DeadLetterQueue(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	config.MaxRetries = 2
	mo := NewMailOrchestrator(tempDir, config, logger)

	// Initialize beads for dead letter marking
	beadsDir := filepath.Join(tempDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatalf("Failed to create .beads: %v", err)
	}

	msg := &mail.Message{
		ID:       "dead-letter-test",
		From:     "mayor/",
		To:       "invalid/",
		Subject:  "Dead Letter Test",
		Body:     "Test",
		Priority: mail.PriorityHigh,
	}

	qm := &QueuedMessage{
		Message:  msg,
		QueuedAt: time.Now(),
		Attempts: 2, // Already at max retries
	}

	// Move to dead letter
	mo.moveToDeadLetter(qm)

	// Check dead letter queue
	stats := mo.GetStats()
	if stats.DeadLetterQueueSize != 1 {
		t.Errorf("Expected dead letter queue size 1, got %d", stats.DeadLetterQueueSize)
	}
}

func TestMailOrchestrator_RetryLogic(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	config.MaxRetries = 3
	config.RetryDelay = 1 * time.Second
	mo := NewMailOrchestrator(tempDir, config, logger)

	msg := &mail.Message{
		ID:       "retry-test",
		From:     "mayor/",
		To:       "deacon/",
		Subject:  "Retry Test",
		Body:     "Test",
		Priority: mail.PriorityHigh,
	}

	qm := &QueuedMessage{
		Message:     msg,
		QueuedAt:    time.Now(),
		Attempts:    1,
		LastAttempt: time.Now().Add(-2 * time.Second), // Old enough for retry
	}

	mo.outboundMu.Lock()
	mo.outboundQueue = append(mo.outboundQueue, qm)
	mo.outboundMu.Unlock()

	// Trigger retry processing
	mo.retryFailedMessages()

	// Check that message moved to inbound queue
	mo.inboundMu.Lock()
	defer mo.inboundMu.Unlock()

	if len(mo.inboundQueue) != 1 {
		t.Errorf("Expected 1 message in inbound queue after retry, got %d", len(mo.inboundQueue))
	}
}
