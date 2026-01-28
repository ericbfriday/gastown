package cmd

import (
	"github.com/spf13/cobra"
)

// Hooks command shared flags
var (
	hooksVerbose bool
)

var hooksCmd = &cobra.Command{
	Use:     "hooks",
	GroupID: GroupConfig,
	Short:   "Manage hooks and event handlers",
	Long: `Manage hooks for both Claude Code sessions and infrastructure lifecycle events.

Gas Town supports two hook systems:

1. **Registry Hooks** (hooks/registry.toml)
   Claude Code session hooks (PreToolUse, PostToolUse, etc.)
   Use 'gt hooks list' and 'gt hooks install'

2. **Lifecycle Hooks** (.gastown/hooks.json or .claude/hooks.json)
   Infrastructure event hooks (pre-shutdown, mail-received, etc.)
   Use 'gt hooks lifecycle list|fire|test'

Examples:
  gt hooks list                   # List registry hooks
  gt hooks install pre-shutdown   # Install a registry hook
  gt hooks lifecycle list         # List lifecycle hooks
  gt hooks lifecycle fire pre-shutdown   # Fire lifecycle hooks`,
}

func init() {
	rootCmd.AddCommand(hooksCmd)
}
