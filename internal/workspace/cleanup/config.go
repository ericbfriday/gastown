package cleanup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// DefaultConfigs returns default cleanup configurations for all workspace types.
func DefaultConfigs() map[WorkspaceType]*CleanupConfig {
	return map[WorkspaceType]*CleanupConfig{
		WorkspaceTypeCrew:     defaultCrewConfig(),
		WorkspaceTypePolecat:  defaultPolecatConfig(),
		WorkspaceTypeMayor:    defaultMayorConfig(),
		WorkspaceTypeRefinery: defaultRefineryConfig(),
		WorkspaceTypeTown:     defaultTownConfig(),
	}
}

// defaultCrewConfig returns cleanup config for persistent crew workspaces.
func defaultCrewConfig() *CleanupConfig {
	return &CleanupConfig{
		WorkspaceType: WorkspaceTypeCrew,
		Enabled:       true,
		Preflight: []CleanupRule{
			{
				Action:      ActionVerifyGitClean,
				Enabled:     true,
				SafetyCheck: true,
				Description: "Verify git working directory is clean before starting work",
			},
			{
				Action:      ActionRemoveDSStore,
				Enabled:     true,
				Patterns:    []string{".DS_Store"},
				Recursive:   true,
				SafetyCheck: false,
				Description: "Remove macOS metadata files",
			},
		},
		Postflight: []CleanupRule{
			{
				Action:      ActionBackupUncommitted,
				Enabled:     true,
				SafetyCheck: true,
				BackupFirst: true,
				Description: "Backup any uncommitted changes before cleanup",
			},
			{
				Action:      ActionRemoveTempFiles,
				Enabled:     true,
				Patterns:    []string{"*.tmp", "*.temp", "*.swp", "*.swo", "*~"},
				Recursive:   true,
				MaxDepth:    5,
				SafetyCheck: false,
				Description: "Remove temporary files",
			},
			{
				Action:      ActionRemoveLogs,
				Enabled:     true,
				Patterns:    []string{"*.log"},
				Recursive:   true,
				MaxDepth:    3,
				Exclude:     []string{"**/important.log", "**/keep/**"},
				SafetyCheck: false,
				Description: "Remove log files (except excluded paths)",
			},
			{
				Action:      ActionRemoveDSStore,
				Enabled:     true,
				Patterns:    []string{".DS_Store"},
				Recursive:   true,
				SafetyCheck: false,
				Description: "Remove macOS metadata files",
			},
		},
		OnIdle: []CleanupRule{
			{
				Action:      ActionClearBuildCache,
				Enabled:     true,
				Patterns:    []string{"node_modules/.cache/**", ".cache/**", "dist/**", "build/**"},
				Recursive:   false,
				SafetyCheck: true,
				Description: "Clear build caches when idle",
			},
		},
	}
}

// defaultPolecatConfig returns cleanup config for ephemeral polecat workspaces.
func defaultPolecatConfig() *CleanupConfig {
	return &CleanupConfig{
		WorkspaceType: WorkspaceTypePolecat,
		Enabled:       true,
		Preflight: []CleanupRule{
			{
				Action:      ActionCleanGitWorktree,
				Enabled:     true,
				SafetyCheck: false,
				Description: "Ensure clean git worktree state",
			},
		},
		Postflight: []CleanupRule{
			// Polecats are destroyed after use, so aggressive cleanup is safe
			{
				Action:      ActionRemoveTempFiles,
				Enabled:     true,
				Patterns:    []string{"*.tmp", "*.temp", "*.swp", "*.swo", "*~", "*.log"},
				Recursive:   true,
				SafetyCheck: false,
				Description: "Remove all temporary files (aggressive)",
			},
			{
				Action:      ActionClearBuildCache,
				Enabled:     true,
				Patterns:    []string{"node_modules/.cache/**", ".cache/**", "dist/**", "build/**"},
				Recursive:   false,
				SafetyCheck: false,
				Description: "Clear all build artifacts",
			},
			{
				Action:      ActionRemoveDSStore,
				Enabled:     true,
				Patterns:    []string{".DS_Store"},
				Recursive:   true,
				SafetyCheck: false,
				Description: "Remove macOS metadata files",
			},
		},
		OnIdle: []CleanupRule{},
	}
}

// defaultMayorConfig returns cleanup config for canonical mayor workspaces.
func defaultMayorConfig() *CleanupConfig {
	return &CleanupConfig{
		WorkspaceType: WorkspaceTypeMayor,
		Enabled:       true,
		Preflight: []CleanupRule{
			{
				Action:      ActionVerifyGitClean,
				Enabled:     true,
				SafetyCheck: true,
				Description: "Verify canonical clone is pristine",
			},
		},
		Postflight: []CleanupRule{
			{
				Action:      ActionRemoveDSStore,
				Enabled:     true,
				Patterns:    []string{".DS_Store"},
				Recursive:   true,
				SafetyCheck: false,
				Description: "Remove macOS metadata files",
			},
		},
		OnIdle: []CleanupRule{},
	}
}

// defaultRefineryConfig returns cleanup config for refinery merge queue workspaces.
func defaultRefineryConfig() *CleanupConfig {
	return &CleanupConfig{
		WorkspaceType: WorkspaceTypeRefinery,
		Enabled:       true,
		Preflight: []CleanupRule{
			{
				Action:      ActionVerifyGitClean,
				Enabled:     true,
				SafetyCheck: true,
				Description: "Verify merge queue is clean before processing",
			},
			{
				Action:      ActionRemoveTempFiles,
				Enabled:     true,
				Patterns:    []string{"*.tmp", "*.temp", "*.swp", "*.swo"},
				Recursive:   true,
				MaxDepth:    3,
				SafetyCheck: false,
				Description: "Remove temporary files before merge",
			},
		},
		Postflight: []CleanupRule{
			{
				Action:      ActionClearBuildCache,
				Enabled:     true,
				Patterns:    []string{"node_modules/.cache/**", ".cache/**", "dist/**", "build/**"},
				Recursive:   false,
				SafetyCheck: false,
				Description: "Clear build artifacts after merge processing",
			},
			{
				Action:      ActionRemoveDSStore,
				Enabled:     true,
				Patterns:    []string{".DS_Store"},
				Recursive:   true,
				SafetyCheck: false,
				Description: "Remove macOS metadata files",
			},
		},
		OnIdle: []CleanupRule{},
	}
}

// defaultTownConfig returns cleanup config for town-level workspace.
func defaultTownConfig() *CleanupConfig {
	return &CleanupConfig{
		WorkspaceType: WorkspaceTypeTown,
		Enabled:       true,
		Preflight: []CleanupRule{
			{
				Action:      ActionRemoveDSStore,
				Enabled:     true,
				Patterns:    []string{".DS_Store"},
				Recursive:   true,
				SafetyCheck: false,
				Description: "Remove macOS metadata files",
			},
		},
		Postflight: []CleanupRule{
			{
				Action:      ActionRemoveLogs,
				Enabled:     true,
				Patterns:    []string{"*.log"},
				Recursive:   false,
				Exclude:     []string{"harness/**", "daemon/**"},
				SafetyCheck: true,
				Description: "Remove top-level log files",
			},
			{
				Action:      ActionRemoveDSStore,
				Enabled:     true,
				Patterns:    []string{".DS_Store"},
				Recursive:   true,
				SafetyCheck: false,
				Description: "Remove macOS metadata files",
			},
		},
		OnIdle: []CleanupRule{},
	}
}

// LoadConfig loads cleanup configuration from file.
// Searches for:
// - .gastown/cleanup.json (preferred)
// - .claude/cleanup.json (fallback)
// - Returns default config if none found
func LoadConfig(workspaceRoot string, workspaceType WorkspaceType) (*CleanupConfig, error) {
	// Try loading from .gastown/cleanup.json
	configPath := filepath.Join(workspaceRoot, ".gastown", "cleanup.json")
	if config, err := loadConfigFile(configPath, workspaceType); err == nil {
		return config, nil
	}

	// Try loading from .claude/cleanup.json
	configPath = filepath.Join(workspaceRoot, ".claude", "cleanup.json")
	if config, err := loadConfigFile(configPath, workspaceType); err == nil {
		return config, nil
	}

	// Return default config
	defaults := DefaultConfigs()
	if config, ok := defaults[workspaceType]; ok {
		return config, nil
	}

	return nil, fmt.Errorf("no default config for workspace type: %s", workspaceType)
}

// loadConfigFile loads cleanup configuration from a specific file.
func loadConfigFile(path string, workspaceType WorkspaceType) (*CleanupConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Configuration file can contain multiple workspace type configs
	var configs map[WorkspaceType]*CleanupConfig
	if err := json.Unmarshal(data, &configs); err != nil {
		// Try loading single config
		var config CleanupConfig
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("invalid config format: %w", err)
		}
		return &config, nil
	}

	config, ok := configs[workspaceType]
	if !ok {
		return nil, fmt.Errorf("no config for workspace type: %s", workspaceType)
	}

	return config, nil
}

// SaveConfig saves cleanup configuration to file.
func SaveConfig(workspaceRoot string, config *CleanupConfig) error {
	// Save to .gastown/cleanup.json
	configDir := filepath.Join(workspaceRoot, ".gastown")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "cleanup.json")

	// Load existing configs if present
	configs := make(map[WorkspaceType]*CleanupConfig)
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &configs)
	}

	// Update with new config
	configs[config.WorkspaceType] = config

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ExportDefaultConfigs exports all default configurations to a file.
func ExportDefaultConfigs(outputPath string) error {
	configs := DefaultConfigs()

	data, err := json.MarshalIndent(configs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configs: %w", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write configs: %w", err)
	}

	return nil
}
