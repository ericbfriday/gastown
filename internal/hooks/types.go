// Package hooks provides event-based extensibility for Gas Town.
// Hooks allow external scripts and internal functions to respond to
// lifecycle events such as session start/stop, mail received, etc.
package hooks

import (
	"encoding/json"
	"time"
)

// Event represents a lifecycle event that can trigger hooks.
type Event string

const (
	EventPreSessionStart  Event = "pre-session-start"
	EventPostSessionStart Event = "post-session-start"
	EventPreShutdown      Event = "pre-shutdown"
	EventPostShutdown     Event = "post-shutdown"
	EventOnPaneOutput     Event = "on-pane-output"
	EventSessionIdle      Event = "session-idle"
	EventMailReceived     Event = "mail-received"
	EventWorkAssigned     Event = "work-assigned"
)

// AllEvents returns a slice of all defined events.
func AllEvents() []Event {
	return []Event{
		EventPreSessionStart,
		EventPostSessionStart,
		EventPreShutdown,
		EventPostShutdown,
		EventOnPaneOutput,
		EventSessionIdle,
		EventMailReceived,
		EventWorkAssigned,
	}
}

// HookType represents the type of hook execution.
type HookType string

const (
	HookTypeCommand HookType = "command" // Execute external script
	HookTypeBuiltin HookType = "builtin" // Internal Go function
)

// HookDefinition defines a single hook to be executed for an event.
type HookDefinition struct {
	Type HookType `json:"type"`           // "command" or "builtin"
	Cmd  string   `json:"cmd,omitempty"`  // Command to execute (for type=command)
	Name string   `json:"name,omitempty"` // Built-in function name (for type=builtin)
	Args []string `json:"args,omitempty"` // Optional arguments
}

// HookConfig represents the hook configuration loaded from JSON.
// File: .claude/hooks.json or .gastown/hooks.json
type HookConfig struct {
	Hooks map[Event][]HookDefinition `json:"hooks"`
}

// HookContext provides context information to hooks during execution.
type HookContext struct {
	Event      Event                  `json:"event"`
	Timestamp  time.Time              `json:"timestamp"`
	WorkingDir string                 `json:"working_dir"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// HookResult represents the result of executing a hook.
type HookResult struct {
	Success bool   `json:"success"`           // Whether hook executed successfully
	Message string `json:"message,omitempty"` // Optional message from hook
	Block   bool   `json:"block,omitempty"`   // For pre-* hooks: should operation be blocked?
	Output  string `json:"output,omitempty"`  // Output from command hooks
	Error   string `json:"error,omitempty"`   // Error message if any

	// Execution metadata
	Duration time.Duration `json:"duration,omitempty"`
	ExitCode int           `json:"exit_code,omitempty"` // For command hooks
}

// BuiltinHookFunc is the signature for built-in hook functions.
type BuiltinHookFunc func(ctx *HookContext) (*HookResult, error)

// MarshalJSON implements custom JSON marshaling for HookResult to handle Duration.
func (r *HookResult) MarshalJSON() ([]byte, error) {
	type Alias HookResult
	return json.Marshal(&struct {
		Duration string `json:"duration,omitempty"`
		*Alias
	}{
		Duration: r.Duration.String(),
		Alias:    (*Alias)(r),
	})
}
