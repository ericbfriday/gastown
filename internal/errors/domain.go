package errors

import "fmt"

// SessionError represents errors related to session operations.
type SessionError struct {
	Op      string
	Polecat string
	Session string
	Err     error
}

func (e *SessionError) Error() string {
	if e.Polecat != "" {
		return fmt.Sprintf("session %s for polecat %s: %v", e.Op, e.Polecat, e.Err)
	}
	if e.Session != "" {
		return fmt.Sprintf("session %s for %s: %v", e.Op, e.Session, e.Err)
	}
	return fmt.Sprintf("session %s: %v", e.Op, e.Err)
}

func (e *SessionError) Unwrap() error {
	return e.Err
}

// NewSessionError creates a new SessionError.
func NewSessionError(op, polecat string, err error) *SessionError {
	return &SessionError{
		Op:      op,
		Polecat: polecat,
		Err:     err,
	}
}

// GitError represents errors related to git operations.
type GitError struct {
	Op         string
	Repository string
	Branch     string
	Err        error
}

func (e *GitError) Error() string {
	if e.Branch != "" {
		return fmt.Sprintf("git %s in %s on branch %s: %v", e.Op, e.Repository, e.Branch, e.Err)
	}
	if e.Repository != "" {
		return fmt.Sprintf("git %s in %s: %v", e.Op, e.Repository, e.Err)
	}
	return fmt.Sprintf("git %s: %v", e.Op, e.Err)
}

func (e *GitError) Unwrap() error {
	return e.Err
}

// NewGitError creates a new GitError.
func NewGitError(op, repo, branch string, err error) *GitError {
	return &GitError{
		Op:         op,
		Repository: repo,
		Branch:     branch,
		Err:        err,
	}
}

// BeadsError represents errors related to beads operations.
type BeadsError struct {
	Op     string
	IssueID string
	Err    error
}

func (e *BeadsError) Error() string {
	if e.IssueID != "" {
		return fmt.Sprintf("beads %s for issue %s: %v", e.Op, e.IssueID, e.Err)
	}
	return fmt.Sprintf("beads %s: %v", e.Op, e.Err)
}

func (e *BeadsError) Unwrap() error {
	return e.Err
}

// NewBeadsError creates a new BeadsError.
func NewBeadsError(op, issueID string, err error) *BeadsError {
	return &BeadsError{
		Op:      op,
		IssueID: issueID,
		Err:     err,
	}
}

// RefineryError represents errors related to refinery operations.
type RefineryError struct {
	Op    string
	Rig   string
	MRID  string
	Err   error
}

func (e *RefineryError) Error() string {
	if e.MRID != "" {
		return fmt.Sprintf("refinery %s for MR %s: %v", e.Op, e.MRID, e.Err)
	}
	if e.Rig != "" {
		return fmt.Sprintf("refinery %s for rig %s: %v", e.Op, e.Rig, e.Err)
	}
	return fmt.Sprintf("refinery %s: %v", e.Op, e.Err)
}

func (e *RefineryError) Unwrap() error {
	return e.Err
}

// NewRefineryError creates a new RefineryError.
func NewRefineryError(op, rig, mrID string, err error) *RefineryError {
	return &RefineryError{
		Op:   op,
		Rig:  rig,
		MRID: mrID,
		Err:  err,
	}
}

// NetworkError represents errors related to network operations.
type NetworkError struct {
	Op       string
	Host     string
	Err      error
}

func (e *NetworkError) Error() string {
	if e.Host != "" {
		return fmt.Sprintf("network %s to %s: %v", e.Op, e.Host, e.Err)
	}
	return fmt.Sprintf("network %s: %v", e.Op, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// NewNetworkError creates a new NetworkError.
func NewNetworkError(op, host string, err error) *NetworkError {
	return &NetworkError{
		Op:   op,
		Host: host,
		Err:  err,
	}
}

// FileError represents errors related to file operations.
type FileError struct {
	Op   string
	Path string
	Err  error
}

func (e *FileError) Error() string {
	if e.Path != "" {
		return fmt.Sprintf("file %s %s: %v", e.Op, e.Path, e.Err)
	}
	return fmt.Sprintf("file %s: %v", e.Op, e.Err)
}

func (e *FileError) Unwrap() error {
	return e.Err
}

// NewFileError creates a new FileError.
func NewFileError(op, path string, err error) *FileError {
	return &FileError{
		Op:   op,
		Path: path,
		Err:  err,
	}
}

// ConfigError represents errors related to configuration.
type ConfigError struct {
	Op     string
	File   string
	Field  string
	Err    error
}

func (e *ConfigError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("config %s in %s (field: %s): %v", e.Op, e.File, e.Field, e.Err)
	}
	if e.File != "" {
		return fmt.Sprintf("config %s in %s: %v", e.Op, e.File, e.Err)
	}
	return fmt.Sprintf("config %s: %v", e.Op, e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new ConfigError.
func NewConfigError(op, file, field string, err error) *ConfigError {
	return &ConfigError{
		Op:    op,
		File:  file,
		Field: field,
		Err:   err,
	}
}
