package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// parseHooksFile reads and parses a Claude settings.json file,
// converting it to HookInfo structures for the specified agent.
func parseHooksFile(settingsPath, agent string) ([]HookInfo, error) {
	// Read the settings file
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil, err
	}

	// Parse as ClaudeSettings
	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	// Convert to HookInfo slice
	var hooks []HookInfo
	for hookType, matchers := range settings.Hooks {
		for _, matcher := range matchers {
			// Extract commands from the hooks
			var commands []string
			for _, hook := range matcher.Hooks {
				if hook.Command != "" {
					commands = append(commands, hook.Command)
				}
			}

			if len(commands) > 0 {
				hooks = append(hooks, HookInfo{
					Agent:    agent,
					Type:     hookType,
					Matcher:  matcher.Matcher,
					Commands: commands,
				})
			}
		}
	}

	return hooks, nil
}

// discoverHooks recursively finds all .claude/settings.json files under townRoot
// and parses them, returning all configured hooks with their agent paths.
func discoverHooks(townRoot string) ([]HookInfo, error) {
	var allHooks []HookInfo

	err := filepath.Walk(townRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip dotted directories, including those named .claude if they're in worker directories
		if info.IsDir() {
			name := filepath.Base(path)
			if strings.HasPrefix(name, ".") {
				parent := filepath.Base(filepath.Dir(path))
				// Only allow .claude if it's directly under a non-dotted directory
				// This allows rig/.claude, crew/.claude, etc. but skips polecats/.something/.claude
				if name != ".claude" || strings.HasPrefix(parent, ".") {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check if this is a settings.json file in a .claude directory
		if info.Name() == "settings.json" && filepath.Base(filepath.Dir(path)) == ".claude" {
			// Determine agent path (relative to townRoot)
			claudeDir := filepath.Dir(path)
			agentDir := filepath.Dir(claudeDir)
			relPath, err := filepath.Rel(townRoot, agentDir)
			if err != nil {
				return err
			}

			// Parse hooks from this settings file
			hooks, err := parseHooksFile(path, relPath)
			if err != nil {
				// Log error but continue discovery
				return nil
			}

			allHooks = append(allHooks, hooks...)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return allHooks, nil
}
