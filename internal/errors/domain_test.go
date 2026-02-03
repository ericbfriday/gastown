package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestSessionError(t *testing.T) {
	baseErr := errors.New("tmux session not found")

	tests := []struct {
		name     string
		err      *SessionError
		contains []string
	}{
		{
			name: "with polecat",
			err: &SessionError{
				Op:      "start",
				Polecat: "Toast",
				Err:     baseErr,
			},
			contains: []string{"start", "Toast", "tmux session not found"},
		},
		{
			name: "with session ID",
			err: &SessionError{
				Op:      "stop",
				Session: "gt-myrig-refinery",
				Err:     baseErr,
			},
			contains: []string{"stop", "gt-myrig-refinery", "tmux session not found"},
		},
		{
			name: "minimal",
			err: &SessionError{
				Op:  "check",
				Err: baseErr,
			},
			contains: []string{"check", "tmux session not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("SessionError should contain %q, got %q", s, errStr)
				}
			}
		})
	}
}

func TestSessionError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	sessionErr := NewSessionError("test", "polecat", baseErr)

	if !errors.Is(sessionErr, baseErr) {
		t.Error("SessionError should wrap base error")
	}
}

func TestGitError(t *testing.T) {
	baseErr := errors.New("merge conflict")

	tests := []struct {
		name     string
		err      *GitError
		contains []string
	}{
		{
			name: "with branch",
			err: &GitError{
				Op:         "merge",
				Repository: "/path/to/repo",
				Branch:     "feature/new-feature",
				Err:        baseErr,
			},
			contains: []string{"merge", "/path/to/repo", "feature/new-feature", "merge conflict"},
		},
		{
			name: "with repository only",
			err: &GitError{
				Op:         "fetch",
				Repository: "/path/to/repo",
				Err:        baseErr,
			},
			contains: []string{"fetch", "/path/to/repo", "merge conflict"},
		},
		{
			name: "minimal",
			err: &GitError{
				Op:  "status",
				Err: baseErr,
			},
			contains: []string{"status", "merge conflict"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("GitError should contain %q, got %q", s, errStr)
				}
			}
		})
	}
}

func TestGitError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	gitErr := NewGitError("test", "repo", "branch", baseErr)

	if !errors.Is(gitErr, baseErr) {
		t.Error("GitError should wrap base error")
	}
}

func TestBeadsError(t *testing.T) {
	baseErr := errors.New("issue not found")

	tests := []struct {
		name     string
		err      *BeadsError
		contains []string
	}{
		{
			name: "with issue ID",
			err: &BeadsError{
				Op:      "show",
				IssueID: "gt-123",
				Err:     baseErr,
			},
			contains: []string{"show", "gt-123", "issue not found"},
		},
		{
			name: "minimal",
			err: &BeadsError{
				Op:  "list",
				Err: baseErr,
			},
			contains: []string{"list", "issue not found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("BeadsError should contain %q, got %q", s, errStr)
				}
			}
		})
	}
}

func TestBeadsError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	beadsErr := NewBeadsError("test", "issue-1", baseErr)

	if !errors.Is(beadsErr, baseErr) {
		t.Error("BeadsError should wrap base error")
	}
}

func TestRefineryError(t *testing.T) {
	baseErr := errors.New("merge failed")

	tests := []struct {
		name     string
		err      *RefineryError
		contains []string
	}{
		{
			name: "with MR ID",
			err: &RefineryError{
				Op:   "process",
				Rig:  "myrig",
				MRID: "mr-456",
				Err:  baseErr,
			},
			contains: []string{"process", "mr-456", "merge failed"},
		},
		{
			name: "with rig only",
			err: &RefineryError{
				Op:  "start",
				Rig: "myrig",
				Err: baseErr,
			},
			contains: []string{"start", "myrig", "merge failed"},
		},
		{
			name: "minimal",
			err: &RefineryError{
				Op:  "queue",
				Err: baseErr,
			},
			contains: []string{"queue", "merge failed"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("RefineryError should contain %q, got %q", s, errStr)
				}
			}
		})
	}
}

func TestRefineryError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	refineryErr := NewRefineryError("test", "rig", "mr-1", baseErr)

	if !errors.Is(refineryErr, baseErr) {
		t.Error("RefineryError should wrap base error")
	}
}

func TestNetworkError(t *testing.T) {
	baseErr := errors.New("connection timeout")

	tests := []struct {
		name     string
		err      *NetworkError
		contains []string
	}{
		{
			name: "with host",
			err: &NetworkError{
				Op:   "connect",
				Host: "github.com",
				Err:  baseErr,
			},
			contains: []string{"connect", "github.com", "connection timeout"},
		},
		{
			name: "minimal",
			err: &NetworkError{
				Op:  "fetch",
				Err: baseErr,
			},
			contains: []string{"fetch", "connection timeout"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("NetworkError should contain %q, got %q", s, errStr)
				}
			}
		})
	}
}

func TestNetworkError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	netErr := NewNetworkError("test", "host", baseErr)

	if !errors.Is(netErr, baseErr) {
		t.Error("NetworkError should wrap base error")
	}
}

func TestFileError(t *testing.T) {
	baseErr := errors.New("permission denied")

	tests := []struct {
		name     string
		err      *FileError
		contains []string
	}{
		{
			name: "with path",
			err: &FileError{
				Op:   "read",
				Path: "/path/to/file.txt",
				Err:  baseErr,
			},
			contains: []string{"read", "/path/to/file.txt", "permission denied"},
		},
		{
			name: "minimal",
			err: &FileError{
				Op:  "write",
				Err: baseErr,
			},
			contains: []string{"write", "permission denied"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("FileError should contain %q, got %q", s, errStr)
				}
			}
		})
	}
}

func TestFileError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	fileErr := NewFileError("test", "path", baseErr)

	if !errors.Is(fileErr, baseErr) {
		t.Error("FileError should wrap base error")
	}
}

func TestConfigError(t *testing.T) {
	baseErr := errors.New("invalid value")

	tests := []struct {
		name     string
		err      *ConfigError
		contains []string
	}{
		{
			name: "with field",
			err: &ConfigError{
				Op:    "parse",
				File:  "config.json",
				Field: "timeout",
				Err:   baseErr,
			},
			contains: []string{"parse", "config.json", "timeout", "invalid value"},
		},
		{
			name: "with file only",
			err: &ConfigError{
				Op:   "load",
				File: "config.json",
				Err:  baseErr,
			},
			contains: []string{"load", "config.json", "invalid value"},
		},
		{
			name: "minimal",
			err: &ConfigError{
				Op:  "validate",
				Err: baseErr,
			},
			contains: []string{"validate", "invalid value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("ConfigError should contain %q, got %q", s, errStr)
				}
			}
		})
	}
}

func TestConfigError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	configErr := NewConfigError("test", "file", "field", baseErr)

	if !errors.Is(configErr, baseErr) {
		t.Error("ConfigError should wrap base error")
	}
}
