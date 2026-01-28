package cleanup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PreflightChecker performs preflight checks before session start.
type PreflightChecker struct {
	workingDir string
	verbose    bool
}

// NewPreflightChecker creates a new preflight checker.
func NewPreflightChecker(workingDir string) *PreflightChecker {
	return &PreflightChecker{
		workingDir: workingDir,
		verbose:    false,
	}
}

// SetVerbose enables or disables verbose output.
func (p *PreflightChecker) SetVerbose(verbose bool) {
	p.verbose = verbose
}

// Check performs all preflight checks.
func (p *PreflightChecker) Check() (*PreflightCheckResult, error) {
	result := &PreflightCheckResult{
		Passed:     true,
		CanProceed: true,
	}

	// Check git status
	gitClean, uncommitted, err := p.checkGitStatus()
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("git check failed: %v", err))
		result.Passed = false
	}
	result.GitClean = gitClean
	result.NoUncommitted = len(uncommitted) == 0
	result.UncommittedFiles = uncommitted

	if !gitClean {
		result.Warnings = append(result.Warnings, "Working directory has uncommitted changes")
		result.RequiresBackup = true
	}

	// Check build state
	buildClean, buildIssues := p.checkBuildState()
	result.BuildClean = buildClean
	if !buildClean {
		result.Warnings = append(result.Warnings, buildIssues...)
	}

	// Determine if can proceed
	if !result.GitClean && !result.RequiresBackup {
		result.CanProceed = false
		result.Issues = append(result.Issues, "Uncommitted changes found and backup not enabled")
	}

	// Overall passed if no critical issues
	result.Passed = len(result.Issues) == 0

	return result, nil
}

// checkGitStatus checks git working directory status.
func (p *PreflightChecker) checkGitStatus() (bool, []string, error) {
	// Check if directory is a git repository
	gitDir := filepath.Join(p.workingDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Not a git repo, consider it "clean"
		return true, nil, nil
	}

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = p.workingDir
	output, err := cmd.Output()
	if err != nil {
		return false, nil, fmt.Errorf("git status failed: %w", err)
	}

	// Parse output
	lines := strings.Split(string(output), "\n")
	var uncommitted []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			uncommitted = append(uncommitted, line)
		}
	}

	clean := len(uncommitted) == 0
	return clean, uncommitted, nil
}

// checkBuildState checks for build artifacts and cache state.
func (p *PreflightChecker) checkBuildState() (bool, []string) {
	var issues []string
	clean := true

	// Check for common build artifact directories
	artifacts := []string{
		"dist",
		"build",
		".cache",
		"node_modules/.cache",
	}

	for _, artifact := range artifacts {
		path := filepath.Join(p.workingDir, artifact)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			// Check if directory is large
			size, err := getDirSize(path)
			if err == nil && size > 100*1024*1024 { // > 100MB
				issues = append(issues, fmt.Sprintf("%s is large (%.1fMB)", artifact, float64(size)/(1024*1024)))
				clean = false
			}
		}
	}

	return clean, issues
}

// CheckGitClean is a standalone function to check if git is clean.
func CheckGitClean(workingDir string) (bool, error) {
	checker := NewPreflightChecker(workingDir)
	clean, _, err := checker.checkGitStatus()
	return clean, err
}

// getDirSize calculates the total size of a directory.
func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// GetWorkspaceState captures the current state of a workspace.
func GetWorkspaceState(workingDir string, workspaceType WorkspaceType) (*WorkspaceState, error) {
	state := &WorkspaceState{
		Path: workingDir,
		Type: workspaceType,
	}

	// Get git status
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = workingDir
	output, err := cmd.Output()
	if err == nil {
		state.GitStatus = "clean"
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, "??") {
				state.UntrackedFiles++
			} else {
				state.DirtyFiles++
			}
		}

		if state.DirtyFiles > 0 || state.UntrackedFiles > 0 {
			state.GitStatus = "dirty"
		}
	}

	// Count temp files
	tempPatterns := []string{"*.tmp", "*.temp", "*.swp", "*.swo", "*~"}
	for _, pattern := range tempPatterns {
		matches, _ := filepath.Glob(filepath.Join(workingDir, pattern))
		state.TempFileCount += len(matches)
		for _, match := range matches {
			if info, err := os.Stat(match); err == nil {
				state.TempFileSize += info.Size()
			}
		}
	}

	// Check for active session (tmux)
	// This would require tmux integration, left for future enhancement
	state.SessionActive = false

	return state, nil
}
