package swarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/steveyegge/gastown/internal/errors"
	"github.com/steveyegge/gastown/internal/rig"
)

// Common errors
var (
	ErrSwarmNotFound = errors.Permanent("swarm.NotFound", nil).
		WithHint("Use 'gt swarm list' to see available swarms")
	ErrSwarmExists = errors.User("swarm.Exists", "swarm already exists").
		WithHint("Use a different swarm ID or check existing swarms with 'gt swarm list'")
	ErrInvalidState = errors.User("swarm.InvalidState", "invalid state transition").
		WithHint("Check swarm status with 'gt swarm status <id>' to see current state")
	ErrNoReadyTasks = errors.Permanent("swarm.NoReadyTasks", nil).
		WithHint("Check epic dependencies with 'bd swarm status <epic-id>' to see blocked tasks")
	ErrBeadsNotFound = errors.System("swarm.BeadsNotFound", nil).
		WithHint("Install beads with: brew install beads")
)

// Manager handles swarm lifecycle operations.
// Manager is stateless - all swarm state is discovered from beads.
type Manager struct {
	rig       *rig.Rig
	beadsDir  string // Path for beads operations (git-synced)
	gitDir    string // Path for git operations (rig root)
}

// NewManager creates a new swarm manager for a rig.
func NewManager(r *rig.Rig) *Manager {
	return &Manager{
		rig:      r,
		beadsDir: r.BeadsPath(), // Use BeadsPath() for git-synced beads operations
		gitDir:   r.Path,        // Use rig root for git operations
	}
}

// LoadSwarm loads swarm state from beads by querying the epic.
// This is the canonical way to get swarm state - no in-memory caching.
func (m *Manager) LoadSwarm(epicID string) (*Swarm, error) {
	// Query beads for the epic with retry logic
	var epic struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		Status    string `json:"status"`
		MolType   string `json:"mol_type"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}

	err := errors.RetryWithContext(context.Background(), func() error {
		cmd := exec.Command("bd", "show", epicID, "--json")
		cmd.Dir = m.beadsDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			stderrStr := strings.TrimSpace(stderr.String())
			// Check for permanent errors
			if strings.Contains(stderrStr, "not found") || strings.Contains(stderrStr, "command not found") {
				return errors.Permanent("swarm.BeadsShowFailed", err).
					WithContext("epic_id", epicID).
					WithContext("stderr", stderrStr).
					WithHint("Verify the epic exists with: bd show " + epicID)
			}
			// Transient error - will retry
			return errors.Transient("swarm.BeadsShowTimeout", err).
				WithContext("epic_id", epicID).
				WithContext("stderr", stderrStr)
		}

		// Parse the epic
		if err := json.Unmarshal(stdout.Bytes(), &epic); err != nil {
			return errors.Permanent("swarm.ParseEpicFailed", err).
				WithContext("epic_id", epicID).
				WithHint("Epic data may be corrupted. Try: bd show " + epicID)
		}

		return nil
	}, errors.DefaultRetryConfig())

	if err != nil {
		return nil, err
	}

	// Verify it's a swarm molecule
	if epic.MolType != "swarm" {
		return nil, errors.User("swarm.NotASwarm", "epic is not a swarm").
			WithContext("epic_id", epicID).
			WithContext("mol_type", epic.MolType).
			WithHint("Check epic type with: bd show " + epicID)
	}

	// Get current git commit as base
	baseCommit, _ := m.getGitHead()
	if baseCommit == "" {
		baseCommit = "unknown"
	}

	// Map status to swarm state
	state := SwarmActive
	if epic.Status == "closed" {
		state = SwarmLanded
	}

	swarm := &Swarm{
		ID:           epicID,
		RigName:      m.rig.Name,
		EpicID:       epicID,
		BaseCommit:   baseCommit,
		Integration:  fmt.Sprintf("swarm/%s", epicID),
		TargetBranch: m.rig.DefaultBranch(),
		State:        state,
		Workers:      []string{}, // Discovered from active tasks
		Tasks:        []SwarmTask{},
	}

	// Load tasks from beads (children of the epic)
	tasks, err := m.loadTasksFromBeads(epicID)
	if err == nil {
		swarm.Tasks = tasks
		// Discover workers from assigned tasks
		for _, task := range tasks {
			if task.Assignee != "" {
				swarm.Workers = appendUnique(swarm.Workers, task.Assignee)
			}
		}
	}

	return swarm, nil
}

// appendUnique appends s to slice if not already present.
func appendUnique(slice []string, s string) []string {
	for _, v := range slice {
		if v == s {
			return slice
		}
	}
	return append(slice, s)
}

// GetSwarm loads a swarm from beads. Alias for LoadSwarm for compatibility.
func (m *Manager) GetSwarm(id string) (*Swarm, error) {
	return m.LoadSwarm(id)
}

// GetReadyTasks returns tasks ready to be assigned by querying beads.
func (m *Manager) GetReadyTasks(swarmID string) ([]SwarmTask, error) {
	var status struct {
		Ready []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"ready"`
	}

	err := errors.RetryWithContext(context.Background(), func() error {
		cmd := exec.Command("bd", "swarm", "status", swarmID, "--json")
		cmd.Dir = m.beadsDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			stderrStr := strings.TrimSpace(stderr.String())
			if strings.Contains(stderrStr, "not found") {
				return errors.Permanent("swarm.StatusNotFound", err).
					WithContext("swarm_id", swarmID).
					WithHint("Use 'gt swarm list' to see available swarms")
			}
			return errors.Transient("swarm.StatusQueryFailed", err).
				WithContext("swarm_id", swarmID).
				WithContext("stderr", stderrStr)
		}

		if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
			return errors.Permanent("swarm.ParseStatusFailed", err).
				WithContext("swarm_id", swarmID).
				WithHint("Try running: bd swarm status " + swarmID)
		}

		return nil
	}, errors.DefaultRetryConfig())

	if err != nil {
		return nil, err
	}

	if len(status.Ready) == 0 {
		return nil, ErrNoReadyTasks
	}

	tasks := make([]SwarmTask, len(status.Ready))
	for i, r := range status.Ready {
		tasks[i] = SwarmTask{
			IssueID: r.ID,
			Title:   r.Title,
			State:   TaskPending,
		}
	}
	return tasks, nil
}

// IsComplete checks if all tasks are closed by querying beads.
func (m *Manager) IsComplete(swarmID string) (bool, error) {
	var status struct {
		Ready   []struct{ ID string } `json:"ready"`
		Active  []struct{ ID string } `json:"active"`
		Blocked []struct{ ID string } `json:"blocked"`
	}

	err := errors.RetryWithContext(context.Background(), func() error {
		cmd := exec.Command("bd", "swarm", "status", swarmID, "--json")
		cmd.Dir = m.beadsDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			stderrStr := strings.TrimSpace(stderr.String())
			if strings.Contains(stderrStr, "not found") {
				return errors.Permanent("swarm.IsCompleteNotFound", err).
					WithContext("swarm_id", swarmID).
					WithHint("Use 'gt swarm list' to see available swarms")
			}
			return errors.Transient("swarm.IsCompleteQueryFailed", err).
				WithContext("swarm_id", swarmID).
				WithContext("stderr", stderrStr)
		}

		if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
			return errors.Permanent("swarm.ParseCompletionStatusFailed", err).
				WithContext("swarm_id", swarmID).
				WithHint("Try running: bd swarm status " + swarmID)
		}

		return nil
	}, errors.DefaultRetryConfig())

	if err != nil {
		return false, err
	}

	// Complete if nothing is ready, active, or blocked
	return len(status.Ready) == 0 && len(status.Active) == 0 && len(status.Blocked) == 0, nil
}

// isValidTransition checks if a state transition is allowed.
func isValidTransition(from, to SwarmState) bool {
	transitions := map[SwarmState][]SwarmState{
		SwarmCreated:  {SwarmActive, SwarmCanceled},
		SwarmActive:   {SwarmMerging, SwarmFailed, SwarmCanceled},
		SwarmMerging:  {SwarmLanded, SwarmFailed, SwarmCanceled},
		SwarmLanded:   {}, // Terminal
		SwarmFailed:   {}, // Terminal
		SwarmCanceled: {}, // Terminal
	}

	allowed, ok := transitions[from]
	if !ok {
		return false
	}

	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// loadTasksFromBeads loads child issues from beads CLI.
func (m *Manager) loadTasksFromBeads(epicID string) ([]SwarmTask, error) {
	var issues []struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		Status     string `json:"status"`
		Dependents []struct {
			ID             string `json:"id"`
			Title          string `json:"title"`
			Status         string `json:"status"`
			Assignee       string `json:"assignee"`
			DependencyType string `json:"dependency_type"`
		} `json:"dependents"`
	}

	err := errors.RetryWithContext(context.Background(), func() error {
		cmd := exec.Command("bd", "show", epicID, "--json")
		cmd.Dir = m.beadsDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			stderrStr := strings.TrimSpace(stderr.String())
			// Permanent error if not found
			if strings.Contains(stderrStr, "not found") {
				return errors.Permanent("swarm.LoadTasksNotFound", err).
					WithContext("epic_id", epicID).
					WithHint("Verify the epic exists with: bd show " + epicID)
			}
			// Transient error - will retry
			return errors.Transient("swarm.LoadTasksTimeout", err).
				WithContext("epic_id", epicID).
				WithContext("stderr", stderrStr)
		}

		if err := json.Unmarshal(stdout.Bytes(), &issues); err != nil {
			return errors.Permanent("swarm.ParseTasksFailed", err).
				WithContext("epic_id", epicID).
				WithHint("Epic data may be corrupted. Try: bd show " + epicID)
		}

		return nil
	}, errors.DefaultRetryConfig())

	if err != nil {
		return nil, err
	}

	if len(issues) == 0 {
		return nil, errors.Permanent("swarm.EpicNotFound", nil).
			WithContext("epic_id", epicID).
			WithHint("Verify the epic exists with: bd show " + epicID)
	}

	// Extract dependents as tasks (issues that depend on/are blocked by this epic)
	// Accept both "parent-child" and "blocks" relationships
	var tasks []SwarmTask
	for _, dep := range issues[0].Dependents {
		if dep.DependencyType != "parent-child" && dep.DependencyType != "blocks" {
			continue
		}

		state := TaskPending
		switch dep.Status {
		case "in_progress", "hooked":
			state = TaskInProgress
		case "closed":
			state = TaskMerged
		}

		tasks = append(tasks, SwarmTask{
			IssueID:  dep.ID,
			Title:    dep.Title,
			State:    state,
			Assignee: dep.Assignee,
		})
	}

	return tasks, nil
}

// getGitHead returns the current HEAD commit.
func (m *Manager) getGitHead() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = m.gitDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}
