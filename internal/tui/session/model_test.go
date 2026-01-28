package session

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTransitionPhases(t *testing.T) {
	tests := []struct {
		name          string
		phase         TransitionPhase
		wantMessage   bool
		wantSpinner   bool
		wantCompleted bool
	}{
		{
			name:          "idle phase",
			phase:         PhaseIdle,
			wantMessage:   false,
			wantSpinner:   false,
			wantCompleted: false,
		},
		{
			name:          "pre-shutdown phase",
			phase:         PhasePreShutdown,
			wantMessage:   true,
			wantSpinner:   true,
			wantCompleted: false,
		},
		{
			name:          "complete phase",
			phase:         PhaseComplete,
			wantMessage:   true,
			wantSpinner:   false,
			wantCompleted: true,
		},
		{
			name:          "error phase",
			phase:         PhaseError,
			wantMessage:   true,
			wantSpinner:   false,
			wantCompleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := TransitionState{
				Phase:   tt.phase,
				Message: "test message",
			}

			if tt.wantCompleted && state.Phase != PhaseComplete {
				t.Errorf("expected complete phase, got %v", state.Phase)
			}
		})
	}
}

func TestTransitionProgress(t *testing.T) {
	state := TransitionState{
		CurrentStep: 3,
		TotalSteps:  7,
	}

	expectedProgress := float64(3) / float64(7)
	actualProgress := float64(state.CurrentStep) / float64(state.TotalSteps)

	if actualProgress != expectedProgress {
		t.Errorf("expected progress %v, got %v", expectedProgress, actualProgress)
	}
}

func TestContextPreservation(t *testing.T) {
	state := TransitionState{
		PreservedData: make(map[string]interface{}),
	}

	// Add preserved data
	state.PreservedData["last_output"] = "test output"
	state.PreservedData["custom_key"] = "custom_value"

	if len(state.PreservedData) != 2 {
		t.Errorf("expected 2 preserved items, got %d", len(state.PreservedData))
	}

	if state.PreservedData["last_output"] != "test output" {
		t.Errorf("expected preserved output, got %v", state.PreservedData["last_output"])
	}
}

func TestMessageHandling(t *testing.T) {
	m := Model{
		state: TransitionState{
			Phase: PhaseIdle,
		},
	}

	// Test transition step message
	stepMsg := transitionStepMsg{
		nextPhase: PhaseStarting,
		message:   "Starting session",
		preservedData: map[string]interface{}{
			"test": "value",
		},
	}

	updated, _ := m.handleTransitionStep(stepMsg)
	model := updated.(Model)

	if model.state.Phase != PhaseStarting {
		t.Errorf("expected phase %v, got %v", PhaseStarting, model.state.Phase)
	}

	if model.state.Message != "Starting session" {
		t.Errorf("expected message 'Starting session', got %q", model.state.Message)
	}

	if model.state.PreservedData["test"] != "value" {
		t.Errorf("expected preserved data, got %v", model.state.PreservedData)
	}
}

func TestWindowResize(t *testing.T) {
	m := Model{}

	msg := tea.WindowSizeMsg{
		Width:  80,
		Height: 24,
	}

	updated, _ := m.Update(msg)
	model := updated.(tea.Model).(Model)

	if model.width != 80 {
		t.Errorf("expected width 80, got %d", model.width)
	}

	if model.height != 24 {
		t.Errorf("expected height 24, got %d", model.height)
	}
}

func TestPhaseProgression(t *testing.T) {
	phases := []TransitionPhase{
		PhasePreShutdown,
		PhaseShuttingDown,
		PhaseShutdownHook,
		PhasePreStart,
		PhaseStarting,
		PhaseStartupHook,
		PhaseComplete,
	}

	for i, phase := range phases {
		t.Run(string(phase), func(t *testing.T) {
			state := TransitionState{
				Phase:       phase,
				CurrentStep: i,
				TotalSteps:  len(phases),
			}

			progress := float64(state.CurrentStep) / float64(state.TotalSteps)
			if progress < 0 || progress > 1 {
				t.Errorf("invalid progress %v for phase %v", progress, phase)
			}
		})
	}
}

func TestDurationFormatting(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{100 * time.Millisecond, "100ms"},
		{1500 * time.Millisecond, "1.5s"},
		{65 * time.Second, "1m 5s"},
		{125 * time.Second, "2m 5s"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"this is a long string", 10, "this is..."},
		{"newline\nremoval", 20, "newline removal"},
		{"  spaces  ", 20, "spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestErrorState(t *testing.T) {
	m := Model{
		state: TransitionState{
			Phase: PhaseIdle,
		},
	}

	errorMsg := transitionErrorMsg{
		err:     &testError{msg: "test error"},
		message: "operation failed",
	}

	updated, _ := m.Update(errorMsg)
	model := updated.(tea.Model).(Model)

	if !model.HasError() {
		t.Error("expected model to have error")
	}

	if model.GetError() == nil {
		t.Error("expected non-nil error")
	}

	if model.state.Phase != PhaseError {
		t.Errorf("expected error phase, got %v", model.state.Phase)
	}

	if model.state.Message != "operation failed" {
		t.Errorf("expected error message, got %q", model.state.Message)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
