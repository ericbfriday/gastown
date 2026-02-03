package crew

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/steveyegge/gastown/internal/beads"
	"github.com/steveyegge/gastown/internal/claude"
	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/errors"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/session"
	"github.com/steveyegge/gastown/internal/tmux"
	"github.com/steveyegge/gastown/internal/util"
)

// Common errors
var (
	ErrCrewExists = errors.Permanent("crew.Exists", nil).
		WithHint("Use a different worker name or remove the existing worker with 'gt crew rm <name>'")
	ErrCrewNotFound = errors.Permanent("crew.NotFound", nil).
		WithHint("Use 'gt crew list' to see available crew workers")
	ErrHasChanges = errors.User("crew.UncommittedChanges", "worker has uncommitted changes").
		WithHint("Commit or stash changes, or use --force to remove anyway")
	ErrInvalidCrewName = errors.User("crew.InvalidName", "invalid crew worker name").
		WithHint("Worker names must be lowercase, alphanumeric with underscores only")
	ErrSessionRunning = errors.Permanent("crew.SessionRunning", nil).
		WithHint("Stop the session first with 'gt crew stop <name>' or use --restart")
	ErrSessionNotFound = errors.Permanent("crew.SessionNotFound", nil).
		WithHint("Start the session first with 'gt crew start <name>'")
)

// StartOptions configures crew session startup.
type StartOptions struct {
	// Account specifies the account handle to use (overrides default).
	Account string

	// ClaudeConfigDir is resolved CLAUDE_CONFIG_DIR for the account.
	// If set, this is injected as an environment variable.
	ClaudeConfigDir string

	// KillExisting kills any existing session before starting (for restart operations).
	// If false and a session is running, Start() returns ErrSessionRunning.
	KillExisting bool

	// Topic is the startup nudge topic (e.g., "start", "restart", "refresh").
	// Defaults to "start" if empty.
	Topic string

	// Interactive removes --dangerously-skip-permissions for interactive/refresh mode.
	Interactive bool

	// AgentOverride specifies an alternate agent alias (e.g., for testing).
	AgentOverride string
}

// validateCrewName checks that a crew name is safe and valid.
// Rejects path traversal attempts and characters that break agent ID parsing.
func validateCrewName(name string) error {
	if name == "" {
		return errors.User("crew.EmptyName", "worker name cannot be empty").
			WithHint("Provide a valid worker name (lowercase, alphanumeric with underscores)")
	}
	if name == "." || name == ".." {
		return errors.User("crew.InvalidName", "reserved name").
			WithContext("name", name).
			WithHint("Use a descriptive worker name (lowercase, alphanumeric with underscores)")
	}
	if strings.ContainsAny(name, "/\\") {
		return errors.User("crew.PathSeparators", "worker name contains path separators").
			WithContext("name", name).
			WithHint("Worker names cannot contain '/' or '\\' characters")
	}
	if strings.Contains(name, "..") {
		return errors.User("crew.PathTraversal", "worker name contains path traversal sequence").
			WithContext("name", name).
			WithHint("Worker names cannot contain '..' sequence")
	}
	// Reject characters that break agent ID parsing (same as rig names)
	if strings.ContainsAny(name, "-. ") {
		sanitized := strings.NewReplacer("-", "_", ".", "_", " ", "_").Replace(name)
		sanitized = strings.ToLower(sanitized)
		return errors.User("crew.InvalidCharacters", "worker name contains invalid characters").
			WithContext("name", name).
			WithContext("suggested_name", sanitized).
			WithHint(fmt.Sprintf("Hyphens, dots, and spaces are reserved for agent ID parsing. Try %q instead", sanitized))
	}
	return nil
}

// Manager handles crew worker lifecycle.
type Manager struct {
	rig *rig.Rig
	git *git.Git
}

// NewManager creates a new crew manager.
func NewManager(r *rig.Rig, g *git.Git) *Manager {
	return &Manager{
		rig: r,
		git: g,
	}
}

// crewDir returns the directory for a crew worker.
func (m *Manager) crewDir(name string) string {
	return filepath.Join(m.rig.Path, "crew", name)
}

// stateFile returns the state file path for a crew worker.
func (m *Manager) stateFile(name string) string {
	return filepath.Join(m.crewDir(name), "state.json")
}

// mailDir returns the mail directory path for a crew worker.
func (m *Manager) mailDir(name string) string {
	return filepath.Join(m.crewDir(name), "mail")
}

// exists checks if a crew worker exists.
func (m *Manager) exists(name string) bool {
	_, err := os.Stat(m.crewDir(name))
	return err == nil
}

// Add creates a new crew worker with a clone of the rig.
func (m *Manager) Add(name string, createBranch bool) (*CrewWorker, error) {
	if err := validateCrewName(name); err != nil {
		return nil, err
	}
	if m.exists(name) {
		return nil, errors.Permanent("crew.AlreadyExists", nil).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithHint("Use a different worker name or remove the existing worker with 'gt crew rm " + name + "'")
	}

	crewPath := m.crewDir(name)

	// Create crew directory if needed
	crewBaseDir := filepath.Join(m.rig.Path, "crew")
	if err := os.MkdirAll(crewBaseDir, 0755); err != nil {
		return nil, errors.System("crew.CreateBaseDirFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", crewBaseDir).
			WithHint("Check directory permissions and available disk space")
	}

	// Clone the rig repo with retry logic
	cloneErr := errors.RetryWithContext(context.Background(), func() error {
		if m.rig.LocalRepo != "" {
			if err := m.git.CloneWithReference(m.rig.GitURL, crewPath, m.rig.LocalRepo); err != nil {
				fmt.Printf("Warning: could not clone with local repo reference: %v\n", err)
				if err := m.git.Clone(m.rig.GitURL, crewPath); err != nil {
					return errors.Transient("crew.CloneFailed", err).
						WithContext("worker_name", name).
						WithContext("rig_name", m.rig.Name).
						WithContext("git_url", m.rig.GitURL).
						WithContext("path", crewPath)
				}
			}
		} else {
			if err := m.git.Clone(m.rig.GitURL, crewPath); err != nil {
				return errors.Transient("crew.CloneFailed", err).
					WithContext("worker_name", name).
					WithContext("rig_name", m.rig.Name).
					WithContext("git_url", m.rig.GitURL).
					WithContext("path", crewPath)
			}
		}
		return nil
	}, errors.NetworkRetryConfig())

	if cloneErr != nil {
		return nil, errors.Transient("crew.CloneWithRetryFailed", cloneErr).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("git_url", m.rig.GitURL).
			WithHint("Check network connectivity, git credentials, and that the repository URL is correct")
	}

	crewGit := git.NewGit(crewPath)
	branchName := m.rig.DefaultBranch()

	// Optionally create a working branch
	if createBranch {
		branchName = fmt.Sprintf("crew/%s", name)
		if err := crewGit.CreateBranch(branchName); err != nil {
			_ = os.RemoveAll(crewPath) // best-effort cleanup
			return nil, errors.Transient("crew.CreateBranchFailed", err).
				WithContext("worker_name", name).
				WithContext("rig_name", m.rig.Name).
				WithContext("branch", branchName).
				WithHint("Git operations may be failing. Check git installation and repository state")
		}
		if err := crewGit.Checkout(branchName); err != nil {
			_ = os.RemoveAll(crewPath) // best-effort cleanup
			return nil, errors.Transient("crew.CheckoutBranchFailed", err).
				WithContext("worker_name", name).
				WithContext("rig_name", m.rig.Name).
				WithContext("branch", branchName).
				WithHint("Git checkout failed. The branch may not exist or repository may be corrupted")
		}
	}

	// Create mail directory for mail delivery
	mailPath := m.mailDir(name)
	if err := os.MkdirAll(mailPath, 0755); err != nil {
		_ = os.RemoveAll(crewPath) // best-effort cleanup
		return nil, errors.System("crew.CreateMailDirFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", mailPath).
			WithHint("Check directory permissions and available disk space")
	}

	// Set up shared beads: crew uses rig's shared beads via redirect file
	if err := m.setupSharedBeads(crewPath); err != nil {
		// Non-fatal - crew can still work, warn but don't fail
		fmt.Printf("Warning: could not set up shared beads: %v\n", err)
	}

	// Provision PRIME.md with Gas Town context for this worker.
	// This is the fallback if SessionStart hook fails - ensures crew workers
	// always have GUPP and essential Gas Town context.
	if err := beads.ProvisionPrimeMDForWorktree(crewPath); err != nil {
		// Non-fatal - crew can still work via hook, warn but don't fail
		fmt.Printf("Warning: could not provision PRIME.md: %v\n", err)
	}

	// Copy overlay files from .runtime/overlay/ to crew root.
	// This allows services to have .env and other config files at their root.
	if err := rig.CopyOverlay(m.rig.Path, crewPath); err != nil {
		// Non-fatal - log warning but continue
		fmt.Printf("Warning: could not copy overlay files: %v\n", err)
	}

	// Ensure .gitignore has required Gas Town patterns
	if err := rig.EnsureGitignorePatterns(crewPath); err != nil {
		// Non-fatal - log warning but continue
		fmt.Printf("Warning: could not update .gitignore: %v\n", err)
	}

	// NOTE: Slash commands (.claude/commands/) are provisioned at town level by gt install.
	// All agents inherit them via Claude's directory traversal - no per-workspace copies needed.

	// NOTE: We intentionally do NOT write to CLAUDE.md here.
	// Gas Town context is injected ephemerally via SessionStart hook (gt prime).
	// Writing to CLAUDE.md would overwrite project instructions and leak
	// Gas Town internals into the project repo when workers commit/push.

	// Create crew worker state
	now := time.Now()
	crew := &CrewWorker{
		Name:      name,
		Rig:       m.rig.Name,
		ClonePath: crewPath,
		Branch:    branchName,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save state
	if err := m.saveState(crew); err != nil {
		_ = os.RemoveAll(crewPath) // best-effort cleanup
		return nil, errors.System("crew.SaveStateFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", m.stateFile(name)).
			WithHint("Check directory permissions and available disk space")
	}

	return crew, nil
}

// Remove deletes a crew worker.
func (m *Manager) Remove(name string, force bool) error {
	if err := validateCrewName(name); err != nil {
		return err
	}
	if !m.exists(name) {
		return errors.Permanent("crew.NotFound", nil).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithHint("Use 'gt crew list' to see available crew workers")
	}

	crewPath := m.crewDir(name)

	if !force {
		crewGit := git.NewGit(crewPath)
		hasChanges, err := crewGit.HasUncommittedChanges()
		if err == nil && hasChanges {
			return errors.User("crew.UncommittedChanges", "worker has uncommitted changes").
				WithContext("worker_name", name).
				WithContext("rig_name", m.rig.Name).
				WithContext("path", crewPath).
				WithHint("Commit or stash changes with 'cd " + crewPath + " && git add . && git commit', or use --force to remove anyway")
		}
	}

	// Remove directory
	if err := os.RemoveAll(crewPath); err != nil {
		return errors.System("crew.RemoveDirFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", crewPath).
			WithHint("Check directory permissions. You may need to manually delete: " + crewPath)
	}

	return nil
}

// List returns all crew workers in the rig.
func (m *Manager) List() ([]*CrewWorker, error) {
	crewBaseDir := filepath.Join(m.rig.Path, "crew")

	entries, err := os.ReadDir(crewBaseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.System("crew.ReadDirFailed", err).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", crewBaseDir).
			WithHint("Check directory permissions for: " + crewBaseDir)
	}

	var workers []*CrewWorker
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		worker, err := m.Get(entry.Name())
		if err != nil {
			continue // Skip invalid workers
		}
		workers = append(workers, worker)
	}

	return workers, nil
}

// Get returns a specific crew worker by name.
func (m *Manager) Get(name string) (*CrewWorker, error) {
	if err := validateCrewName(name); err != nil {
		return nil, err
	}
	if !m.exists(name) {
		return nil, errors.Permanent("crew.NotFound", nil).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithHint("Use 'gt crew list' to see available crew workers")
	}

	return m.loadState(name)
}

// saveState persists crew worker state to disk using atomic write.
func (m *Manager) saveState(crew *CrewWorker) error {
	stateFile := m.stateFile(crew.Name)
	if err := util.AtomicWriteJSON(stateFile, crew); err != nil {
		return errors.System("crew.WriteStateFailed", err).
			WithContext("worker_name", crew.Name).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", stateFile).
			WithHint("Check directory permissions and available disk space")
	}

	return nil
}

// loadState reads crew worker state from disk.
func (m *Manager) loadState(name string) (*CrewWorker, error) {
	stateFile := m.stateFile(name)

	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Return minimal crew worker if state file missing
			return &CrewWorker{
				Name:      name,
				Rig:       m.rig.Name,
				ClonePath: m.crewDir(name),
			}, nil
		}
		return nil, errors.System("crew.ReadStateFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", stateFile).
			WithHint("Check directory permissions for: " + stateFile)
	}

	var crew CrewWorker
	if err := json.Unmarshal(data, &crew); err != nil {
		return nil, errors.System("crew.ParseStateFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", stateFile).
			WithHint("State file may be corrupted. Try removing and recreating the worker")
	}

	// Directory name is source of truth for Name and ClonePath.
	// state.json can become stale after directory rename, copy, or corruption.
	crew.Name = name
	crew.ClonePath = m.crewDir(name)

	// Rig only needs backfill when empty (less likely to drift)
	if crew.Rig == "" {
		crew.Rig = m.rig.Name
	}

	return &crew, nil
}

// Rename renames a crew worker from oldName to newName.
func (m *Manager) Rename(oldName, newName string) error {
	if !m.exists(oldName) {
		return errors.Permanent("crew.NotFound", nil).
			WithContext("worker_name", oldName).
			WithContext("rig_name", m.rig.Name).
			WithHint("Use 'gt crew list' to see available crew workers")
	}
	if m.exists(newName) {
		return errors.Permanent("crew.AlreadyExists", nil).
			WithContext("old_name", oldName).
			WithContext("new_name", newName).
			WithContext("rig_name", m.rig.Name).
			WithHint("Use a different worker name or remove the existing worker with 'gt crew rm " + newName + "'")
	}

	oldPath := m.crewDir(oldName)
	newPath := m.crewDir(newName)

	// Rename directory
	if err := os.Rename(oldPath, newPath); err != nil {
		return errors.System("crew.RenameDirFailed", err).
			WithContext("old_name", oldName).
			WithContext("new_name", newName).
			WithContext("rig_name", m.rig.Name).
			WithContext("old_path", oldPath).
			WithContext("new_path", newPath).
			WithHint("Check directory permissions and that the worker is not in use")
	}

	// Update state file with new name and path
	crew, err := m.loadState(newName)
	if err != nil {
		// Rollback on error (best-effort)
		_ = os.Rename(newPath, oldPath)
		return errors.System("crew.LoadStateAfterRenameFailed", err).
			WithContext("old_name", oldName).
			WithContext("new_name", newName).
			WithContext("rig_name", m.rig.Name).
			WithHint("Worker was renamed but state could not be loaded. Try manually fixing: " + newPath)
	}

	crew.Name = newName
	crew.ClonePath = newPath
	crew.UpdatedAt = time.Now()

	if err := m.saveState(crew); err != nil {
		// Rollback on error (best-effort)
		_ = os.Rename(newPath, oldPath)
		return errors.System("crew.SaveStateAfterRenameFailed", err).
			WithContext("old_name", oldName).
			WithContext("new_name", newName).
			WithContext("rig_name", m.rig.Name).
			WithHint("Worker was renamed but state could not be saved. Try manually fixing: " + newPath)
	}

	return nil
}

// Pristine ensures a crew worker is up-to-date with remote.
// It runs git pull --rebase.
func (m *Manager) Pristine(name string) (*PristineResult, error) {
	if err := validateCrewName(name); err != nil {
		return nil, err
	}
	if !m.exists(name) {
		return nil, errors.Permanent("crew.NotFound", nil).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithHint("Use 'gt crew list' to see available crew workers")
	}

	crewPath := m.crewDir(name)
	crewGit := git.NewGit(crewPath)

	result := &PristineResult{
		Name: name,
	}

	// Check for uncommitted changes
	hasChanges, err := crewGit.HasUncommittedChanges()
	if err != nil {
		return nil, errors.Transient("crew.CheckChangesFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", crewPath).
			WithHint("Git status check failed. Verify git installation and repository state")
	}
	result.HadChanges = hasChanges

	// Pull latest (use origin and current branch) with retry logic
	pullErr := errors.RetryWithContext(context.Background(), func() error {
		if err := crewGit.Pull("origin", ""); err != nil {
			return errors.Transient("crew.PullFailed", err).
				WithContext("worker_name", name).
				WithContext("rig_name", m.rig.Name).
				WithContext("path", crewPath)
		}
		return nil
	}, errors.NetworkRetryConfig())

	if pullErr != nil {
		result.PullError = pullErr.Error()
		// Add hint for common pull errors
		var gErr *errors.Error
		if errors.As(pullErr, &gErr) {
			gErr.WithHint("Check network connectivity and git credentials. If you have uncommitted changes, commit or stash them first")
		}
	} else {
		result.Pulled = true
	}

	// Note: With Dolt backend, beads changes are persisted immediately - no sync needed
	result.Synced = true

	return result, nil
}

// PristineResult captures the results of a pristine operation.
type PristineResult struct {
	Name       string `json:"name"`
	HadChanges bool   `json:"had_changes"`
	Pulled     bool   `json:"pulled"`
	PullError  string `json:"pull_error,omitempty"`
	Synced     bool   `json:"synced"`
	SyncError  string `json:"sync_error,omitempty"`
}

// setupSharedBeads creates a redirect file so the crew worker uses the rig's shared .beads database.
// This eliminates the need for git sync between crew clones - all crew members share one database.
func (m *Manager) setupSharedBeads(crewPath string) error {
	townRoot := filepath.Dir(m.rig.Path)
	return beads.SetupRedirect(townRoot, crewPath)
}

// SessionName returns the tmux session name for a crew member.
func (m *Manager) SessionName(name string) string {
	return fmt.Sprintf("gt-%s-crew-%s", m.rig.Name, name)
}

// Start creates and starts a tmux session for a crew member.
// If the crew member doesn't exist, it will be created first.
func (m *Manager) Start(name string, opts StartOptions) error {
	if err := validateCrewName(name); err != nil {
		return err
	}

	// Get or create the crew worker
	worker, err := m.Get(name)
	if err != nil {
		// Check if it's a not found error
		var gErr *errors.Error
		if errors.As(err, &gErr) && gErr.Op == "crew.NotFound" {
			worker, err = m.Add(name, false) // No feature branch for crew
			if err != nil {
				return errors.System("crew.CreateWorkspaceFailed", err).
					WithContext("worker_name", name).
					WithContext("rig_name", m.rig.Name).
					WithHint("Failed to create crew workspace. Check permissions and available disk space")
			}
		} else {
			return errors.System("crew.GetWorkerFailed", err).
				WithContext("worker_name", name).
				WithContext("rig_name", m.rig.Name).
				WithHint("Failed to get crew worker information")
		}
	}

	t := tmux.NewTmux()
	sessionID := m.SessionName(name)

	// Check if session already exists
	running, err := t.HasSession(sessionID)
	if err != nil {
		return errors.System("crew.CheckSessionFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("session_id", sessionID).
			WithHint("Failed to check tmux session. Verify tmux is installed and running")
	}
	if running {
		if opts.KillExisting {
			// Restart mode - kill existing session.
			// Use KillSessionWithProcesses to ensure all descendant processes are killed.
			if err := t.KillSessionWithProcesses(sessionID); err != nil {
				return errors.System("crew.KillExistingSessionFailed", err).
					WithContext("worker_name", name).
					WithContext("rig_name", m.rig.Name).
					WithContext("session_id", sessionID).
					WithHint("Failed to kill existing session. Try 'tmux kill-session -t " + sessionID + "'")
			}
		} else {
			// Normal start - session exists, check if Claude is actually running
			if t.IsClaudeRunning(sessionID) {
				return errors.Permanent("crew.SessionRunning", nil).
					WithContext("worker_name", name).
					WithContext("rig_name", m.rig.Name).
					WithContext("session_id", sessionID).
					WithHint("Stop the session first with 'gt crew stop " + name + "' or use --restart to force restart")
			}
			// Zombie session - kill and recreate.
			// Use KillSessionWithProcesses to ensure all descendant processes are killed.
			if err := t.KillSessionWithProcesses(sessionID); err != nil {
				return errors.System("crew.KillZombieSessionFailed", err).
					WithContext("worker_name", name).
					WithContext("rig_name", m.rig.Name).
					WithContext("session_id", sessionID).
					WithHint("Failed to kill zombie session. Try 'tmux kill-session -t " + sessionID + "'")
			}
		}
	}

	// Ensure Claude settings exist in crew/ (not crew/<name>/) so we don't
	// write into the source repo. Claude walks up the tree to find settings.
	// All crew members share the same settings file.
	crewBaseDir := filepath.Join(m.rig.Path, "crew")
	if err := claude.EnsureSettingsForRole(crewBaseDir, "crew"); err != nil {
		return errors.System("crew.EnsureSettingsFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("path", crewBaseDir).
			WithHint("Failed to ensure Claude settings. Check directory permissions")
	}

	// Build the startup beacon for predecessor discovery via /resume
	// Pass it as Claude's initial prompt - processed when Claude is ready
	address := fmt.Sprintf("%s/crew/%s", m.rig.Name, name)
	topic := opts.Topic
	if topic == "" {
		topic = "start"
	}
	beacon := session.FormatStartupBeacon(session.BeaconConfig{
		Recipient: address,
		Sender:    "human",
		Topic:     topic,
	})

	// Build startup command first
	// SessionStart hook handles context loading (gt prime --hook)
	claudeCmd, err := config.BuildCrewStartupCommandWithAgentOverride(m.rig.Name, name, m.rig.Path, beacon, opts.AgentOverride)
	if err != nil {
		return errors.System("crew.BuildStartupCommandFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithHint("Failed to build startup command. Check Claude configuration")
	}

	// For interactive/refresh mode, remove --dangerously-skip-permissions
	if opts.Interactive {
		claudeCmd = strings.Replace(claudeCmd, " --dangerously-skip-permissions", "", 1)
	}

	// Create session with command directly to avoid send-keys race condition.
	// See: https://github.com/anthropics/gastown/issues/280
	if err := t.NewSessionWithCommand(sessionID, worker.ClonePath, claudeCmd); err != nil {
		return errors.System("crew.CreateSessionFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("session_id", sessionID).
			WithContext("path", worker.ClonePath).
			WithHint("Failed to create tmux session. Verify tmux is installed and the worker path exists")
	}

	// Set environment variables (non-fatal: session works without these)
	// Use centralized AgentEnv for consistency across all role startup paths
	townRoot := filepath.Dir(m.rig.Path)
	envVars := config.AgentEnv(config.AgentEnvConfig{
		Role:             "crew",
		Rig:              m.rig.Name,
		AgentName:        name,
		TownRoot:         townRoot,
		RuntimeConfigDir: opts.ClaudeConfigDir,
		BeadsNoDaemon:    true,
	})
	for k, v := range envVars {
		_ = t.SetEnvironment(sessionID, k, v)
	}

	// Apply rig-based theming (non-fatal: theming failure doesn't affect operation)
	theme := tmux.AssignTheme(m.rig.Name)
	_ = t.ConfigureGasTownSession(sessionID, theme, m.rig.Name, name, "crew")

	// Set up C-b n/p keybindings for crew session cycling (non-fatal)
	_ = t.SetCrewCycleBindings(sessionID)

	// Note: We intentionally don't wait for Claude to start here.
	// The session is created in detached mode, and blocking for 60 seconds
	// serves no purpose. If the caller needs to know when Claude is ready,
	// they can check with IsClaudeRunning().

	return nil
}

// Stop terminates a crew member's tmux session.
func (m *Manager) Stop(name string) error {
	if err := validateCrewName(name); err != nil {
		return err
	}

	t := tmux.NewTmux()
	sessionID := m.SessionName(name)

	// Check if session exists
	running, err := t.HasSession(sessionID)
	if err != nil {
		return errors.System("crew.CheckSessionFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("session_id", sessionID).
			WithHint("Failed to check tmux session. Verify tmux is installed and running")
	}
	if !running {
		return errors.Permanent("crew.SessionNotFound", nil).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("session_id", sessionID).
			WithHint("Session is not running. Start it with 'gt crew start " + name + "'")
	}

	// Kill the session.
	// Use KillSessionWithProcesses to ensure all descendant processes are killed.
	// This prevents orphan bash processes from Claude's Bash tool surviving session termination.
	if err := t.KillSessionWithProcesses(sessionID); err != nil {
		return errors.System("crew.KillSessionFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("session_id", sessionID).
			WithHint("Failed to kill session. Try 'tmux kill-session -t " + sessionID + "'")
	}

	return nil
}

// IsRunning checks if a crew member's session is active.
func (m *Manager) IsRunning(name string) (bool, error) {
	t := tmux.NewTmux()
	sessionID := m.SessionName(name)
	running, err := t.HasSession(sessionID)
	if err != nil {
		return false, errors.System("crew.CheckSessionFailed", err).
			WithContext("worker_name", name).
			WithContext("rig_name", m.rig.Name).
			WithContext("session_id", sessionID).
			WithHint("Failed to check tmux session. Verify tmux is installed and running")
	}
	return running, nil
}

// IsNotFoundError checks if an error is a crew not found error.
func IsNotFoundError(err error) bool {
	var gErr *errors.Error
	if errors.As(err, &gErr) {
		return gErr.Op == "crew.NotFound"
	}
	return false
}

// IsAlreadyExistsError checks if an error is a crew already exists error.
func IsAlreadyExistsError(err error) bool {
	var gErr *errors.Error
	if errors.As(err, &gErr) {
		return gErr.Op == "crew.AlreadyExists" || gErr.Op == "crew.Exists"
	}
	return false
}

// IsUncommittedChangesError checks if an error is an uncommitted changes error.
func IsUncommittedChangesError(err error) bool {
	var gErr *errors.Error
	if errors.As(err, &gErr) {
		return gErr.Op == "crew.UncommittedChanges"
	}
	return false
}

// IsSessionRunningError checks if an error is a session already running error.
func IsSessionRunningError(err error) bool {
	var gErr *errors.Error
	if errors.As(err, &gErr) {
		return gErr.Op == "crew.SessionRunning"
	}
	return false
}

// IsSessionNotFoundError checks if an error is a session not found error.
func IsSessionNotFoundError(err error) bool {
	var gErr *errors.Error
	if errors.As(err, &gErr) {
		return gErr.Op == "crew.SessionNotFound"
	}
	return false
}

