package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/ui"
)

var namesCmd = &cobra.Command{
	Use:     "names",
	GroupID: GroupWorkspace,
	Short:   "Manage polecat naming pool",
	Long: `Manage the polecat naming pool for automatic name allocation.

The naming pool tracks available and in-use names for polecats. When spawning
polecats without explicit names, the pool provides themed names from the
configured theme (mad-max, minerals, wasteland, etc.).

Examples:
  gt names list                # Show available and in-use names
  gt names add obsidian        # Add custom name to pool
  gt names remove obsidian     # Remove name from pool (if not in use)
  gt names reserve witness     # Reserve name for specific use
  gt names stats               # Show pool statistics`,
	RunE: runNamesList,
}

var namesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available and in-use names",
	Long: `List all names in the naming pool with their status.

Shows:
- Available names (ready for allocation)
- In-use names (currently assigned to polecats)
- Reserved names (infrastructure agents)`,
	RunE: runNamesList,
}

var namesAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add custom name to the pool",
	Long: `Add a custom name to the naming pool.

The name will be validated and checked for conflicts with:
- Existing names in the pool
- Reserved infrastructure agent names (witness, mayor, deacon, refinery)

Custom names take precedence over themed names during allocation.`,
	Args: cobra.ExactArgs(1),
	RunE: runNamesAdd,
}

var namesRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove name from pool",
	Long: `Remove a custom name from the pool.

Safety checks:
- Only custom names can be removed (themed names are immutable)
- Name must not be currently in use by a polecat
- Reserved names cannot be removed

Use --force to bypass the in-use check.`,
	Args: cobra.ExactArgs(1),
	RunE: runNamesRemove,
}

var namesReserveCmd = &cobra.Command{
	Use:   "reserve <name>",
	Short: "Reserve name for specific use",
	Long: `Reserve a name to prevent it from being allocated to polecats.

Reserved names are typically used for infrastructure agents (witness, mayor, etc.)
or special-purpose agents that need stable identities.

This command adds the name to the reserved names list in the rig configuration.`,
	Args: cobra.ExactArgs(1),
	RunE: runNamesReserve,
}

var namesStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show pool statistics",
	Long: `Display statistics about the naming pool.

Shows:
- Total pool size
- Available capacity
- In-use count
- Reserved count
- Theme information
- Usage patterns`,
	RunE: runNamesStats,
}

var (
	namesRemoveForce bool
)

func init() {
	rootCmd.AddCommand(namesCmd)
	namesCmd.AddCommand(namesListCmd)
	namesCmd.AddCommand(namesAddCmd)
	namesCmd.AddCommand(namesRemoveCmd)
	namesCmd.AddCommand(namesReserveCmd)
	namesCmd.AddCommand(namesStatsCmd)

	namesRemoveCmd.Flags().BoolVar(&namesRemoveForce, "force", false, "Remove even if name is in use")
}

func runNamesList(cmd *cobra.Command, args []string) error {
	rigPath, err := detectRigPath()
	if err != nil {
		return err
	}

	rigName := filepath.Base(rigPath)
	pool, err := loadNamePool(rigPath, rigName)
	if err != nil {
		return fmt.Errorf("loading name pool: %w", err)
	}

	// Get available and in-use names
	availableNames := pool.AvailableNames()
	inUseNames := pool.ActiveNames()

	// Display theme info
	ui.PrintHeading("Name Pool: %s", rigName)
	fmt.Printf("Theme: %s\n", pool.GetTheme())
	fmt.Println()

	// Display reserved infrastructure names
	if len(polecat.ReservedInfraAgentNames) > 0 {
		reserved := make([]string, 0, len(polecat.ReservedInfraAgentNames))
		for name := range polecat.ReservedInfraAgentNames {
			reserved = append(reserved, name)
		}
		sort.Strings(reserved)
		ui.PrintSection("Reserved Names (Infrastructure)")
		for _, name := range reserved {
			fmt.Printf("  %s\n", ui.ColorReserved(name))
		}
		fmt.Println()
	}

	// Display in-use names
	if len(inUseNames) > 0 {
		ui.PrintSection("In Use (%d)", len(inUseNames))
		for _, name := range inUseNames {
			fmt.Printf("  %s\n", ui.ColorInUse(name))
		}
		fmt.Println()
	}

	// Display available names
	if len(availableNames) > 0 {
		ui.PrintSection("Available (%d)", len(availableNames))
		// Show first 50 available names (pool size)
		displayCount := len(availableNames)
		if displayCount > 50 {
			displayCount = 50
		}
		for i := 0; i < displayCount; i++ {
			fmt.Printf("  %s\n", ui.ColorAvailable(availableNames[i]))
		}
		if len(availableNames) > displayCount {
			fmt.Printf("  ... and %d more\n", len(availableNames)-displayCount)
		}
	} else {
		ui.PrintSection("Available (0)")
		fmt.Println("  Pool exhausted - new polecats will use overflow naming")
	}

	return nil
}

func runNamesAdd(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name format
	if err := validateNameFormat(name); err != nil {
		return err
	}

	// Check for reserved names
	if polecat.ReservedInfraAgentNames[name] {
		return fmt.Errorf("cannot add reserved infrastructure name: %s", name)
	}

	rigPath, err := detectRigPath()
	if err != nil {
		return err
	}

	rigName := filepath.Base(rigPath)
	pool, err := loadNamePool(rigPath, rigName)
	if err != nil {
		return fmt.Errorf("loading name pool: %w", err)
	}

	// Check if name already exists
	if pool.HasName(name) {
		fmt.Printf("Name '%s' is already in the pool\n", name)
		return nil
	}

	// Add to pool
	pool.AddCustomName(name)
	if err := pool.Save(); err != nil {
		return fmt.Errorf("saving pool: %w", err)
	}

	// Also persist to settings for configuration permanence
	if err := persistCustomNameToSettings(rigPath, name); err != nil {
		return fmt.Errorf("persisting to settings: %w", err)
	}

	ui.PrintSuccess("Added '%s' to the name pool", name)
	return nil
}

func runNamesRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	rigPath, err := detectRigPath()
	if err != nil {
		return err
	}

	rigName := filepath.Base(rigPath)
	pool, err := loadNamePool(rigPath, rigName)
	if err != nil {
		return fmt.Errorf("loading name pool: %w", err)
	}

	// Check if name is in the pool
	if !pool.HasName(name) {
		return fmt.Errorf("name '%s' not found in pool", name)
	}

	// Check if name is a themed name (can't remove)
	if pool.IsThemedName(name) {
		return fmt.Errorf("cannot remove themed name '%s' - only custom names can be removed", name)
	}

	// Check if name is in use
	if pool.IsInUse(name) && !namesRemoveForce {
		return fmt.Errorf("name '%s' is currently in use - use --force to remove anyway", name)
	}

	// Remove from pool
	if err := pool.RemoveCustomName(name); err != nil {
		return fmt.Errorf("removing name: %w", err)
	}

	if err := pool.Save(); err != nil {
		return fmt.Errorf("saving pool: %w", err)
	}

	// Also remove from settings
	if err := removeCustomNameFromSettings(rigPath, name); err != nil {
		return fmt.Errorf("removing from settings: %w", err)
	}

	ui.PrintSuccess("Removed '%s' from the pool", name)
	return nil
}

func runNamesReserve(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Validate name format
	if err := validateNameFormat(name); err != nil {
		return err
	}

	rigPath, err := detectRigPath()
	if err != nil {
		return err
	}

	// Load settings
	settingsPath := filepath.Join(rigPath, "settings", "config.json")
	settings, err := loadRigSettings(settingsPath)
	if err != nil {
		if os.IsNotExist(err) || strings.Contains(err.Error(), "not found") {
			settings = newRigSettings()
		} else {
			return fmt.Errorf("loading settings: %w", err)
		}
	}

	// Initialize namepool config if needed
	if settings.Namepool == nil {
		settings.Namepool = defaultNamepoolConfig()
	}

	// Initialize reserved names if needed
	if settings.Namepool.ReservedNames == nil {
		settings.Namepool.ReservedNames = make([]string, 0)
	}

	// Check if already reserved
	for _, n := range settings.Namepool.ReservedNames {
		if n == name {
			fmt.Printf("Name '%s' is already reserved\n", name)
			return nil
		}
	}

	// Reserve the name
	settings.Namepool.ReservedNames = append(settings.Namepool.ReservedNames, name)

	// Save settings
	if err := saveRigSettings(settingsPath, settings); err != nil {
		return fmt.Errorf("saving settings: %w", err)
	}

	ui.PrintSuccess("Reserved name '%s'", name)
	return nil
}

func runNamesStats(cmd *cobra.Command, args []string) error {
	rigPath, err := detectRigPath()
	if err != nil {
		return err
	}

	rigName := filepath.Base(rigPath)
	pool, err := loadNamePool(rigPath, rigName)
	if err != nil {
		return fmt.Errorf("loading name pool: %w", err)
	}

	// Gather statistics
	totalNames := pool.TotalNames()
	availableNames := pool.AvailableNames()
	inUseNames := pool.ActiveNames()
	customCount := pool.CustomNameCount()
	theme := pool.GetTheme()
	maxSize := pool.MaxSize

	reservedCount := len(polecat.ReservedInfraAgentNames)

	// Display statistics
	ui.PrintHeading("Name Pool Statistics: %s", rigName)
	fmt.Println()

	ui.PrintSection("Configuration")
	fmt.Printf("  Theme:          %s\n", theme)
	fmt.Printf("  Max Pool Size:  %d\n", maxSize)
	fmt.Printf("  Custom Names:   %d\n", customCount)
	fmt.Println()

	ui.PrintSection("Capacity")
	fmt.Printf("  Total Names:    %d\n", totalNames)
	fmt.Printf("  Available:      %d (%d%%)\n", len(availableNames), percentage(len(availableNames), totalNames))
	fmt.Printf("  In Use:         %d (%d%%)\n", len(inUseNames), percentage(len(inUseNames), totalNames))
	fmt.Printf("  Reserved:       %d\n", reservedCount)
	fmt.Println()

	// Usage patterns
	if len(inUseNames) > 0 {
		ui.PrintSection("Usage Patterns")
		// Count overflow vs pooled names
		pooledInUse := 0
		overflowInUse := 0
		for _, name := range inUseNames {
			if pool.IsThemedName(name) || pool.IsCustomName(name) {
				pooledInUse++
			} else {
				overflowInUse++
			}
		}
		fmt.Printf("  Pooled Names:   %d\n", pooledInUse)
		fmt.Printf("  Overflow Names: %d\n", overflowInUse)

		if overflowInUse > 0 {
			fmt.Println()
			ui.PrintWarning("Pool exhausted - %d polecat(s) using overflow naming", overflowInUse)
		}
	}

	return nil
}

// Helper functions

func detectRigPath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	// Try to find rig from current directory
	rigPath, err := rig.FindRigFromPath(cwd)
	if err != nil {
		return "", fmt.Errorf("not in a rig directory")
	}

	return rigPath, nil
}

func loadNamePool(rigPath, rigName string) (*polecat.NamePool, error) {
	// Load settings for namepool config
	settingsPath := filepath.Join(rigPath, "settings", "config.json")

	// Try to load settings
	settings, err := loadRigSettings(settingsPath)
	if err == nil && settings.Namepool != nil {
		// Use configured namepool settings
		return polecat.NewNamePoolWithConfig(
			rigPath,
			rigName,
			settings.Namepool.Style,
			settings.Namepool.Names,
			settings.Namepool.MaxBeforeNumbering,
		), nil
	}

	// Use defaults
	pool := polecat.NewNamePool(rigPath, rigName)

	// Load state
	if err := pool.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading pool state: %w", err)
	}

	// Reconcile with existing polecats
	existingPolecats := listExistingPolecats(rigPath)
	pool.Reconcile(existingPolecats)

	return pool, nil
}

func listExistingPolecats(rigPath string) []string {
	polecatsDir := filepath.Join(rigPath, "polecats")
	entries, err := os.ReadDir(polecatsDir)
	if err != nil {
		return nil
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			names = append(names, entry.Name())
		}
	}
	return names
}

func validateNameFormat(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Check for valid characters (alphanumeric, hyphen, underscore)
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("invalid name format: '%s' (use only alphanumeric, hyphen, underscore)", name)
		}
	}

	// Check length
	if len(name) > 32 {
		return fmt.Errorf("name too long: %s (max 32 characters)", name)
	}

	return nil
}

func persistCustomNameToSettings(rigPath, name string) error {
	settingsPath := filepath.Join(rigPath, "settings", "config.json")
	settings, err := loadRigSettings(settingsPath)
	if err != nil {
		// Create new settings if not found
		if os.IsNotExist(err) || strings.Contains(err.Error(), "not found") {
			settings = newRigSettings()
		} else {
			return err
		}
	}

	// Initialize namepool config if needed
	if settings.Namepool == nil {
		settings.Namepool = defaultNamepoolConfig()
	}

	// Check if name already exists
	for _, n := range settings.Namepool.Names {
		if n == name {
			return nil // Already persisted
		}
	}

	// Append new name
	settings.Namepool.Names = append(settings.Namepool.Names, name)

	// Save settings
	return saveRigSettings(settingsPath, settings)
}

func removeCustomNameFromSettings(rigPath, name string) error {
	settingsPath := filepath.Join(rigPath, "settings", "config.json")
	settings, err := loadRigSettings(settingsPath)
	if err != nil {
		return err
	}

	if settings.Namepool == nil {
		return nil // No custom names to remove
	}

	// Filter out the name
	filtered := make([]string, 0, len(settings.Namepool.Names))
	for _, n := range settings.Namepool.Names {
		if n != name {
			filtered = append(filtered, n)
		}
	}
	settings.Namepool.Names = filtered

	// Save settings
	return saveRigSettings(settingsPath, settings)
}

func percentage(part, total int) int {
	if total == 0 {
		return 0
	}
	return (part * 100) / total
}

// Stub types for settings (will be properly imported from config package)
type rigSettings struct {
	Namepool *namepoolConfig `json:"namepool,omitempty"`
}

type namepoolConfig struct {
	Style              string   `json:"style,omitempty"`
	Names              []string `json:"names,omitempty"`
	ReservedNames      []string `json:"reserved_names,omitempty"`
	MaxBeforeNumbering int      `json:"max_before_numbering,omitempty"`
}

func loadRigSettings(path string) (*rigSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var settings rigSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

func saveRigSettings(path string, settings *rigSettings) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func newRigSettings() *rigSettings {
	return &rigSettings{}
}

func defaultNamepoolConfig() *namepoolConfig {
	return &namepoolConfig{
		Style:              polecat.DefaultTheme,
		MaxBeforeNumbering: polecat.DefaultPoolSize,
	}
}
