package swarm

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/steveyegge/gastown/internal/errors"
)

// Integration branch errors
var (
	ErrBranchExists = errors.User("swarm.BranchExists", "branch already exists").
		WithHint("Use a different branch name or delete the existing branch first")
	ErrBranchNotFound = errors.Permanent("swarm.BranchNotFound", nil).
		WithHint("Check available branches with: git branch -a")
	ErrNotOnIntegration = errors.User("swarm.NotOnIntegration", "not on integration branch").
		WithHint("Switch to integration branch with: git checkout <integration-branch>")
)

// SwarmGitError contains raw output from a git command for observation.
// ZFC: Callers observe the raw output and decide what to do.
type SwarmGitError struct {
	Command string
	Stdout  string
	Stderr  string
	Err     error
}

func (e *SwarmGitError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("%s: %s", e.Command, e.Stderr)
	}
	return fmt.Sprintf("%s: %v", e.Command, e.Err)
}

// CreateIntegrationBranch creates the integration branch for a swarm.
// The branch is created from the swarm's BaseCommit and pushed to origin.
func (m *Manager) CreateIntegrationBranch(swarmID string) error {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return err
	}

	branchName := swarm.Integration

	// Check if branch already exists
	if m.branchExists(branchName) {
		return ErrBranchExists
	}

	// Create branch from BaseCommit
	if err := m.gitRun("checkout", "-b", branchName, swarm.BaseCommit); err != nil {
		return errors.System("swarm.CreateBranchFailed", err).
			WithContext("branch", branchName).
			WithContext("base_commit", swarm.BaseCommit).
			WithHint("Check git status with: git status")
	}

	// Push to origin with network retry (non-fatal: may not have remote)
	_ = errors.Retry(func() error {
		return m.gitRun("push", "-u", "origin", branchName)
	}, errors.NetworkRetryConfig())

	return nil
}

// MergeToIntegration merges a worker branch into the integration branch.
// Returns ErrMergeConflict if the merge has conflicts.
func (m *Manager) MergeToIntegration(swarmID, workerBranch string) error {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return err
	}

	// Ensure we're on the integration branch
	currentBranch, err := m.getCurrentBranch()
	if err != nil {
		return err
	}
	if currentBranch != swarm.Integration {
		if err := m.gitRun("checkout", swarm.Integration); err != nil {
			return errors.System("swarm.CheckoutIntegrationFailed", err).
				WithContext("integration_branch", swarm.Integration).
				WithHint("Check git status with: git status")
		}
	}

	// Fetch the worker branch with network retry (non-fatal: may not exist on remote, try local)
	_ = errors.Retry(func() error {
		return m.gitRun("fetch", "origin", workerBranch)
	}, errors.NetworkRetryConfig())

	// Attempt merge
	err = m.gitRun("merge", "--no-ff", "-m",
		fmt.Sprintf("Merge %s into %s", workerBranch, swarm.Integration),
		workerBranch)
	if err != nil {
		// ZFC: Use git's porcelain output to detect conflicts instead of parsing stderr.
		conflicts, conflictErr := m.getConflictingFiles()
		if conflictErr == nil && len(conflicts) > 0 {
			// Return enhanced error with conflict details
			return errors.User("swarm.MergeConflict", "merge conflict").
				WithContext("worker_branch", workerBranch).
				WithContext("integration_branch", swarm.Integration).
				WithContext("conflicting_files", strings.Join(conflicts, ", ")).
				WithHint(fmt.Sprintf("Resolve conflicts in: %s\nThen run: git add . && git commit", strings.Join(conflicts, ", ")))
		}
		return errors.System("swarm.MergeFailed", err).
			WithContext("worker_branch", workerBranch).
			WithContext("integration_branch", swarm.Integration).
			WithHint("Check git status with: git status")
	}

	return nil
}

// AbortMerge aborts an in-progress merge.
func (m *Manager) AbortMerge() error {
	return m.gitRun("merge", "--abort")
}

// LandToMain merges the integration branch to the target branch (usually main).
func (m *Manager) LandToMain(swarmID string) error {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return err
	}

	// Checkout target branch
	if err := m.gitRun("checkout", swarm.TargetBranch); err != nil {
		return errors.System("swarm.CheckoutTargetFailed", err).
			WithContext("target_branch", swarm.TargetBranch).
			WithHint("Check git status with: git status")
	}

	// Pull latest with network retry (non-fatal: may fail if remote unreachable)
	_ = errors.Retry(func() error {
		return m.gitRun("pull", "origin", swarm.TargetBranch)
	}, errors.NetworkRetryConfig())

	// Merge integration branch
	err = m.gitRun("merge", "--no-ff", "-m",
		fmt.Sprintf("Land swarm %s", swarmID),
		swarm.Integration)
	if err != nil {
		// ZFC: Use git's porcelain output to detect conflicts instead of parsing stderr.
		conflicts, conflictErr := m.getConflictingFiles()
		if conflictErr == nil && len(conflicts) > 0 {
			// Return enhanced error with conflict details
			return errors.User("swarm.LandConflict", "merge conflict during landing").
				WithContext("swarm_id", swarmID).
				WithContext("target_branch", swarm.TargetBranch).
				WithContext("integration_branch", swarm.Integration).
				WithContext("conflicting_files", strings.Join(conflicts, ", ")).
				WithHint(fmt.Sprintf("Resolve conflicts in: %s\nThen run: git add . && git commit", strings.Join(conflicts, ", ")))
		}
		return errors.System("swarm.LandMergeFailed", err).
			WithContext("swarm_id", swarmID).
			WithContext("target_branch", swarm.TargetBranch).
			WithHint("Check git status with: git status")
	}

	// Push with network retry
	if err := errors.Retry(func() error {
		return m.gitRun("push", "origin", swarm.TargetBranch)
	}, errors.NetworkRetryConfig()); err != nil {
		return errors.Transient("swarm.PushFailed", err).
			WithContext("target_branch", swarm.TargetBranch).
			WithHint("Check network connection and retry with: git push origin " + swarm.TargetBranch)
	}

	return nil
}

// CleanupBranches removes all branches associated with a swarm.
func (m *Manager) CleanupBranches(swarmID string) error {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return err
	}

	var lastErr error

	// Delete integration branch locally
	if err := m.gitRun("branch", "-D", swarm.Integration); err != nil {
		lastErr = err
	}

	// Delete integration branch remotely with network retry (best-effort cleanup)
	_ = errors.Retry(func() error {
		return m.gitRun("push", "origin", "--delete", swarm.Integration)
	}, errors.NetworkRetryConfig())

	// Delete worker branches with network retry (best-effort cleanup)
	for _, task := range swarm.Tasks {
		if task.Branch != "" {
			// Local delete
			_ = m.gitRun("branch", "-D", task.Branch)
			// Remote delete with network retry
			_ = errors.Retry(func() error {
				return m.gitRun("push", "origin", "--delete", task.Branch)
			}, errors.NetworkRetryConfig())
		}
	}

	return lastErr
}

// GetIntegrationBranch returns the integration branch name for a swarm.
func (m *Manager) GetIntegrationBranch(swarmID string) (string, error) {
	swarm, err := m.LoadSwarm(swarmID)
	if err != nil {
		return "", err
	}
	return swarm.Integration, nil
}

// GetWorkerBranch generates the branch name for a worker on a task.
func (m *Manager) GetWorkerBranch(swarmID, worker, taskID string) string {
	return fmt.Sprintf("%s/%s/%s", swarmID, worker, taskID)
}

// branchExists checks if a branch exists locally.
func (m *Manager) branchExists(branch string) bool {
	err := m.gitRun("show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return err == nil
}

// getCurrentBranch returns the current branch name.
func (m *Manager) getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = m.gitDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", errors.System("swarm.GetCurrentBranchFailed", err).
			WithContext("dir", m.gitDir).
			WithContext("stderr", strings.TrimSpace(stderr.String())).
			WithHint("Check git status with: git status")
	}

	return strings.TrimSpace(stdout.String()), nil
}

// getConflictingFiles returns the list of files with merge conflicts.
// ZFC: Uses git's porcelain output (diff --diff-filter=U) instead of parsing stderr.
func (m *Manager) getConflictingFiles() ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = m.gitDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, errors.System("swarm.GetConflictsFailed", err).
			WithContext("dir", m.gitDir).
			WithContext("stderr", strings.TrimSpace(stderr.String())).
			WithHint("Check git status with: git status")
	}

	out := strings.TrimSpace(stdout.String())
	if out == "" {
		return nil, nil
	}

	files := strings.Split(out, "\n")
	var result []string
	for _, f := range files {
		if f != "" {
			result = append(result, f)
		}
	}
	return result, nil
}

// gitRun executes a git command.
// ZFC: Returns SwarmGitError with raw output for agent observation.
func (m *Manager) gitRun(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = m.gitDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Determine command name
		command := ""
		for _, arg := range args {
			if !strings.HasPrefix(arg, "-") {
				command = arg
				break
			}
		}
		if command == "" && len(args) > 0 {
			command = args[0]
		}

		return &SwarmGitError{
			Command: command,
			Stdout:  strings.TrimSpace(stdout.String()),
			Stderr:  strings.TrimSpace(stderr.String()),
			Err:     err,
		}
	}

	return nil
}
