package errors

import (
	"fmt"
	"strings"
)

// Common recovery hints for various error scenarios.
const (
	HintPolecatNotFound = "Use 'gt polecat list' to see available polecats, or 'gt polecat create' to create a new one."
	HintPolecatList     = "Run 'gt polecat list' to see all polecats and their status."
	HintPolecatCreate   = "Create a polecat with 'gt polecat create <issue-id>'."

	HintRigNotFound = "Use 'gt rig list' to see available rigs, or ensure you're in a Gas Town workspace."
	HintRigList     = "Run 'gt rig list' to see all rigs in this workspace."

	HintSessionNotFound = "The session may have been stopped or never started. Check with 'gt status'."
	HintSessionStatus   = "Run 'gt status' to see all active sessions."
	HintSessionStart    = "Start the session with the appropriate start command (e.g., 'gt refinery start')."

	HintGitPullFailed = "Try running 'git pull' manually to see the specific error. You may need to resolve conflicts or check your network connection."
	HintGitPushFailed = "Verify you have push permissions and the remote is accessible. Try 'git push' manually for more details."
	HintGitConflict   = "Resolve merge conflicts manually with 'git status' and 'git merge', then retry."

	HintBeadsNotFound    = "Ensure the beads CLI ('bd') is installed and in your PATH. See https://github.com/steveyegge/beads for installation."
	HintBeadsNotInstalled = "Install beads with: brew install beads (or see https://github.com/steveyegge/beads)"
	HintBeadsSyncNeeded  = "Run 'bd sync' to synchronize beads with the remote."

	HintIssueNotFound = "Check the issue ID is correct. Use 'bd list' to see available issues."
	HintIssueClosed   = "This issue is closed. Use 'bd show <issue-id>' to see details."

	HintMRNotFound = "Use 'gt mq list' to see merge requests in the queue."
	HintMRList     = "Run 'gt mq list' to see all merge requests and their status."

	HintNetworkError  = "Check your network connection and try again. If the problem persists, verify the remote server is accessible."
	HintTimeoutError  = "The operation timed out. Try again, or check if the service is experiencing issues."
	HintRateLimited   = "You've exceeded rate limits. Wait a moment and try again."

	HintFileNotFound    = "Verify the file path is correct and the file exists."
	HintPermissionDenied = "Check file permissions. You may need to run with appropriate privileges."
	HintDiskFull        = "Free up disk space and try again."

	HintConfigInvalid   = "Check the configuration file syntax and required fields. See documentation for the correct format."
	HintConfigNotFound  = "Create a configuration file or ensure you're in the correct directory."

	HintTmuxNotFound = "Install tmux: brew install tmux (macOS) or apt-get install tmux (Linux)."
	HintTmuxError    = "Check if tmux is running: 'tmux list-sessions'. You may need to kill and restart sessions."

	HintWorkspaceNotFound = "Ensure you're in a Gas Town workspace. Initialize one with 'gt init'."
	HintWorkspaceInit     = "Initialize a workspace with 'gt init' in the desired directory."

	HintRetryOperation = "This operation can be retried. Wait a moment and try again."
	HintContactSupport = "If this problem persists, please file an issue at https://github.com/steveyegge/gastown/issues"
)

// HintBuilder helps construct contextual recovery hints.
type HintBuilder struct {
	hints []string
}

// NewHintBuilder creates a new HintBuilder.
func NewHintBuilder() *HintBuilder {
	return &HintBuilder{hints: []string{}}
}

// Add adds a hint to the builder.
func (h *HintBuilder) Add(hint string) *HintBuilder {
	if hint != "" {
		h.hints = append(h.hints, hint)
	}
	return h
}

// AddIf conditionally adds a hint if condition is true.
func (h *HintBuilder) AddIf(condition bool, hint string) *HintBuilder {
	if condition && hint != "" {
		h.hints = append(h.hints, hint)
	}
	return h
}

// AddFormatted adds a formatted hint to the builder.
func (h *HintBuilder) AddFormatted(format string, args ...interface{}) *HintBuilder {
	h.hints = append(h.hints, fmt.Sprintf(format, args...))
	return h
}

// Build returns the combined hint string.
func (h *HintBuilder) Build() string {
	if len(h.hints) == 0 {
		return ""
	}
	if len(h.hints) == 1 {
		return h.hints[0]
	}
	return strings.Join(h.hints, "\n")
}

// WithPolecatNotFoundHint adds context-specific hint for polecat not found.
func WithPolecatNotFoundHint(polecatName string) string {
	return fmt.Sprintf("Polecat '%s' not found. %s", polecatName, HintPolecatNotFound)
}

// WithRigNotFoundHint adds context-specific hint for rig not found.
func WithRigNotFoundHint(rigName string) string {
	return fmt.Sprintf("Rig '%s' not found. %s", rigName, HintRigNotFound)
}

// WithIssueNotFoundHint adds context-specific hint for issue not found.
func WithIssueNotFoundHint(issueID string) string {
	return fmt.Sprintf("Issue '%s' not found. %s", issueID, HintIssueNotFound)
}

// WithMRNotFoundHint adds context-specific hint for merge request not found.
func WithMRNotFoundHint(mrID string) string {
	return fmt.Sprintf("Merge request '%s' not found. %s", mrID, HintMRNotFound)
}

// WithGitBranchHint adds context-specific hint for git branch operations.
func WithGitBranchHint(branch string) string {
	return fmt.Sprintf("Branch '%s' may not exist. Check with 'git branch -a'.", branch)
}

// WithCommandNotFoundHint adds context-specific hint for missing commands.
func WithCommandNotFoundHint(cmd string) string {
	hints := map[string]string{
		"bd":   HintBeadsNotInstalled,
		"tmux": HintTmuxNotFound,
		"git":  "Install git: brew install git (macOS) or apt-get install git (Linux).",
	}
	if hint, ok := hints[cmd]; ok {
		return hint
	}
	return fmt.Sprintf("Command '%s' not found. Ensure it's installed and in your PATH.", cmd)
}

// SuggestRetry creates a hint suggesting retry with operation details.
func SuggestRetry(operation string, reason string) string {
	if reason != "" {
		return fmt.Sprintf("The %s operation failed (%s) but can be retried. %s", operation, reason, HintRetryOperation)
	}
	return fmt.Sprintf("The %s operation failed but can be retried. %s", operation, HintRetryOperation)
}

// SuggestManualIntervention creates a hint for when manual intervention is needed.
func SuggestManualIntervention(operation, steps string) string {
	return fmt.Sprintf("The %s operation requires manual intervention:\n%s", operation, steps)
}

// SuggestCheckCommand creates a hint to run a check command.
func SuggestCheckCommand(description, command string) string {
	return fmt.Sprintf("%s\nRun: %s", description, command)
}

// SuggestDocumentation creates a hint pointing to documentation.
func SuggestDocumentation(topic, url string) string {
	if url != "" {
		return fmt.Sprintf("For help with %s, see: %s", topic, url)
	}
	return fmt.Sprintf("For help with %s, check the Gas Town documentation.", topic)
}

// EnrichErrorWithHint adds an appropriate hint to an error based on its type and content.
func EnrichErrorWithHint(err error) error {
	if err == nil {
		return nil
	}

	// If it's already our Error type with a hint, return as-is
	var e *Error
	if As(err, &e) && e.Hint != "" {
		return err
	}

	// Check for specific error patterns and add hints
	errStr := err.Error()

	// Command not found errors
	if strings.Contains(errStr, "executable file not found") || strings.Contains(errStr, "command not found") {
		if strings.Contains(errStr, "bd") {
			return Transient("command", err).WithHint(HintBeadsNotInstalled)
		}
		if strings.Contains(errStr, "tmux") {
			return System("command", err).WithHint(HintTmuxNotFound)
		}
	}

	// Git errors
	if strings.Contains(errStr, "git") {
		if strings.Contains(errStr, "push") {
			return Transient("git.push", err).WithHint(HintGitPushFailed)
		}
		if strings.Contains(errStr, "pull") {
			return Transient("git.pull", err).WithHint(HintGitPullFailed)
		}
		if strings.Contains(errStr, "conflict") {
			return User("git.conflict", errStr).WithHint(HintGitConflict)
		}
	}

	// Merge conflicts (may not have "git" in error message)
	if strings.Contains(errStr, "conflict") || strings.Contains(errStr, "merge") {
		return User("git.conflict", errStr).WithHint(HintGitConflict)
	}

	// Network errors
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "no route to host") {
		return Transient("network", err).WithHint(HintNetworkError)
	}
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return Transient("network", err).WithHint(HintTimeoutError)
	}

	// File errors
	if strings.Contains(errStr, "no such file") || strings.Contains(errStr, "not found") {
		return User("file", errStr).WithHint(HintFileNotFound)
	}
	if strings.Contains(errStr, "permission denied") {
		return User("file", errStr).WithHint(HintPermissionDenied)
	}

	// Beads errors
	if strings.Contains(errStr, "beads") || strings.Contains(errStr, "bd") {
		if strings.Contains(errStr, "not found") {
			return User("beads", errStr).WithHint(HintBeadsNotFound)
		}
	}

	// Return original error if no pattern matched
	return err
}
