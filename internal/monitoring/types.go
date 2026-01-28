// Package monitoring provides agent status tracking and activity inference.
// Monitors agent activity and infers status from pane output patterns.
package monitoring

import (
	"time"
)

// AgentStatus represents the current operational state of an agent.
type AgentStatus string

const (
	StatusAvailable AgentStatus = "available" // Ready for work
	StatusWorking   AgentStatus = "working"   // Actively working
	StatusThinking  AgentStatus = "thinking"  // Processing/analyzing
	StatusBlocked   AgentStatus = "blocked"   // Waiting for external dependency
	StatusWaiting   AgentStatus = "waiting"   // Waiting for user input
	StatusReviewing AgentStatus = "reviewing" // Reviewing changes
	StatusIdle      AgentStatus = "idle"      // No recent activity
	StatusPaused    AgentStatus = "paused"    // Manually paused
	StatusError     AgentStatus = "error"     // Error state
	StatusOffline   AgentStatus = "offline"   // Not running
)

// AllStatuses returns all defined agent statuses.
func AllStatuses() []AgentStatus {
	return []AgentStatus{
		StatusAvailable,
		StatusWorking,
		StatusThinking,
		StatusBlocked,
		StatusWaiting,
		StatusReviewing,
		StatusIdle,
		StatusPaused,
		StatusError,
		StatusOffline,
	}
}

// String returns the string representation of the status.
func (s AgentStatus) String() string {
	return string(s)
}

// StatusSource indicates how a status was determined.
// Sources are prioritized: boss > self > inferred
type StatusSource string

const (
	SourceBossOverride StatusSource = "boss"     // Set by witness/mayor (highest priority)
	SourceSelfReported StatusSource = "self"     // Agent reported own status
	SourceInferred     StatusSource = "inferred" // Detected from activity (lowest priority)
)

// Priority returns the priority level of the source (higher = more authoritative).
func (s StatusSource) Priority() int {
	switch s {
	case SourceBossOverride:
		return 3
	case SourceSelfReported:
		return 2
	case SourceInferred:
		return 1
	default:
		return 0
	}
}

// StatusReport represents a status observation for an agent.
type StatusReport struct {
	// AgentID is the agent identifier (e.g., "duneagent/polecats/rust")
	AgentID string `json:"agent_id"`

	// Status is the current status
	Status AgentStatus `json:"status"`

	// Source indicates how the status was determined
	Source StatusSource `json:"source"`

	// Timestamp when this status was observed
	Timestamp time.Time `json:"timestamp"`

	// Message provides additional context (optional)
	Message string `json:"message,omitempty"`

	// Metadata contains additional information about the status
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ActivityPattern defines a pattern to match against agent output.
type ActivityPattern struct {
	// Pattern is the string or regex to match
	Pattern string

	// Status is the inferred status when this pattern matches
	Status AgentStatus

	// Priority controls which pattern wins if multiple match (higher wins)
	Priority int

	// IsRegex indicates if Pattern should be treated as a regex
	IsRegex bool

	// Description explains what this pattern detects
	Description string
}

// IdleConfig configures idle detection behavior.
type IdleConfig struct {
	// Timeout is how long without activity before marking as idle
	Timeout time.Duration

	// CheckInterval is how often to check for idle agents
	CheckInterval time.Duration

	// Enabled controls whether idle detection is active
	Enabled bool
}

// DefaultIdleConfig returns the default idle detection configuration.
func DefaultIdleConfig() IdleConfig {
	return IdleConfig{
		Timeout:       5 * time.Minute,
		CheckInterval: 30 * time.Second,
		Enabled:       true,
	}
}

// AgentActivity tracks activity information for an agent.
type AgentActivity struct {
	// AgentID is the agent identifier
	AgentID string

	// LastActivity is when the agent last had activity
	LastActivity time.Time

	// LastOutput is the most recent output line
	LastOutput string

	// CurrentStatus is the current detected status
	CurrentStatus AgentStatus

	// CurrentSource is how the current status was determined
	CurrentSource StatusSource

	// StatusHistory maintains recent status changes
	StatusHistory []StatusReport

	// IdleSince indicates when the agent became idle (nil if not idle)
	IdleSince *time.Time
}

// IsIdle returns true if the agent is currently considered idle.
func (a *AgentActivity) IsIdle(timeout time.Duration) bool {
	if a.LastActivity.IsZero() {
		return true
	}
	return time.Since(a.LastActivity) > timeout
}

// UpdateActivity records new activity for the agent.
func (a *AgentActivity) UpdateActivity(output string, status AgentStatus, source StatusSource) {
	a.LastActivity = time.Now()
	a.LastOutput = output

	// Only update status if new source has higher or equal priority
	if source.Priority() >= a.CurrentSource.Priority() {
		a.CurrentStatus = status
		a.CurrentSource = source

		// Add to history
		report := StatusReport{
			AgentID:   a.AgentID,
			Status:    status,
			Source:    source,
			Timestamp: time.Now(),
		}
		a.StatusHistory = append(a.StatusHistory, report)

		// Keep only last 50 status changes
		if len(a.StatusHistory) > 50 {
			a.StatusHistory = a.StatusHistory[len(a.StatusHistory)-50:]
		}
	}

	// Clear idle state
	a.IdleSince = nil
}

// MarkIdle marks the agent as idle.
func (a *AgentActivity) MarkIdle() {
	if a.IdleSince == nil {
		now := time.Now()
		a.IdleSince = &now
		a.CurrentStatus = StatusIdle
		a.CurrentSource = SourceInferred
	}
}
