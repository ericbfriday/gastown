package cmd

// ClaudeSettings represents the structure of .claude/settings.json files.
// This is the configuration file that Claude Code reads to determine hook behavior.
type ClaudeSettings struct {
	Hooks          map[string][]ClaudeHookMatcher `json:"hooks,omitempty"`
	EnabledPlugins map[string]bool                `json:"enabledPlugins,omitempty"`
}

// ClaudeHookMatcher represents a hook entry in the settings file.
// Each matcher can trigger multiple hooks based on the matcher pattern.
type ClaudeHookMatcher struct {
	Matcher string       `json:"matcher"`
	Hooks   []ClaudeHook `json:"hooks"`
}

// ClaudeHook represents a single hook action to execute.
type ClaudeHook struct {
	Type    string `json:"type"`    // "command" or other hook types
	Command string `json:"command"` // Shell command to execute
}

// HookInfo represents discovered hook information for display.
// Used by the hooks list command to show what hooks are configured.
type HookInfo struct {
	Agent    string
	Type     string   // Hook event type (SessionStart, PreToolUse, etc.)
	Matcher  string   // File matcher pattern
	Commands []string // Commands to execute
}
