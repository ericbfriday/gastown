package rig

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/errors"
	"github.com/steveyegge/gastown/internal/git"
)

// Load loads a rig from the given path.
// This is a convenience function that creates a temporary Manager to load the rig.
func Load(rigPath string) (*Rig, error) {
	// Get town root by walking up from rig path
	townRoot := filepath.Dir(rigPath)

	// Load town config to get rig entry
	rigsConfig, err := config.LoadRigsConfig(townRoot)
	if err != nil {
		return nil, errors.Permanent("rig.LoadConfigFailed", err).
			WithContext("town_root", townRoot).
			WithHint("Verify the town configuration exists and is valid")
	}

	// Get rig name from path
	rigName := filepath.Base(rigPath)

	// Get rig entry
	entry, ok := rigsConfig.Rigs[rigName]
	if !ok {
		return nil, errors.Permanent("rig.NotFoundInConfig", nil).
			WithContext("rig_name", rigName).
			WithHint("Use 'gt rig list' to see available rigs")
	}

	// Create manager and load rig
	g := git.NewGit(townRoot)
	mgr := NewManager(townRoot, rigsConfig, g)
	return mgr.loadRig(rigName, entry)
}

// FindFromCwd finds the rig containing the current working directory.
func FindFromCwd() (*Rig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, errors.System("rig.GetCwdFailed", err).
			WithHint("Unable to determine current working directory")
	}
	return FindRigFromPath(cwd)
}

// FindRigFromPath finds the rig containing the given path.
func FindRigFromPath(path string) (*Rig, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.System("rig.AbsPathFailed", err).
			WithContext("path", path).
			WithHint("Unable to resolve absolute path")
	}

	// Walk up to find town root (directory containing config/ dir)
	townRoot := ""
	current := absPath
	for {
		configDir := filepath.Join(current, "config")
		if info, err := os.Stat(configDir); err == nil && info.IsDir() {
			townRoot = current
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root without finding town
			return nil, errors.User("rig.NotInTown", "not in a town").
				WithContext("path", absPath).
				WithHint("Navigate to a Gas Town directory (containing config/) or initialize a new town with 'gt install'")
		}
		current = parent
	}

	// Load rigs config
	rigsConfig, err := config.LoadRigsConfig(townRoot)
	if err != nil {
		return nil, errors.Permanent("rig.LoadConfigFailed", err).
			WithContext("town_root", townRoot).
			WithHint("Verify the town configuration exists and is valid")
	}

	// Create manager to discover rigs
	g := git.NewGit(townRoot)
	mgr := NewManager(townRoot, rigsConfig, g)
	rigs, err := mgr.DiscoverRigs()
	if err != nil {
		return nil, err // Already wrapped by DiscoverRigs
	}

	// Find rig containing the given path
	for _, r := range rigs {
		if strings.HasPrefix(absPath, r.Path+string(filepath.Separator)) || absPath == r.Path {
			return r, nil
		}
	}

	return nil, errors.Permanent("rig.NoRigFound", nil).
		WithContext("path", absPath).
		WithHint("Navigate to a rig directory or use 'gt rig list' to see available rigs")
}
