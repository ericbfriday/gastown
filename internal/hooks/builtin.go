package hooks

import (
	"fmt"
	"os"
	"os/exec"
)

// registerBuiltinHooks registers all built-in hook functions with the runner.
func registerBuiltinHooks(r *HookRunner) {
	r.RegisterBuiltin("pre-shutdown-checks", preShutdownChecks)
	r.RegisterBuiltin("verify-git-clean", verifyGitClean)
	r.RegisterBuiltin("check-uncommitted", checkUncommitted)
}

// preShutdownChecks performs standard pre-shutdown verification.
// This is a composite hook that runs multiple checks.
func preShutdownChecks(ctx *HookContext) (*HookResult, error) {
	checks := []struct {
		name string
		fn   BuiltinHookFunc
	}{
		{"git-clean", verifyGitClean},
		{"uncommitted-check", checkUncommitted},
	}

	messages := []string{}
	for _, check := range checks {
		result, err := check.fn(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", check.name, err)
		}
		if !result.Success {
			messages = append(messages, fmt.Sprintf("%s: %s", check.name, result.Message))
		}
	}

	if len(messages) > 0 {
		return &HookResult{
			Success: false,
			Block:   true,
			Message: fmt.Sprintf("Pre-shutdown checks failed:\n%s", joinMessages(messages)),
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
		gitPath := fmt.Sprintf("%s/.git", dir)
		if stat, err := os.Stat(gitPath); err == nil && stat.IsDir() {
			return gitPath
		}

		// Move up one directory
		parent := fmt.Sprintf("%s/..", dir)
		if parent == dir {
			break // Reached root
		}
		dir = parent
	}
	return ""
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
