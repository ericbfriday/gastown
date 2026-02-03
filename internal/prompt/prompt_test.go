package prompt

import (
	"os"
	"testing"
	"time"
)

func TestConfirm_WithForce(t *testing.T) {
	result := Confirm("Test message", WithForce(true))
	if !result {
		t.Error("Expected true with force flag")
	}
}

func TestConfirm_WithYes(t *testing.T) {
	result := Confirm("Test message", WithYes(true))
	if !result {
		t.Error("Expected true with yes flag")
	}
}

func TestConfirm_WithEnvVar(t *testing.T) {
	// Set environment variable
	oldVal := os.Getenv("GT_YES")
	os.Setenv("GT_YES", "1")
	defer func() {
		if oldVal == "" {
			os.Unsetenv("GT_YES")
		} else {
			os.Setenv("GT_YES", oldVal)
		}
	}()

	result := Confirm("Test message")
	if !result {
		t.Error("Expected true with GT_YES=1")
	}
}

func TestConfirm_NonInteractive(t *testing.T) {
	// Non-interactive with default false
	result := Confirm("Test message", WithNonInteractive(true), WithDefaultResponse(false))
	if result {
		t.Error("Expected false in non-interactive mode with default false")
	}

	// Non-interactive with default true
	result = Confirm("Test message", WithNonInteractive(true), WithDefaultResponse(true))
	if !result {
		t.Error("Expected true in non-interactive mode with default true")
	}
}

func TestConfirmBatch(t *testing.T) {
	// Test with force
	result := ConfirmBatch("Delete", 5, WithForce(true))
	if !result {
		t.Error("Expected true with force flag")
	}

	// Test with yes
	result = ConfirmBatch("Delete", 5, WithYes(true))
	if !result {
		t.Error("Expected true with yes flag")
	}
}

func TestChoice_NonInteractive(t *testing.T) {
	options := []string{"Option 1", "Option 2", "Option 3"}

	// Non-interactive with default true returns first option
	result := Choice("Select option", options, WithNonInteractive(true), WithDefaultResponse(true))
	if result != 0 {
		t.Errorf("Expected 0, got %d", result)
	}

	// Non-interactive with default false returns -1
	result = Choice("Select option", options, WithNonInteractive(true), WithDefaultResponse(false))
	if result != -1 {
		t.Errorf("Expected -1, got %d", result)
	}
}

func TestTimeout(t *testing.T) {
	// This test would hang in real usage, so we test with non-interactive
	result := Confirm("Test",
		WithNonInteractive(true),
		WithTimeout(100*time.Millisecond),
		WithDefaultResponse(true))

	if !result {
		t.Error("Expected true with default response")
	}
}

func TestGlobalConfig(t *testing.T) {
	// Save original
	original := GlobalConfig

	// Modify global config
	GlobalConfig = Config{Force: true}

	// Test that it applies
	result := Confirm("Test message")
	if !result {
		t.Error("Expected true with global force flag")
	}

	// Restore
	GlobalConfig = original
}

func TestInput_NonInteractive(t *testing.T) {
	validator := func(s string) error {
		if s == "" {
			return os.ErrInvalid
		}
		return nil
	}

	// Non-interactive always fails
	input, ok := Input("Enter value", validator, WithNonInteractive(true))
	if ok || input != "" {
		t.Error("Expected failure in non-interactive mode")
	}
}
