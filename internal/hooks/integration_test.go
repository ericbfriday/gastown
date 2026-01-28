package hooks_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/gastown/internal/hooks"
)

// TestHookIntegration verifies that hooks can be loaded and fired.
func TestHookIntegration(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Create a simple hooks configuration
	configDir := filepath.Join(tmpDir, ".gastown")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, "hooks.json")
	configContent := `{
  "hooks": {
    "pre-session-start": [
      {
        "type": "builtin",
        "name": "verify-git-clean"
      }
    ],
    "mail-received": [
      {
        "type": "command",
        "cmd": "true"
      }
    ]
  }
}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create hook runner
	runner, err := hooks.NewHookRunner(tmpDir)
	if err != nil {
		t.Fatalf("NewHookRunner failed: %v", err)
	}

	// Verify config was loaded
	if runner.ConfigPath() != configPath {
		t.Errorf("Expected config path %s, got %s", configPath, runner.ConfigPath())
	}

	// Test listing hooks
	hookMap := runner.ListHooks("")
	if len(hookMap) != 2 {
		t.Errorf("Expected 2 events with hooks, got %d", len(hookMap))
	}

	// Test firing pre-session-start hooks (should pass since we're not in a git repo)
	ctx := &hooks.HookContext{
		WorkingDir: tmpDir,
		Metadata: map[string]interface{}{
			"polecat": "test",
			"rig":     "testrig",
		},
	}

	results := runner.Fire(hooks.EventPreSessionStart, ctx)
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
	if len(results) > 0 && !results[0].Success {
		t.Errorf("Expected hook to succeed (not a git repo), got error: %s", results[0].Error)
	}

	// Test firing mail-received hooks
	mailCtx := &hooks.HookContext{
		WorkingDir: tmpDir,
		Metadata: map[string]interface{}{
			"from":    "mayor",
			"to":      "testrig/polecat",
			"subject": "Test message",
		},
	}

	mailResults := runner.Fire(hooks.EventMailReceived, mailCtx)
	if len(mailResults) != 1 {
		t.Errorf("Expected 1 mail hook result, got %d", len(mailResults))
	}
	if len(mailResults) > 0 && !mailResults[0].Success {
		t.Errorf("Mail hook failed: %s", mailResults[0].Error)
	}
}

// TestAllEvents verifies all event constants are defined.
func TestAllEvents(t *testing.T) {
	events := hooks.AllEvents()
	expectedCount := 8
	if len(events) != expectedCount {
		t.Errorf("Expected %d events, got %d", expectedCount, len(events))
	}

	// Verify specific events exist
	expectedEvents := map[hooks.Event]bool{
		hooks.EventPreSessionStart:  true,
		hooks.EventPostSessionStart: true,
		hooks.EventPreShutdown:      true,
		hooks.EventPostShutdown:     true,
		hooks.EventOnPaneOutput:     true,
		hooks.EventSessionIdle:      true,
		hooks.EventMailReceived:     true,
		hooks.EventWorkAssigned:     true,
	}

	for _, event := range events {
		if !expectedEvents[event] {
			t.Errorf("Unexpected event: %s", event)
		}
	}
}
