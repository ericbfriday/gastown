// Package beads provides routing helpers for prefix-based beads resolution.
package beads

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/config"
	"github.com/steveyegge/gastown/internal/filelock"
)

// Route represents a prefix-to-path routing rule.
// This mirrors the structure in bd's internal/routing package.
type Route struct {
	Prefix string `json:"prefix"` // Issue ID prefix (e.g., "gt-")
	Path   string `json:"path"`   // Relative path to .beads directory from town root
}

// RoutesFileName is the name of the routes configuration file.
const RoutesFileName = "routes.jsonl"

// LoadRoutes loads routes from routes.jsonl in the given beads directory.
// Returns an empty slice if the file doesn't exist.
func LoadRoutes(beadsDir string) ([]Route, error) {
	routesPath := filepath.Join(beadsDir, RoutesFileName)
	var routes []Route

	err := filelock.WithReadLock(routesPath, func() error {
		return loadRoutesUnsafe(routesPath, &routes)
	})

	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No routes file is not an error
		}
		return nil, err
	}

	return routes, nil
}

// loadRoutesUnsafe loads routes without locking (internal helper).
func loadRoutesUnsafe(routesPath string, routes *[]Route) error {
	file, err := os.Open(routesPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		var route Route
		if err := json.Unmarshal([]byte(line), &route); err != nil {
			continue // Skip malformed lines
		}
		if route.Prefix != "" && route.Path != "" {
			*routes = append(*routes, route)
		}
	}

	return scanner.Err()
}

// AppendRoute appends a route to routes.jsonl in the town's beads directory.
// If the prefix already exists, it updates the path.
func AppendRoute(townRoot string, route Route) error {
	beadsDir := filepath.Join(townRoot, ".beads")
	return AppendRouteToDir(beadsDir, route)
}

// AppendRouteToDir appends a route to routes.jsonl in the given beads directory.
// If the prefix already exists, it updates the path.
func AppendRouteToDir(beadsDir string, route Route) error {
	routesPath := filepath.Join(beadsDir, RoutesFileName)

	// Entire read-modify-write cycle must be atomic
	return filelock.WithWriteLock(routesPath, func() error {
		// Load existing routes (without lock)
		var routes []Route
		if err := loadRoutesUnsafe(routesPath, &routes); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("loading routes: %w", err)
		}

		// Check if prefix already exists
		found := false
		for i, r := range routes {
			if r.Prefix == route.Prefix {
				routes[i].Path = route.Path
				found = true
				break
			}
		}

		if !found {
			routes = append(routes, route)
		}

		// Write back (without lock)
		return writeRoutesUnsafe(routesPath, routes)
	})
}

// RemoveRoute removes a route by prefix from routes.jsonl.
func RemoveRoute(townRoot string, prefix string) error {
	beadsDir := filepath.Join(townRoot, ".beads")
	routesPath := filepath.Join(beadsDir, RoutesFileName)

	// Entire read-modify-write cycle must be atomic
	return filelock.WithWriteLock(routesPath, func() error {
		// Load existing routes (without lock)
		var routes []Route
		if err := loadRoutesUnsafe(routesPath, &routes); err != nil {
			return fmt.Errorf("loading routes: %w", err)
		}

		// Filter out the prefix
		var filtered []Route
		for _, r := range routes {
			if r.Prefix != prefix {
				filtered = append(filtered, r)
			}
		}

		// Write back (without lock)
		return writeRoutesUnsafe(routesPath, filtered)
	})
}

// WriteRoutes writes routes to routes.jsonl, overwriting existing content.
func WriteRoutes(beadsDir string, routes []Route) error {
	// Ensure beads directory exists
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		return fmt.Errorf("creating beads directory: %w", err)
	}

	routesPath := filepath.Join(beadsDir, RoutesFileName)

	return filelock.WithWriteLock(routesPath, func() error {
		return writeRoutesUnsafe(routesPath, routes)
	})
}

// writeRoutesUnsafe writes routes without locking, using atomic tmp+rename pattern.
func writeRoutesUnsafe(routesPath string, routes []Route) error {
	// Ensure parent directory exists
	beadsDir := filepath.Dir(routesPath)
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		return fmt.Errorf("creating beads directory: %w", err)
	}

	// Create temp file with unique name using CreateTemp for safety
	// Use a pattern that's recognizable but unique per call
	tmpFile, err := os.CreateTemp(beadsDir, ".routes.jsonl.tmp.")
	if err != nil {
		return fmt.Errorf("creating temp routes file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Ensure cleanup on error
	var writeErr error
	defer func() {
		if writeErr != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Write all routes
	for _, r := range routes {
		data, err := json.Marshal(r)
		if err != nil {
			writeErr = fmt.Errorf("marshaling route: %w", err)
			return writeErr
		}
		if _, err := tmpFile.Write(data); err != nil {
			writeErr = fmt.Errorf("writing route: %w", err)
			return writeErr
		}
		if _, err := tmpFile.WriteString("\n"); err != nil {
			writeErr = fmt.Errorf("writing newline: %w", err)
			return writeErr
		}
	}

	// Close file
	if err := tmpFile.Close(); err != nil {
		writeErr = fmt.Errorf("closing temp file: %w", err)
		return writeErr
	}

	// Atomic rename
	if err := os.Rename(tmpPath, routesPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// GetTownBeadsPath returns the path to town-level beads directory.
// Town beads store hq-* prefixed issues including Mayor, Deacon, and role beads.
// The townRoot should be the Gas Town root directory (e.g., ~/gt).
func GetTownBeadsPath(townRoot string) string {
	return filepath.Join(townRoot, ".beads")
}

// GetPrefixForRig returns the beads prefix for a given rig name.
// The prefix is returned without the trailing hyphen (e.g., "bd" not "bd-").
// If the rig is not found in routes, returns "gt" as the default.
// The townRoot should be the Gas Town root directory (e.g., ~/gt).
func GetPrefixForRig(townRoot, rigName string) string {
	beadsDir := filepath.Join(townRoot, ".beads")
	routes, err := LoadRoutes(beadsDir)
	if err != nil || routes == nil {
		return config.GetRigPrefix(townRoot, rigName)
	}

	// Look for a route where the path starts with the rig name
	// Routes paths are like "gastown/mayor/rig" or "beads/mayor/rig"
	for _, r := range routes {
		parts := strings.SplitN(r.Path, "/", 2)
		if len(parts) > 0 && parts[0] == rigName {
			// Return prefix without trailing hyphen
			return strings.TrimSuffix(r.Prefix, "-")
		}
	}

	return config.GetRigPrefix(townRoot, rigName)
}

// FindConflictingPrefixes checks for duplicate prefixes in routes.
// Returns a map of prefix -> list of paths that use it.
func FindConflictingPrefixes(beadsDir string) (map[string][]string, error) {
	routes, err := LoadRoutes(beadsDir)
	if err != nil {
		return nil, err
	}

	// Group by prefix
	prefixPaths := make(map[string][]string)
	for _, r := range routes {
		prefixPaths[r.Prefix] = append(prefixPaths[r.Prefix], r.Path)
	}

	// Filter to only conflicts (more than one path per prefix)
	conflicts := make(map[string][]string)
	for prefix, paths := range prefixPaths {
		if len(paths) > 1 {
			conflicts[prefix] = paths
		}
	}

	return conflicts, nil
}

// ExtractPrefix extracts the prefix from a bead ID.
// For example, "ap-qtsup.16" returns "ap-", "hq-cv-abc" returns "hq-".
// Returns empty string if no valid prefix found (empty input, no hyphen,
// or hyphen at position 0 which would indicate an invalid prefix).
func ExtractPrefix(beadID string) string {
	if beadID == "" {
		return ""
	}

	idx := strings.Index(beadID, "-")
	if idx <= 0 {
		return ""
	}

	return beadID[:idx+1]
}

// GetRigPathForPrefix returns the rig path for a given bead ID prefix.
// The townRoot should be the Gas Town root directory (e.g., ~/gt).
// Returns the full absolute path to the rig directory, or empty string if not found.
// For town-level beads (path="."), returns townRoot.
func GetRigPathForPrefix(townRoot, prefix string) string {
	beadsDir := filepath.Join(townRoot, ".beads")
	routes, err := LoadRoutes(beadsDir)
	if err != nil || routes == nil {
		return ""
	}

	for _, r := range routes {
		if r.Prefix == prefix {
			if r.Path == "." {
				return townRoot // Town-level beads
			}
			return filepath.Join(townRoot, r.Path)
		}
	}

	return ""
}

// ResolveHookDir determines the directory for running bd update on a bead.
// Since bd update doesn't support routing or redirects, we must resolve the
// actual rig directory from the bead's prefix. hookWorkDir is only used as
// a fallback if prefix resolution fails.
func ResolveHookDir(townRoot, beadID, hookWorkDir string) string {
	// Always try prefix resolution first - bd update needs the actual rig dir
	prefix := ExtractPrefix(beadID)
	if rigPath := GetRigPathForPrefix(townRoot, prefix); rigPath != "" {
		return rigPath
	}
	// Fallback to hookWorkDir if provided
	if hookWorkDir != "" {
		return hookWorkDir
	}
	return townRoot
}
