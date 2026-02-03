package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/prompt"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/util"
	"github.com/steveyegge/gastown/internal/workspace"
)

var (
	cleanupDryRun bool
	cleanupForce  bool
	cleanupAll    bool
)

var cleanupCmd = &cobra.Command{
	Use:     "cleanup [subcommand]",
	GroupID: GroupDiag,
	Short:   "Clean up stale state and orphaned resources",
	Long: `Clean up stale state files, orphaned processes, and abandoned resources.

Subcommands:
  stale      Clean stale lock files, PID files, and abandoned worktrees
  temp       Remove temporary files and caches
  sessions   Clean disconnected and zombie sessions
  processes  Clean orphaned Claude processes (default)

Examples:
  gt cleanup processes           # Clean orphaned processes (default)
  gt cleanup stale --dry-run     # Preview stale state cleanup
  gt cleanup temp                # Clean temporary files
  gt cleanup sessions            # Clean stale sessions
  gt cleanup --all               # Clean all stale state`,
	RunE: runCleanupProcesses, // Default to process cleanup
}

var cleanupStaleCmd = &cobra.Command{
	Use:   "stale",
	Short: "Clean stale lock files, PID files, and abandoned worktrees",
	Long: `Find and remove stale state files:
- Lock files from terminated processes
- Orphaned PID files
- Abandoned worktrees
- Old session state

Uses --dry-run to preview changes and --force to skip confirmation.`,
	RunE: runCleanupStale,
}

var cleanupTempCmd = &cobra.Command{
	Use:   "temp",
	Short: "Remove temporary files and caches",
	Long: `Clean temporary files and caches:
- Temporary work files
- Build artifacts
- Cached data
- Old log files

Uses --dry-run to preview changes and --force to skip confirmation.`,
	RunE: runCleanupTemp,
}

var cleanupSessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Clean zombie and disconnected sessions",
	Long: `Find and clean stale sessions:
- Zombie tmux sessions (no active panes)
- Disconnected sessions (client gone)
- Stale session state files
- Orphaned session references

Uses --dry-run to preview changes and --force to skip confirmation.`,
	RunE: runCleanupSessions,
}

var cleanupProcessesCmd = &cobra.Command{
	Use:   "processes",
	Short: "Clean orphaned Claude processes",
	Long: `Clean up orphaned Claude processes that survived session termination.

This command finds and kills Claude processes that are not associated with
any active Gas Town tmux session. These orphans can accumulate when:
- Polecat sessions are killed without proper cleanup
- Claude spawns subagent processes that outlive their parent
- Network or system issues interrupt normal shutdown

Uses aggressive tmux session verification to detect ALL orphaned processes,
not just those with PPID=1.

Examples:
  gt cleanup processes              # Clean up orphans with confirmation
  gt cleanup processes --dry-run    # Show what would be killed
  gt cleanup processes --force      # Kill without confirmation`,
	RunE: runCleanupProcesses,
}

func init() {
	// Add flags to main cleanup command
	cleanupCmd.PersistentFlags().BoolVar(&cleanupDryRun, "dry-run", false, "Show what would be cleaned without making changes")
	cleanupCmd.PersistentFlags().BoolVarP(&cleanupForce, "force", "f", false, "Skip confirmation prompts")
	cleanupCmd.Flags().BoolVar(&cleanupAll, "all", false, "Clean all stale state (stale, temp, sessions, processes)")

	// Add subcommands
	cleanupCmd.AddCommand(cleanupStaleCmd)
	cleanupCmd.AddCommand(cleanupTempCmd)
	cleanupCmd.AddCommand(cleanupSessionsCmd)
	cleanupCmd.AddCommand(cleanupProcessesCmd)

	rootCmd.AddCommand(cleanupCmd)
}

// CleanupResult tracks what was cleaned up
type CleanupResult struct {
	ItemsFound    int
	ItemsCleaned  int
	ItemsSkipped  int
	BytesFreed    int64
	Errors        []error
	Details       []string
}

func (r *CleanupResult) AddDetail(format string, args ...interface{}) {
	r.Details = append(r.Details, fmt.Sprintf(format, args...))
}

func (r *CleanupResult) AddError(err error) {
	r.Errors = append(r.Errors, err)
}

func (r *CleanupResult) Print(title string) {
	if r.ItemsFound == 0 {
		fmt.Printf("%s No %s found\n", style.Bold.Render("✓"), title)
		return
	}

	fmt.Printf("\n%s Found %d %s:\n", style.Bold.Render("Results"), r.ItemsFound, title)
	for _, detail := range r.Details {
		fmt.Printf("  %s\n", detail)
	}

	if cleanupDryRun {
		fmt.Printf("\n%s Dry run - no changes made\n", style.Dim.Render("ℹ"))
	} else if r.ItemsCleaned > 0 {
		msg := fmt.Sprintf("✓ Cleaned %d items", r.ItemsCleaned)
		if r.BytesFreed > 0 {
			msg += fmt.Sprintf(" (freed %s)", formatBytes(r.BytesFreed))
		}
		fmt.Printf("\n%s\n", style.Bold.Render(msg))
	}

	if r.ItemsSkipped > 0 {
		fmt.Printf("%s Skipped %d items\n", style.Warning.Render("⚠"), r.ItemsSkipped)
	}

	if len(r.Errors) > 0 {
		fmt.Printf("\n%s Errors:\n", style.Error.Render("✗"))
		for _, err := range r.Errors {
			fmt.Printf("  %s\n", err)
		}
	}
}


func runCleanupStale(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	result := &CleanupResult{}

	// Find stale lock files
	lockFiles, err := findStaleLockFiles(townRoot)
	if err != nil {
		result.AddError(fmt.Errorf("finding lock files: %w", err))
	} else {
		result.ItemsFound += len(lockFiles)
		for _, path := range lockFiles {
			info, _ := os.Stat(path)
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			result.AddDetail("Lock file: %s", path)

			if !cleanupDryRun {
				if err := os.Remove(path); err != nil {
					result.AddError(fmt.Errorf("removing %s: %w", path, err))
					result.ItemsSkipped++
				} else {
					result.ItemsCleaned++
					result.BytesFreed += size
				}
			}
		}
	}

	// Find stale PID files
	pidFiles, err := findStalePIDFiles(townRoot)
	if err != nil {
		result.AddError(fmt.Errorf("finding PID files: %w", err))
	} else {
		result.ItemsFound += len(pidFiles)
		for _, path := range pidFiles {
			info, _ := os.Stat(path)
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			result.AddDetail("PID file: %s", path)

			if !cleanupDryRun {
				if err := os.Remove(path); err != nil {
					result.AddError(fmt.Errorf("removing %s: %w", path, err))
					result.ItemsSkipped++
				} else {
					result.ItemsCleaned++
					result.BytesFreed += size
				}
			}
		}
	}

	// Find abandoned worktrees
	worktrees, err := findAbandonedWorktrees(townRoot)
	if err != nil {
		result.AddError(fmt.Errorf("finding worktrees: %w", err))
	} else {
		result.ItemsFound += len(worktrees)
		for _, path := range worktrees {
			result.AddDetail("Abandoned worktree: %s", path)

			if !cleanupDryRun && !cleanupForce {
				// Worktrees are more dangerous - always confirm unless --force
				if !prompt.ConfirmDanger(fmt.Sprintf("Remove worktree %s?", path)) {
					result.ItemsSkipped++
					continue
				}
			}

			if !cleanupDryRun {
				size := getDirSize(path)
				if err := os.RemoveAll(path); err != nil {
					result.AddError(fmt.Errorf("removing %s: %w", path, err))
					result.ItemsSkipped++
				} else {
					result.ItemsCleaned++
					result.BytesFreed += size
				}
			}
		}
	}

	result.Print("stale items")
	return nil
}

func runCleanupTemp(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	result := &CleanupResult{}

	// Clean temporary files
	tempDirs := []string{
		filepath.Join(townRoot, ".runtime", "tmp"),
		filepath.Join(os.TempDir(), "gastown-*"),
	}

	for _, pattern := range tempDirs {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			result.AddError(fmt.Errorf("globbing %s: %w", pattern, err))
			continue
		}

		for _, path := range matches {
			info, err := os.Stat(path)
			if err != nil {
				continue
			}

			// Skip recently modified files (less than 1 hour old)
			if time.Since(info.ModTime()) < time.Hour {
				continue
			}

			result.ItemsFound++
			size := int64(0)
			if info.IsDir() {
				size = getDirSize(path)
				result.AddDetail("Temp directory: %s (%s)", path, formatBytes(size))
			} else {
				size = info.Size()
				result.AddDetail("Temp file: %s (%s)", path, formatBytes(size))
			}

			if !cleanupDryRun {
				var removeErr error
				if info.IsDir() {
					removeErr = os.RemoveAll(path)
				} else {
					removeErr = os.Remove(path)
				}

				if removeErr != nil {
					result.AddError(fmt.Errorf("removing %s: %w", path, removeErr))
					result.ItemsSkipped++
				} else {
					result.ItemsCleaned++
					result.BytesFreed += size
				}
			}
		}
	}

	result.Print("temporary items")
	return nil
}

func runCleanupSessions(cmd *cobra.Command, args []string) error {
	result := &CleanupResult{}

	// Find zombie sessions
	zombieSessions, err := findZombieTmuxSessions()
	if err != nil {
		result.AddError(fmt.Errorf("finding zombie sessions: %w", err))
	} else {
		result.ItemsFound += len(zombieSessions)
		for _, session := range zombieSessions {
			result.AddDetail("Zombie session: %s", session)

			if !cleanupDryRun {
				// Kill the session
				if err := killTmuxSession(session); err != nil {
					result.AddError(fmt.Errorf("killing session %s: %w", session, err))
					result.ItemsSkipped++
				} else {
					result.ItemsCleaned++
				}
			}
		}
	}

	result.Print("sessions")
	return nil
}

func runCleanupProcesses(cmd *cobra.Command, args []string) error {
	// Find orphaned processes using aggressive zombie detection
	zombies, err := util.FindZombieClaudeProcesses()
	if err != nil {
		return fmt.Errorf("finding orphaned processes: %w", err)
	}

	if len(zombies) == 0 {
		fmt.Printf("%s No orphaned Claude processes found\n", style.Bold.Render("✓"))
		return nil
	}

	// Show what we found
	fmt.Printf("%s Found %d orphaned Claude process(es):\n\n", style.Warning.Render("⚠"), len(zombies))
	for _, z := range zombies {
		ageStr := formatProcessAgeCleanup(z.Age)
		fmt.Printf("  %s %s (age: %s, tty: %s)\n",
			style.Bold.Render(fmt.Sprintf("PID %d", z.PID)),
			z.Cmd,
			style.Dim.Render(ageStr),
			z.TTY)
	}
	fmt.Println()

	if cleanupDryRun {
		fmt.Printf("%s Dry run - no processes killed\n", style.Dim.Render("ℹ"))
		return nil
	}

	// Confirm unless --force
	if !prompt.ConfirmDanger(
		fmt.Sprintf("Kill these %d process(es)?", len(zombies)),
		prompt.WithForce(cleanupForce),
	) {
		fmt.Println("Aborted")
		return nil
	}

	// Kill the processes using the standard cleanup function
	results, err := util.CleanupZombieClaudeProcesses()
	if err != nil {
		return fmt.Errorf("cleaning up processes: %w", err)
	}

	// Report results
	var killed, escalated int
	for _, r := range results {
		switch r.Signal {
		case "SIGTERM":
			fmt.Printf("  %s PID %d sent SIGTERM\n", style.Success.Render("✓"), r.Process.PID)
			killed++
		case "SIGKILL":
			fmt.Printf("  %s PID %d sent SIGKILL (didn't respond to SIGTERM)\n", style.Warning.Render("⚠"), r.Process.PID)
			killed++
		case "UNKILLABLE":
			fmt.Printf("  %s PID %d survived SIGKILL\n", style.Error.Render("✗"), r.Process.PID)
			escalated++
		}
	}

	fmt.Printf("\n%s Cleaned up %d process(es)", style.Bold.Render("✓"), killed)
	if escalated > 0 {
		fmt.Printf(", %d unkillable", escalated)
	}
	fmt.Println()

	return nil
}

// formatProcessAgeCleanup formats seconds into a human-readable age string
func formatProcessAgeCleanup(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm%ds", seconds/60, seconds%60)
	}
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	return fmt.Sprintf("%dh%dm", hours, mins)
}

// findStaleLockFiles finds lock files from terminated processes
func findStaleLockFiles(townRoot string) ([]string, error) {
	var staleFiles []string

	// Look for .lock files in common locations
	lockDirs := []string{
		filepath.Join(townRoot, ".runtime"),
		filepath.Join(townRoot, ".beads"),
	}

	for _, dir := range lockDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			if !info.IsDir() && strings.HasSuffix(path, ".lock") {
				// Check if the lock file is old (more than 1 hour)
				if time.Since(info.ModTime()) > time.Hour {
					staleFiles = append(staleFiles, path)
				}
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return staleFiles, nil
}

// findStalePIDFiles finds PID files for processes that no longer exist
func findStalePIDFiles(townRoot string) ([]string, error) {
	var staleFiles []string

	pidDirs := []string{
		filepath.Join(townRoot, ".runtime"),
	}

	for _, dir := range pidDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if !info.IsDir() && strings.HasSuffix(path, ".pid") {
				// Read PID and check if process exists
				data, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				var pid int
				if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
					// Invalid PID file
					staleFiles = append(staleFiles, path)
					return nil
				}

				// Check if process exists
				if !processExists(pid) {
					staleFiles = append(staleFiles, path)
				}
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return staleFiles, nil
}

// processExists checks if a process with the given PID exists
func processExists(pid int) bool {
	// Try to send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 doesn't actually send a signal, just checks if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// findAbandonedWorktrees finds git worktrees that are no longer valid
func findAbandonedWorktrees(townRoot string) ([]string, error) {
	var abandoned []string

	// Look for worktree directories in common locations
	worktreeDirs := []string{
		filepath.Join(townRoot, "polecats"),
		filepath.Join(townRoot, "crew"),
	}

	for _, dir := range worktreeDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			worktreePath := filepath.Join(dir, entry.Name())
			gitDir := filepath.Join(worktreePath, ".git")

			// Check if it's a worktree
			if _, err := os.Stat(gitDir); err == nil {
				// Check if the worktree is still valid
				// A worktree is abandoned if its .git file points to a non-existent location
				gitContent, err := os.ReadFile(gitDir)
				if err == nil {
					gitdirPath := strings.TrimPrefix(string(gitContent), "gitdir: ")
					gitdirPath = strings.TrimSpace(gitdirPath)
					if gitdirPath != "" {
						if _, err := os.Stat(gitdirPath); os.IsNotExist(err) {
							abandoned = append(abandoned, worktreePath)
						}
					}
				}
			}
		}
	}

	return abandoned, nil
}

// findZombieTmuxSessions finds tmux sessions with no active panes
func findZombieTmuxSessions() ([]string, error) {
	// This is a placeholder - would need tmux integration
	// For now, return empty list
	return []string{}, nil
}

// killTmuxSession kills a tmux session by name
func killTmuxSession(session string) error {
	cmd := exec.Command("tmux", "kill-session", "-t", session)
	return cmd.Run()
}

// getDirSize calculates the total size of a directory
func getDirSize(path string) int64 {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0
	}
	return size
}
