package rig

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/errors"
)

// CopyOverlay copies files from <rigPath>/.runtime/overlay/ to the destination path.
// This allows storing gitignored files (like .env) that services need at their root.
// The overlay is copied non-recursively - only files, not subdirectories.
// File permissions from the source are preserved.
//
// Structure:
//
//	rig/
//	  .runtime/
//	    overlay/
//	      .env          <- Copied to destPath
//	      config.json   <- Copied to destPath
//
// Returns nil if the overlay directory doesn't exist (nothing to copy).
// Individual file copy failures are logged as warnings but don't stop the process.
func CopyOverlay(rigPath, destPath string) error {
	overlayDir := filepath.Join(rigPath, ".runtime", "overlay")

	// Check if overlay directory exists
	entries, err := os.ReadDir(overlayDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No overlay directory - not an error, just nothing to copy
			return nil
		}
		return errors.System("rig.ReadOverlayDirFailed", err).
			WithContext("overlay_dir", overlayDir).
			WithHint("Check file system permissions")
	}

	// Copy each file (not directories) from overlay to destination
	for _, entry := range entries {
		if entry.IsDir() {
			// Skip subdirectories - only copy files at overlay root
			continue
		}

		srcPath := filepath.Join(overlayDir, entry.Name())
		dstPath := filepath.Join(destPath, entry.Name())

		if err := copyFilePreserveMode(srcPath, dstPath); err != nil {
			// Log warning but continue - don't fail spawn for overlay issues
			fmt.Printf("Warning: could not copy overlay file %s: %v\n", entry.Name(), err)
			continue
		}
	}

	return nil
}

// EnsureGitignorePatterns ensures the .gitignore has required Gas Town patterns.
// This is called after cloning to add patterns that may be missing from the source repo.
func EnsureGitignorePatterns(worktreePath string) error {
	gitignorePath := filepath.Join(worktreePath, ".gitignore")

	// Required patterns for Gas Town worktrees
	requiredPatterns := []string{
		".runtime/",
		".claude/",
		".logs/",
		".beads/",
	}

	// Read existing gitignore content
	var existingContent string
	data, err := errors.RetryFunc(func() ([]byte, error) {
		return os.ReadFile(gitignorePath)
	}, errors.FileIORetryConfig())
	if err == nil {
		existingContent = string(data)
	}

	// Find missing patterns
	var missing []string
	for _, pattern := range requiredPatterns {
		// Check various forms: .runtime, .runtime/, /.runtime, etc.
		found := false
		for _, line := range strings.Split(existingContent, "\n") {
			line = strings.TrimSpace(line)
			if line == pattern || line == strings.TrimSuffix(pattern, "/") ||
				line == "/"+pattern || line == "/"+strings.TrimSuffix(pattern, "/") {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, pattern)
		}
	}

	if len(missing) == 0 {
		return nil // All patterns present
	}

	// Append missing patterns
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.System("rig.OpenGitignoreFailed", err).
			WithContext("gitignore_path", gitignorePath).
			WithHint("Check file system permissions")
	}
	defer f.Close()

	// Add header if appending to existing file
	if existingContent != "" && !strings.HasSuffix(existingContent, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return errors.System("rig.WriteGitignoreFailed", err).
				WithContext("gitignore_path", gitignorePath).
				WithHint("Check file system permissions and disk space")
		}
	}
	if existingContent != "" {
		if _, err := f.WriteString("\n# Gas Town (added by gt)\n"); err != nil {
			return errors.System("rig.WriteGitignoreFailed", err).
				WithContext("gitignore_path", gitignorePath).
				WithHint("Check file system permissions and disk space")
		}
	}

	for _, pattern := range missing {
		if _, err := f.WriteString(pattern + "\n"); err != nil {
			return errors.System("rig.WriteGitignoreFailed", err).
				WithContext("gitignore_path", gitignorePath).
				WithContext("pattern", pattern).
				WithHint("Check file system permissions and disk space")
		}
	}

	return nil
}

// copyFilePreserveMode copies a file from src to dst, preserving the source file's permissions.
func copyFilePreserveMode(src, dst string) error {
	// Get source file info for permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return errors.System("rig.StatSourceFailed", err).
			WithContext("source", src).
			WithHint("Check that the source file exists and is readable")
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return errors.System("rig.OpenSourceFailed", err).
			WithContext("source", src).
			WithHint("Check file system permissions")
	}
	defer srcFile.Close()

	// Create destination file with same permissions
	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode().Perm())
	if err != nil {
		return errors.System("rig.CreateDestinationFailed", err).
			WithContext("destination", dst).
			WithHint("Check file system permissions and disk space")
	}
	defer dstFile.Close()

	// Copy contents with retry for transient failures
	if err := errors.WithFileIORetry(func() error {
		_, err := io.Copy(dstFile, srcFile)
		return err
	}); err != nil {
		return errors.System("rig.CopyContentsFailed", err).
			WithContext("source", src).
			WithContext("destination", dst).
			WithHint("Check disk space and file system integrity")
	}

	return nil
}
