package monitoring

import (
	"sync"
	"time"
)

// StatusTracker maintains status information for multiple agents.
type StatusTracker struct {
	mu         sync.RWMutex
	agents     map[string]*AgentActivity
	detector   *Detector
	idleConfig IdleConfig
}

// NewStatusTracker creates a new status tracker.
func NewStatusTracker() *StatusTracker {
	return &StatusTracker{
		agents:     make(map[string]*AgentActivity),
		detector:   NewDetector(),
		idleConfig: DefaultIdleConfig(),
	}
}

// NewStatusTrackerWithConfig creates a tracker with custom idle configuration.
func NewStatusTrackerWithConfig(idleConfig IdleConfig) *StatusTracker {
	return &StatusTracker{
		agents:     make(map[string]*AgentActivity),
		detector:   NewDetector(),
		idleConfig: idleConfig,
	}
}

// UpdateFromOutput processes agent output and updates status.
// This is the primary method for tracking agent activity.
func (t *StatusTracker) UpdateFromOutput(agentID string, output string) StatusReport {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get or create agent activity
	activity, exists := t.agents[agentID]
	if !exists {
		activity = &AgentActivity{
			AgentID:       agentID,
			StatusHistory: make([]StatusReport, 0),
		}
		t.agents[agentID] = activity
	}

	// Detect status from output
	report := t.detector.Detect(agentID, output)

	// Update activity
	activity.UpdateActivity(output, report.Status, report.Source)

	return report
}

// SetStatus manually sets an agent's status (used for boss overrides or self-reports).
func (t *StatusTracker) SetStatus(agentID string, status AgentStatus, source StatusSource, message string) StatusReport {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Get or create agent activity
	activity, exists := t.agents[agentID]
	if !exists {
		activity = &AgentActivity{
			AgentID:       agentID,
			StatusHistory: make([]StatusReport, 0),
		}
		t.agents[agentID] = activity
	}

	// Create report
	report := StatusReport{
		AgentID:   agentID,
		Status:    status,
		Source:    source,
		Timestamp: time.Now(),
		Message:   message,
	}

	// Update activity
	activity.UpdateActivity(message, status, source)

	return report
}

// GetStatus returns the current status for an agent.
func (t *StatusTracker) GetStatus(agentID string) (AgentStatus, StatusSource, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activity, exists := t.agents[agentID]
	if !exists {
		return StatusOffline, SourceInferred, false
	}

	// Check if agent should be marked as idle
	if t.idleConfig.Enabled && activity.IsIdle(t.idleConfig.Timeout) && activity.CurrentStatus != StatusIdle {
		return StatusIdle, SourceInferred, true
	}

	return activity.CurrentStatus, activity.CurrentSource, true
}

// GetActivity returns full activity information for an agent.
func (t *StatusTracker) GetActivity(agentID string) (*AgentActivity, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activity, exists := t.agents[agentID]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modification
	activityCopy := *activity
	activityCopy.StatusHistory = make([]StatusReport, len(activity.StatusHistory))
	copy(activityCopy.StatusHistory, activity.StatusHistory)

	return &activityCopy, true
}

// ListAgents returns all tracked agent IDs.
func (t *StatusTracker) ListAgents() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	agents := make([]string, 0, len(t.agents))
	for agentID := range t.agents {
		agents = append(agents, agentID)
	}
	return agents
}

// GetAllStatuses returns status for all tracked agents.
func (t *StatusTracker) GetAllStatuses() map[string]StatusReport {
	t.mu.RLock()
	defer t.mu.RUnlock()

	statuses := make(map[string]StatusReport)
	for agentID, activity := range t.agents {
		// Check idle
		status := activity.CurrentStatus
		source := activity.CurrentSource
		if t.idleConfig.Enabled && activity.IsIdle(t.idleConfig.Timeout) && status != StatusIdle {
			status = StatusIdle
			source = SourceInferred
		}

		statuses[agentID] = StatusReport{
			AgentID:   agentID,
			Status:    status,
			Source:    source,
			Timestamp: activity.LastActivity,
			Message:   activity.LastOutput,
		}
	}

	return statuses
}

// RemoveAgent removes an agent from tracking.
func (t *StatusTracker) RemoveAgent(agentID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.agents, agentID)
}

// CheckIdle checks all agents for idle state and updates accordingly.
// Returns a list of agents that transitioned to idle.
func (t *StatusTracker) CheckIdle() []string {
	if !t.idleConfig.Enabled {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	idleAgents := make([]string, 0)

	for agentID, activity := range t.agents {
		if activity.IsIdle(t.idleConfig.Timeout) && activity.CurrentStatus != StatusIdle {
			activity.MarkIdle()
			idleAgents = append(idleAgents, agentID)
		}
	}

	return idleAgents
}

// GetStatusHistory returns the status history for an agent.
func (t *StatusTracker) GetStatusHistory(agentID string, limit int) []StatusReport {
	t.mu.RLock()
	defer t.mu.RUnlock()

	activity, exists := t.agents[agentID]
	if !exists {
		return nil
	}

	history := activity.StatusHistory
	if limit > 0 && len(history) > limit {
		history = history[len(history)-limit:]
	}

	// Return a copy
	result := make([]StatusReport, len(history))
	copy(result, history)
	return result
}

// SetIdleConfig updates the idle detection configuration.
func (t *StatusTracker) SetIdleConfig(config IdleConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.idleConfig = config
}

// GetIdleConfig returns the current idle detection configuration.
func (t *StatusTracker) GetIdleConfig() IdleConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.idleConfig
}

// Detector returns the underlying detector (for adding custom patterns).
func (t *StatusTracker) Detector() *Detector {
	return t.detector
}

// Clear removes all tracked agents.
func (t *StatusTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.agents = make(map[string]*AgentActivity)
}

// AgentCount returns the number of tracked agents.
func (t *StatusTracker) AgentCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.agents)
}
