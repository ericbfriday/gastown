// Package cleanup provides workspace cleanup with preflight and postflight hooks.
// This package integrates with the lifecycle hooks system to maintain clean workspaces
// for both persistent crew members and ephemeral polecats.
package cleanup

import (
	"time"
)

// WorkspaceType identifies different types of workspaces.
type WorkspaceType string

const (
	WorkspaceTypeCrew     WorkspaceType = "crew"     // Persistent worker clone
	WorkspaceTypePolecat  WorkspaceType = "polecat"  // Ephemeral worker worktree
	WorkspaceTypeMayor    WorkspaceType = "mayor"    // Canonical read-only clone
	WorkspaceTypeRefinery WorkspaceType = "refinery" // Merge queue worktree
	WorkspaceTypeTown     WorkspaceType = "town"     // Town-level workspace
)

// CleanupAction represents a type of cleanup operation.
type CleanupAction string

const (
	ActionRemoveTempFiles   CleanupAction = "remove-temp-files"
	ActionClearBuildCache   CleanupAction = "clear-build-cache"
	ActionRemoveLogs        CleanupAction = "remove-logs"
	ActionCleanNodeModules  CleanupAction = "clean-node-modules"
	ActionRemoveDSStore     CleanupAction = "remove-ds-store"
	ActionCleanGitWorktree  CleanupAction = "clean-git-worktree"
	ActionVerifyGitClean    CleanupAction = "verify-git-clean"
	ActionBackupUncommitted CleanupAction = "backup-uncommitted"
)

// CleanupRule defines what to clean for a specific workspace type.
type CleanupRule struct {
	Action      CleanupAction `json:"action"`
	Enabled     bool          `json:"enabled"`
	Patterns    []string      `json:"patterns,omitempty"`    // File patterns to match
	Exclude     []string      `json:"exclude,omitempty"`     // Patterns to exclude
	Recursive   bool          `json:"recursive"`             // Recurse into subdirectories
	MaxDepth    int           `json:"max_depth,omitempty"`   // Maximum recursion depth (0=unlimited)
	SafetyCheck bool          `json:"safety_check"`          // Verify before deletion
	BackupFirst bool          `json:"backup_first"`          // Backup before deletion
	Description string        `json:"description,omitempty"` // Human-readable description
}

// CleanupConfig defines cleanup configuration for a workspace type.
type CleanupConfig struct {
	WorkspaceType WorkspaceType  `json:"workspace_type"`
	Preflight     []CleanupRule  `json:"preflight"`  // Run before session starts
	Postflight    []CleanupRule  `json:"postflight"` // Run after session ends
	OnIdle        []CleanupRule  `json:"on_idle"`    // Run when session is idle
	Enabled       bool           `json:"enabled"`
	DryRun        bool           `json:"dry_run"` // Preview changes without applying
}

// CleanupResult represents the result of a cleanup operation.
type CleanupResult struct {
	Action      CleanupAction `json:"action"`
	Success     bool          `json:"success"`
	FilesFound  int           `json:"files_found"`
	FilesRemoved int          `json:"files_removed"`
	BytesFreed  int64         `json:"bytes_freed"`
	Duration    time.Duration `json:"duration"`
	Errors      []string      `json:"errors,omitempty"`
	DryRun      bool          `json:"dry_run"`
	Details     []FileAction  `json:"details,omitempty"`
}

// FileAction represents an action taken on a file.
type FileAction struct {
	Path      string    `json:"path"`
	Action    string    `json:"action"` // "removed", "backed-up", "skipped"
	Size      int64     `json:"size"`
	Reason    string    `json:"reason,omitempty"`
	BackupPath string   `json:"backup_path,omitempty"`
}

// PreflightCheckResult represents the result of preflight checks.
type PreflightCheckResult struct {
	Passed          bool     `json:"passed"`
	GitClean        bool     `json:"git_clean"`
	NoUncommitted   bool     `json:"no_uncommitted"`
	BuildClean      bool     `json:"build_clean"`
	Issues          []string `json:"issues,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	CanProceed      bool     `json:"can_proceed"`
	RequiresBackup  bool     `json:"requires_backup"`
	UncommittedFiles []string `json:"uncommitted_files,omitempty"`
}

// WorkspaceState captures the state of a workspace.
type WorkspaceState struct {
	Path            string        `json:"path"`
	Type            WorkspaceType `json:"type"`
	GitStatus       string        `json:"git_status"`
	DirtyFiles      int           `json:"dirty_files"`
	UntrackedFiles  int           `json:"untracked_files"`
	TempFileCount   int           `json:"temp_file_count"`
	TempFileSize    int64         `json:"temp_file_size"`
	LastCleanup     time.Time     `json:"last_cleanup"`
	SessionActive   bool          `json:"session_active"`
}
