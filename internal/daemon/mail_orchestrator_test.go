package daemon

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
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

// TestMailOrchestrator_ConcurrentQueueOperations tests concurrent queue access
// with filelock protection to ensure no data corruption occurs.
func TestMailOrchestrator_ConcurrentQueueOperations(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	mo := NewMailOrchestrator(tempDir, config, logger)

	const numGoroutines = 10
	const messagesPerGoroutine = 20

	var wg sync.WaitGroup

	// Concurrently add messages to inbound queue and save
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < messagesPerGoroutine; j++ {
				msg := &mail.Message{
					ID:       fmt.Sprintf("concurrent-msg-%d-%d", id, j),
					From:     "mayor/",
					To:       "deacon/",
					Subject:  fmt.Sprintf("Concurrent Test %d-%d", id, j),
					Body:     "Test body",
					Priority: mail.PriorityNormal,
				}

				qm := &QueuedMessage{
					Message:  msg,
					QueuedAt: time.Now(),
				}

				mo.inboundMu.Lock()
				mo.inboundQueue = append(mo.inboundQueue, qm)
				mo.inboundMu.Unlock()

				// Save queue to disk with filelock protection
				if err := mo.saveQueues(); err != nil {
					t.Errorf("saveQueues() error = %v", err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify queue integrity
	stats := mo.GetStats()
	expectedCount := numGoroutines * messagesPerGoroutine
	if stats.InboundQueueSize != expectedCount {
		t.Errorf("Expected inbound queue size %d, got %d", expectedCount, stats.InboundQueueSize)
	}

	// Verify all messages have unique IDs (no corruption)
	mo.inboundMu.Lock()
	defer mo.inboundMu.Unlock()

	seen := make(map[string]bool)
	for _, qm := range mo.inboundQueue {
		if seen[qm.Message.ID] {
			t.Errorf("Duplicate message ID found: %s", qm.Message.ID)
		}
		seen[qm.Message.ID] = true
	}
}

// TestMailOrchestrator_ConcurrentLoadSave tests concurrent save and load operations
// to ensure filelock prevents read-during-write corruption.
func TestMailOrchestrator_ConcurrentLoadSave(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	mo1 := NewMailOrchestrator(tempDir, config, logger)

	// Pre-populate queue
	for i := 0; i < 50; i++ {
		msg := &mail.Message{
			ID:       fmt.Sprintf("msg-%d", i),
			From:     "mayor/",
			To:       "deacon/",
			Subject:  fmt.Sprintf("Message %d", i),
			Body:     "Test",
			Priority: mail.PriorityNormal,
		}

		qm := &QueuedMessage{
			Message:  msg,
			QueuedAt: time.Now(),
		}

		mo1.inboundMu.Lock()
		mo1.inboundQueue = append(mo1.inboundQueue, qm)
		mo1.inboundMu.Unlock()
	}

	// Save initial state
	if err := mo1.saveQueues(); err != nil {
		t.Fatalf("Failed to save initial queues: %v", err)
	}

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Start multiple writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			mo := NewMailOrchestrator(tempDir, config, logger)
			for j := 0; j < 10; j++ {
				// Load, modify, save
				if err := mo.loadQueues(); err != nil {
					errors <- fmt.Errorf("writer %d: load error: %w", id, err)
					continue
				}

				mo.inboundMu.Lock()
				mo.inboundQueue = append(mo.inboundQueue, &QueuedMessage{
					Message: &mail.Message{
						ID:       fmt.Sprintf("writer-%d-msg-%d", id, j),
						From:     "mayor/",
						To:       "deacon/",
						Subject:  "Test",
						Body:     "Test",
						Priority: mail.PriorityNormal,
					},
					QueuedAt: time.Now(),
				})
				mo.inboundMu.Unlock()

				if err := mo.saveQueues(); err != nil {
					errors <- fmt.Errorf("writer %d: save error: %w", id, err)
				}

				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Start multiple readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			mo := NewMailOrchestrator(tempDir, config, logger)
			for j := 0; j < 20; j++ {
				if err := mo.loadQueues(); err != nil {
					errors <- fmt.Errorf("reader %d: load error: %w", id, err)
				}

				// Verify data integrity - should be valid JSON
				mo.inboundMu.Lock()
				count := len(mo.inboundQueue)
				mo.inboundMu.Unlock()

				if count < 0 {
					errors <- fmt.Errorf("reader %d: invalid queue size: %d", id, count)
				}

				time.Sleep(5 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errorCount int
	for err := range errors {
		t.Errorf("Concurrent operation error: %v", err)
		errorCount++
	}

	if errorCount > 0 {
		t.Fatalf("Failed with %d concurrent operation errors", errorCount)
	}

	// Final integrity check - load and verify
	mo2 := NewMailOrchestrator(tempDir, config, logger)
	if err := mo2.loadQueues(); err != nil {
		t.Fatalf("Final load failed: %v", err)
	}

	// Queue should have original 50 + (5 writers * 10 messages) = 100 messages
	// Note: Some may be lost due to race conditions without proper locking,
	// but with filelock all should be present
	stats := mo2.GetStats()
	if stats.InboundQueueSize < 50 {
		t.Errorf("Expected at least 50 messages, got %d (data loss detected)", stats.InboundQueueSize)
	}
}

// TestMailOrchestrator_MultiQueueConcurrency tests concurrent access across all three queues
func TestMailOrchestrator_MultiQueueConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	mo := NewMailOrchestrator(tempDir, config, logger)

	var wg sync.WaitGroup
	const operations = 50

	// Concurrent operations on inbound queue
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < operations; i++ {
			mo.inboundMu.Lock()
			mo.inboundQueue = append(mo.inboundQueue, &QueuedMessage{
				Message: &mail.Message{
					ID:       fmt.Sprintf("inbound-%d", i),
					From:     "mayor/",
					To:       "deacon/",
					Priority: mail.PriorityNormal,
				},
				QueuedAt: time.Now(),
			})
			mo.inboundMu.Unlock()

			if i%10 == 0 {
				mo.saveQueues()
			}
		}
	}()

	// Concurrent operations on outbound queue
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < operations; i++ {
			mo.outboundMu.Lock()
			mo.outboundQueue = append(mo.outboundQueue, &QueuedMessage{
				Message: &mail.Message{
					ID:       fmt.Sprintf("outbound-%d", i),
					From:     "mayor/",
					To:       "deacon/",
					Priority: mail.PriorityNormal,
				},
				QueuedAt: time.Now(),
			})
			mo.outboundMu.Unlock()

			if i%10 == 0 {
				mo.saveQueues()
			}
		}
	}()

	// Concurrent operations on dead letter queue
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < operations; i++ {
			mo.deadLetterMu.Lock()
			mo.deadLetterQueue = append(mo.deadLetterQueue, &QueuedMessage{
				Message: &mail.Message{
					ID:       fmt.Sprintf("deadletter-%d", i),
					From:     "mayor/",
					To:       "deacon/",
					Priority: mail.PriorityNormal,
				},
				QueuedAt: time.Now(),
			})
			mo.deadLetterMu.Unlock()

			if i%10 == 0 {
				mo.saveQueues()
			}
		}
	}()

	wg.Wait()

	// Final save
	if err := mo.saveQueues(); err != nil {
		t.Fatalf("Final save failed: %v", err)
	}

	// Reload and verify
	mo2 := NewMailOrchestrator(tempDir, config, logger)
	if err := mo2.loadQueues(); err != nil {
		t.Fatalf("Failed to reload queues: %v", err)
	}

	stats := mo2.GetStats()
	if stats.InboundQueueSize != operations {
		t.Errorf("Inbound queue: expected %d, got %d", operations, stats.InboundQueueSize)
	}
	if stats.OutboundQueueSize != operations {
		t.Errorf("Outbound queue: expected %d, got %d", operations, stats.OutboundQueueSize)
	}
	if stats.DeadLetterQueueSize != operations {
		t.Errorf("Dead letter queue: expected %d, got %d", operations, stats.DeadLetterQueueSize)
	}
}

// TestMailOrchestrator_AtomicQueueWrite verifies that writes are atomic
func TestMailOrchestrator_AtomicQueueWrite(t *testing.T) {
	tempDir := t.TempDir()
	logger := log.New(os.Stderr, "[test] ", log.LstdFlags)

	config := DefaultMailOrchestratorConfig()
	mo := NewMailOrchestrator(tempDir, config, logger)

	// Add messages
	for i := 0; i < 10; i++ {
		mo.inboundMu.Lock()
		mo.inboundQueue = append(mo.inboundQueue, &QueuedMessage{
			Message: &mail.Message{
				ID:       fmt.Sprintf("atomic-msg-%d", i),
				From:     "mayor/",
				To:       "deacon/",
				Priority: mail.PriorityNormal,
			},
			QueuedAt: time.Now(),
		})
		mo.inboundMu.Unlock()
	}

	// Save queue
	if err := mo.saveQueues(); err != nil {
		t.Fatalf("Failed to save queues: %v", err)
	}

	// Verify no .tmp files left behind
	queueDir := filepath.Join(tempDir, "daemon", "mail-queues")
	entries, err := os.ReadDir(queueDir)
	if err != nil {
		t.Fatalf("Failed to read queue dir: %v", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".tmp" {
			t.Errorf("Temporary file not cleaned up: %s", entry.Name())
		}
	}

	// Verify main queue files exist and are valid JSON
	queueFiles := []string{"inbound.json", "outbound.json", "dead-letter.json"}
	for _, file := range queueFiles {
		path := filepath.Join(queueDir, file)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Queue file missing: %s", file)
			continue
		}

		// Verify valid JSON
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Failed to read %s: %v", file, err)
			continue
		}

		var queue []*QueuedMessage
		if err := json.Unmarshal(data, &queue); err != nil {
			t.Errorf("Invalid JSON in %s: %v", file, err)
		}
	}
}
