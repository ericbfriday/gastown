package mail

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/steveyegge/gastown/internal/errors"
)

// bdError represents an error from running a bd command.
// It wraps the underlying error and includes the stderr output for inspection.
type bdError struct {
	Err    error
	Stderr string
	Args   []string
}

// Error implements the error interface.
func (e *bdError) Error() string {
	if e.Stderr != "" {
		return e.Stderr
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "unknown bd error"
}

// Unwrap returns the underlying error for errors.Is/As compatibility.
func (e *bdError) Unwrap() error {
	return e.Err
}

// ContainsError checks if the stderr message contains the given substring.
func (e *bdError) ContainsError(substr string) bool {
	return strings.Contains(e.Stderr, substr)
}

// ToEnhancedError converts bdError to an enhanced error with categorization.
func (e *bdError) ToEnhancedError() error {
	stderr := strings.ToLower(e.Stderr)
	cmdStr := strings.Join(e.Args, " ")

	// Categorize based on error content
	if strings.Contains(stderr, "not found") || strings.Contains(stderr, "no such") {
		return errors.Permanent("mail.BdNotFound", e.Err).
			WithContext("command", cmdStr).
			WithContext("stderr", e.Stderr).
			WithHint("Check that the resource exists: bd show <id>")
	}

	if strings.Contains(stderr, "command not found") || strings.Contains(stderr, "executable file not found") {
		return errors.System("mail.BdNotInstalled", e.Err).
			WithContext("command", cmdStr).
			WithHint("Install beads with: brew install beads")
	}

	if strings.Contains(stderr, "timeout") || strings.Contains(stderr, "connection") {
		return errors.Transient("mail.BdTimeout", e.Err).
			WithContext("command", cmdStr).
			WithContext("stderr", e.Stderr).
			WithHint("Network or I/O issue - retry the operation")
	}

	if strings.Contains(stderr, "permission denied") || strings.Contains(stderr, "access denied") {
		return errors.System("mail.BdPermission", e.Err).
			WithContext("command", cmdStr).
			WithContext("stderr", e.Stderr).
			WithHint("Check file permissions and user access")
	}

	// Default to transient for unknown errors
	return errors.Transient("mail.BdCommand", e.Err).
		WithContext("command", cmdStr).
		WithContext("stderr", e.Stderr).
		WithHint("Check beads status: bd --version")
}

// runBdCommand executes a bd command with proper environment setup.
// workDir is the directory to run the command in.
// beadsDir is the BEADS_DIR environment variable value.
// extraEnv contains additional environment variables to set (e.g., "BD_IDENTITY=...").
// Returns stdout bytes on success, or a *bdError on failure.
func runBdCommand(args []string, workDir, beadsDir string, extraEnv ...string) ([]byte, error) {
	cmd := exec.Command("bd", args...) //nolint:gosec // G204: bd is a trusted internal tool
	cmd.Dir = workDir

	env := append(cmd.Environ(), "BEADS_DIR="+beadsDir)
	env = append(env, extraEnv...)
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, &bdError{
			Err:    err,
			Stderr: strings.TrimSpace(stderr.String()),
		}
	}

	return stdout.Bytes(), nil
}

// RunBdCommand is the exported version of runBdCommand for use by other packages.
func RunBdCommand(args []string, workDir, beadsDir string, extraEnv ...string) ([]byte, error) {
	cmd := exec.Command("bd", args...) //nolint:gosec // G204: bd is a trusted internal tool
	cmd.Dir = workDir

	env := append(cmd.Environ(), "BEADS_DIR="+beadsDir)
	env = append(env, extraEnv...)
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		bdErr := &bdError{
			Err:    err,
			Stderr: strings.TrimSpace(stderr.String()),
			Args:   args,
		}
		return nil, bdErr.ToEnhancedError()
	}

	return stdout.Bytes(), nil
}
