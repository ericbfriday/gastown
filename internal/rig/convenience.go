package rig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/config"
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
		return nil, fmt.Errorf("loading rigs config: %w", err)
	}

	// Get rig name from path
	rigName := filepath.Base(rigPath)

	// Get rig entry
	entry, ok := rigsConfig.Rigs[rigName]
	if !ok {
		return nil, fmt.Errorf("rig %q not found in config", rigName)
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
		return nil, err
	}
	return FindRigFromPath(cwd)
}

// FindRigFromPath finds the rig containing the given path.
func FindRigFromPath(path string) (*Rig, error) {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
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
			return nil, fmt.Errorf("not in a town: no config/ directory found")
		}
		current = parent
	}

	// Load rigs config
	rigsConfig, err := config.LoadRigsConfig(townRoot)
	if err != nil {
		return nil, fmt.Errorf("loading rigs config: %w", err)
	}

	// Create manager to discover rigs
	g := git.NewGit(townRoot)
	mgr := NewManager(townRoot, rigsConfig, g)
	rigs, err := mgr.DiscoverRigs()
	if err != nil {
		return nil, err
	}

	// Find rig containing the given path
	for _, r := range rigs {
		if strings.HasPrefix(absPath, r.Path+string(filepath.Separator)) || absPath == r.Path {
			return r, nil
		}
	}

	return nil, fmt.Errorf("no rig found containing path: %s", absPath)
}
