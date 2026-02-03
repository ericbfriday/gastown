package hooks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// registerBuiltinHooks registers all built-in hook functions with the runner.
// Note: Workspace cleanup hooks are registered separately via cleanup.RegisterCleanupHooks()
func registerBuiltinHooks(r *HookRunner) {
	r.RegisterBuiltin("pre-shutdown-checks", preShutdownChecks)
	r.RegisterBuiltin("verify-git-clean", verifyGitClean)
	r.RegisterBuiltin("check-uncommitted", checkUncommitted)
	r.RegisterBuiltin("check-commits-pushed", checkCommitsPushed)
	r.RegisterBuiltin("check-beads-synced", checkBeadsSynced)
	r.RegisterBuiltin("check-assigned-issues", checkAssignedIssues)
}

// preShutdownChecks performs standard pre-shutdown verification.
// This is a composite hook that runs multiple checks.
func preShutdownChecks(ctx *HookContext) (*HookResult, error) {
	checks := []struct {
		name string
		fn   BuiltinHookFunc
	}{
		{"git-clean", verifyGitClean},
		{"commits-pushed", checkCommitsPushed},
		{"beads-synced", checkBeadsSynced},
		{"assigned-issues", checkAssignedIssues},
	}

	messages := []string{}
	warnings := []string{}

	for _, check := range checks {
		result, err := check.fn(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", check.name, err)
		}
		if !result.Success {
			if result.Block {
				messages = append(messages, fmt.Sprintf("%s: %s", check.name, result.Message))
			} else {
				// Non-blocking failures are warnings
				warnings = append(warnings, fmt.Sprintf("%s: %s", check.name, result.Message))
			}
		}
	}

	if len(messages) > 0 {
		msg := "Pre-shutdown checks failed:\n" + joinMessages(messages)
		if len(warnings) > 0 {
			msg += "\n\nWarnings:\n" + joinMessages(warnings)
		}
		return &HookResult{
			Success: false,
			Block:   true,
			Message: msg,
		}, nil
	}

	if len(warnings) > 0 {
		return &HookResult{
			Success: true,
			Message: "Pre-shutdown checks passed with warnings:\n" + joinMessages(warnings),
		}, nil
	}

	return &HookResult{
		Success: true,
		Message: "All pre-shutdown checks passed",
	}, nil
}

// verifyGitClean checks if the working directory has uncommitted changes.
func verifyGitClean(ctx *HookContext) (*HookResult, error) {
	workDir := ctx.WorkingDir
	if workDir == "" {
		workDir = "."
	}

	// Check if git repo exists
	gitDir := findGitDir(workDir)
	if gitDir == "" {
		// Not a git repo, pass
		return &HookResult{
			Success: true,
			Message: "Not a git repository",
		}, nil
	}

	// Check for uncommitted changes
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return &HookResult{
			Success: false,
			Message: fmt.Sprintf("Failed to check git status: %v", err),
		}, nil
	}

	if len(output) > 0 {
		return &HookResult{
			Success: false,
			Block:   true,
			Message: "Working directory has uncommitted changes",
			Output:  string(output),
		}, nil
	}

	return &HookResult{
		Success: true,
		Message: "Working directory is clean",
	}, nil
}

// checkUncommitted checks for uncommitted changes (similar to verifyGitClean).
// Kept separate for flexibility in configuration.
func checkUncommitted(ctx *HookContext) (*HookResult, error) {
	return verifyGitClean(ctx)
}

// findGitDir searches for .git directory starting from the given path.
func findGitDir(startPath string) string {
	dir := startPath
	for {
		gitPath := filepath.Join(dir, ".git")
		if stat, err := os.Stat(gitPath); err == nil && stat.IsDir() {
			return gitPath
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}
	return ""
}

// checkCommitsPushed checks if all commits have been pushed to the remote.
func checkCommitsPushed(ctx *HookContext) (*HookResult, error) {
	workDir := ctx.WorkingDir
	if workDir == "" {
		workDir = "."
	}

	// Check if git repo exists
	gitDir := findGitDir(workDir)
	if gitDir == "" {
		// Not a git repo, pass
		return &HookResult{
			Success: true,
			Message: "Not a git repository",
		}, nil
	}

	// Get current branch
	branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = workDir
	branchOutput, err := branchCmd.Output()
	if err != nil {
		return &HookResult{
			Success: false,
			Message: fmt.Sprintf("Failed to get current branch: %v", err),
		}, nil
	}
	branch := string(branchOutput)
	branch = branch[:len(branch)-1] // trim newline

	// Check for unpushed commits (compare with remote tracking branch)
	// git log @{u}..HEAD returns unpushed commits
	cmd := exec.Command("git", "log", "@{u}..HEAD", "--oneline")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		// No upstream branch configured - this is OK, might be a new branch
		return &HookResult{
			Success: true,
			Message: fmt.Sprintf("No upstream branch configured for %s", branch),
		}, nil
	}

	if len(output) > 0 {
		lines := len(strings.Split(strings.TrimSpace(string(output)), "\n"))
		return &HookResult{
			Success: false,
			Block:   true,
			Message: fmt.Sprintf("Branch %s has %d unpushed commit(s)", branch, lines),
			Output:  string(output),
		}, nil
	}

	return &HookResult{
		Success: true,
		Message: "All commits pushed to remote",
	}, nil
}

// checkBeadsSynced checks if the beads database is synced.
func checkBeadsSynced(ctx *HookContext) (*HookResult, error) {
	workDir := ctx.WorkingDir
	if workDir == "" {
		workDir = "."
	}

	// Run 'bd sync --status' to check sync state
	cmd := exec.Command("bd", "sync", "--status")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		// bd sync might not be configured, which is OK
		return &HookResult{
			Success: true,
			Message: "Beads sync not configured or not available",
		}, nil
	}

	// Check output for "in sync" or similar indicators
	statusStr := strings.ToLower(string(output))
	if strings.Contains(statusStr, "not in sync") || strings.Contains(statusStr, "pending") {
		return &HookResult{
			Success: false,
			Block:   false, // Non-blocking warning
			Message: "Beads database may not be in sync",
			Output:  string(output),
		}, nil
	}

	return &HookResult{
		Success: true,
		Message: "Beads database is synced",
	}, nil
}

// checkAssignedIssues checks if there are any assigned issues that need handling.
func checkAssignedIssues(ctx *HookContext) (*HookResult, error) {
	workDir := ctx.WorkingDir
	if workDir == "" {
		workDir = "."
	}

	// Get polecat address from metadata
	polecatName, ok := ctx.Metadata["polecat"].(string)
	if !ok || polecatName == "" {
		// No polecat info, skip check
		return &HookResult{
			Success: true,
			Message: "No polecat information available",
		}, nil
	}

	rigName, ok := ctx.Metadata["rig"].(string)
	if !ok || rigName == "" {
		// No rig info, skip check
		return &HookResult{
			Success: true,
			Message: "No rig information available",
		}, nil
	}

	// Build agent address
	agentAddr := fmt.Sprintf("%s/polecats/%s", rigName, polecatName)

	// Check for hooked issues assigned to this polecat
	cmd := exec.Command("bd", "list", "--assignee="+agentAddr, "--status=hooked", "--json")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		// bd command failed, might not be available
		return &HookResult{
			Success: true,
			Message: "Could not check assigned issues",
		}, nil
	}

	// Parse JSON output
	var issues []map[string]interface{}
	if err := json.Unmarshal(output, &issues); err != nil {
		return &HookResult{
			Success: true,
			Message: "Could not parse issues list",
		}, nil
	}

	if len(issues) > 0 {
		issueIDs := []string{}
		for _, issue := range issues {
			if id, ok := issue["id"].(string); ok {
				issueIDs = append(issueIDs, id)
			}
		}
		return &HookResult{
			Success: false,
			Block:   true,
			Message: fmt.Sprintf("Polecat has %d hooked issue(s) that need handling: %s", len(issues), strings.Join(issueIDs, ", ")),
		}, nil
	}

	return &HookResult{
		Success: true,
		Message: "No pending assigned issues",
	}, nil
}

// joinMessages joins multiple messages with newlines.
func joinMessages(messages []string) string {
	result := ""
	for i, msg := range messages {
		if i > 0 {
			result += "\n"
		}
		result += "  - " + msg
	}
	return result
}
