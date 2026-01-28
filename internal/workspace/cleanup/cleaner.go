package cleanup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Cleaner performs workspace cleanup operations.
type Cleaner struct {
	config      *CleanupConfig
	workingDir  string
	dryRun      bool
	backupDir   string
	verbose     bool
}

// NewCleaner creates a new workspace cleaner.
func NewCleaner(workingDir string, config *CleanupConfig) *Cleaner {
	return &Cleaner{
		config:     config,
		workingDir: workingDir,
		dryRun:     config.DryRun,
		backupDir:  filepath.Join(workingDir, ".gastown", "backups"),
		verbose:    false,
	}
}

// SetDryRun enables or disables dry-run mode.
func (c *Cleaner) SetDryRun(dryRun bool) {
	c.dryRun = dryRun
}

// SetVerbose enables or disables verbose output.
func (c *Cleaner) SetVerbose(verbose bool) {
	c.verbose = verbose
}

// RunPreflight executes preflight cleanup rules.
func (c *Cleaner) RunPreflight() ([]*CleanupResult, error) {
	if !c.config.Enabled {
		return nil, nil
	}

	var results []*CleanupResult
	for _, rule := range c.config.Preflight {
		if !rule.Enabled {
			continue
		}

		result, err := c.executeRule(rule)
		if err != nil {
			return results, fmt.Errorf("preflight rule %s failed: %w", rule.Action, err)
		}
		results = append(results, result)

		// If safety check fails, abort
		if !result.Success && rule.SafetyCheck {
			return results, fmt.Errorf("safety check failed: %s", rule.Action)
		}
	}

	return results, nil
}

// RunPostflight executes postflight cleanup rules.
func (c *Cleaner) RunPostflight() ([]*CleanupResult, error) {
	if !c.config.Enabled {
		return nil, nil
	}

	var results []*CleanupResult
	for _, rule := range c.config.Postflight {
		if !rule.Enabled {
			continue
		}

		result, err := c.executeRule(rule)
		if err != nil {
			// Postflight errors are non-fatal, log and continue
			if c.verbose {
				fmt.Fprintf(os.Stderr, "Warning: postflight rule %s: %v\n", rule.Action, err)
			}
		}
		results = append(results, result)
	}

	return results, nil
}

// RunOnIdle executes idle cleanup rules.
func (c *Cleaner) RunOnIdle() ([]*CleanupResult, error) {
	if !c.config.Enabled {
		return nil, nil
	}

	var results []*CleanupResult
	for _, rule := range c.config.OnIdle {
		if !rule.Enabled {
			continue
		}

		result, err := c.executeRule(rule)
		if err != nil && c.verbose {
			fmt.Fprintf(os.Stderr, "Warning: idle rule %s: %v\n", rule.Action, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// executeRule executes a single cleanup rule.
func (c *Cleaner) executeRule(rule CleanupRule) (*CleanupResult, error) {
	start := time.Now()

	switch rule.Action {
	case ActionVerifyGitClean:
		return c.verifyGitClean()
	case ActionBackupUncommitted:
		return c.backupUncommitted()
	case ActionRemoveTempFiles:
		return c.removeFilesByPattern(rule)
	case ActionRemoveLogs:
		return c.removeFilesByPattern(rule)
	case ActionRemoveDSStore:
		return c.removeFilesByPattern(rule)
	case ActionClearBuildCache:
		return c.clearBuildCache(rule)
	case ActionCleanGitWorktree:
		return c.cleanGitWorktree()
	case ActionCleanNodeModules:
		return c.cleanNodeModules(rule)
	default:
		return &CleanupResult{
			Action:   rule.Action,
			Success:  false,
			Duration: time.Since(start),
			Errors:   []string{fmt.Sprintf("unknown action: %s", rule.Action)},
		}, fmt.Errorf("unknown action: %s", rule.Action)
	}
}

// verifyGitClean checks if git working directory is clean.
func (c *Cleaner) verifyGitClean() (*CleanupResult, error) {
	start := time.Now()
	result := &CleanupResult{
		Action:  ActionVerifyGitClean,
		Success: false,
		DryRun:  c.dryRun,
	}

	// Run git status --porcelain
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = c.workingDir
	output, err := cmd.Output()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("git status failed: %v", err))
		result.Duration = time.Since(start)
		return result, err
	}

	// Check if output is empty (clean)
	if len(output) == 0 {
		result.Success = true
	} else {
		result.Success = false
		result.Errors = append(result.Errors, "git working directory is not clean")
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if line != "" {
				result.Details = append(result.Details, FileAction{
					Path:   strings.TrimSpace(line),
					Action: "dirty",
					Reason: "uncommitted changes",
				})
			}
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// backupUncommitted creates a backup of uncommitted changes.
func (c *Cleaner) backupUncommitted() (*CleanupResult, error) {
	start := time.Now()
	result := &CleanupResult{
		Action:  ActionBackupUncommitted,
		Success: false,
		DryRun:  c.dryRun,
	}

	// Check for uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = c.workingDir
	output, err := cmd.Output()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("git status failed: %v", err))
		result.Duration = time.Since(start)
		return result, err
	}

	// If clean, no backup needed
	if len(output) == 0 {
		result.Success = true
		result.Duration = time.Since(start)
		return result, nil
	}

	if c.dryRun {
		result.Success = true
		result.Details = append(result.Details, FileAction{
			Action: "would-backup",
			Reason: "dry-run mode",
		})
		result.Duration = time.Since(start)
		return result, nil
	}

	// Create backup directory
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(c.backupDir, fmt.Sprintf("backup-%s.patch", timestamp))

	if err := os.MkdirAll(c.backupDir, 0755); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to create backup dir: %v", err))
		result.Duration = time.Since(start)
		return result, err
	}

	// Create git diff patch
	cmd = exec.Command("git", "diff", "HEAD")
	cmd.Dir = c.workingDir
	patch, err := cmd.Output()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("git diff failed: %v", err))
		result.Duration = time.Since(start)
		return result, err
	}

	if err := os.WriteFile(backupPath, patch, 0644); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("failed to write backup: %v", err))
		result.Duration = time.Since(start)
		return result, err
	}

	result.Success = true
	result.Details = append(result.Details, FileAction{
		Action:     "backed-up",
		BackupPath: backupPath,
		Size:       int64(len(patch)),
	})
	result.Duration = time.Since(start)
	return result, nil
}

// removeFilesByPattern removes files matching patterns.
func (c *Cleaner) removeFilesByPattern(rule CleanupRule) (*CleanupResult, error) {
	start := time.Now()
	result := &CleanupResult{
		Action:  rule.Action,
		Success: true,
		DryRun:  c.dryRun,
	}

	for _, pattern := range rule.Patterns {
		files, err := c.findFiles(pattern, rule.Recursive, rule.MaxDepth, rule.Exclude)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("pattern %s: %v", pattern, err))
			continue
		}

		for _, file := range files {
			result.FilesFound++

			info, err := os.Stat(file)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", file, err))
				continue
			}

			if c.dryRun {
				result.Details = append(result.Details, FileAction{
					Path:   file,
					Action: "would-remove",
					Size:   info.Size(),
				})
			} else {
				if err := os.Remove(file); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", file, err))
				} else {
					result.FilesRemoved++
					result.BytesFreed += info.Size()
					result.Details = append(result.Details, FileAction{
						Path:   file,
						Action: "removed",
						Size:   info.Size(),
					})
				}
			}
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// clearBuildCache clears build caches.
func (c *Cleaner) clearBuildCache(rule CleanupRule) (*CleanupResult, error) {
	// Reuse removeFilesByPattern logic
	return c.removeFilesByPattern(rule)
}

// cleanGitWorktree cleans git worktree (removes untracked files).
func (c *Cleaner) cleanGitWorktree() (*CleanupResult, error) {
	start := time.Now()
	result := &CleanupResult{
		Action:  ActionCleanGitWorktree,
		Success: false,
		DryRun:  c.dryRun,
	}

	args := []string{"clean", "-fd"}
	if c.dryRun {
		args = append(args, "-n")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = c.workingDir
	output, err := cmd.Output()
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("git clean failed: %v", err))
		result.Duration = time.Since(start)
		return result, err
	}

	result.Success = true
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line != "" && strings.HasPrefix(line, "Removing ") {
			path := strings.TrimPrefix(line, "Removing ")
			result.FilesRemoved++
			result.Details = append(result.Details, FileAction{
				Path:   path,
				Action: "removed",
			})
		} else if line != "" && strings.HasPrefix(line, "Would remove ") {
			path := strings.TrimPrefix(line, "Would remove ")
			result.FilesFound++
			result.Details = append(result.Details, FileAction{
				Path:   path,
				Action: "would-remove",
			})
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// cleanNodeModules cleans node_modules directory.
func (c *Cleaner) cleanNodeModules(rule CleanupRule) (*CleanupResult, error) {
	start := time.Now()
	result := &CleanupResult{
		Action:  ActionCleanNodeModules,
		Success: true,
		DryRun:  c.dryRun,
	}

	nodeModulesPath := filepath.Join(c.workingDir, "node_modules")
	info, err := os.Stat(nodeModulesPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Not an error if node_modules doesn't exist
			result.Success = true
			result.Duration = time.Since(start)
			return result, nil
		}
		result.Errors = append(result.Errors, fmt.Sprintf("stat node_modules: %v", err))
		result.Duration = time.Since(start)
		return result, err
	}

	if c.dryRun {
		result.Details = append(result.Details, FileAction{
			Path:   nodeModulesPath,
			Action: "would-remove",
			Size:   info.Size(),
		})
	} else {
		if err := os.RemoveAll(nodeModulesPath); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("remove node_modules: %v", err))
			result.Duration = time.Since(start)
			return result, err
		}
		result.FilesRemoved = 1
		result.Details = append(result.Details, FileAction{
			Path:   nodeModulesPath,
			Action: "removed",
			Size:   info.Size(),
		})
	}

	result.Duration = time.Since(start)
	return result, nil
}

// findFiles finds files matching a pattern.
func (c *Cleaner) findFiles(pattern string, recursive bool, maxDepth int, exclude []string) ([]string, error) {
	var files []string

	if !recursive {
		// Non-recursive: match pattern in working directory only
		matches, err := filepath.Glob(filepath.Join(c.workingDir, pattern))
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	} else {
		// Recursive: walk directory tree
		err := filepath.Walk(c.workingDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Check depth limit
			if maxDepth > 0 {
				relPath, _ := filepath.Rel(c.workingDir, path)
				depth := strings.Count(relPath, string(filepath.Separator))
				if depth > maxDepth {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Check if excluded
			for _, excl := range exclude {
				matched, _ := filepath.Match(excl, path)
				if matched {
					return nil
				}
			}

			// Match pattern
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				return err
			}

			if matched {
				files = append(files, path)
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return files, nil
}
