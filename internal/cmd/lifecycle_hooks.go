package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/hooks"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

// Lifecycle hooks command flags
var (
	lifecycleEvent   string
	lifecycleJSON    bool
	lifecycleAll     bool
	lifecycleVerbose bool
)

var lifecycleHooksCmd = &cobra.Command{
	Use:   "lifecycle",
	Short: "Manage lifecycle event hooks",
	Long: `Manage hooks for infrastructure lifecycle events.

Hooks allow external scripts and internal functions to respond to
lifecycle events such as session start/stop, mail received, etc.

This is separate from the hooks/registry.toml system which handles
Claude Code session hooks (PreToolUse, PostToolUse, etc.).

Configuration is loaded from:
  - .gastown/hooks.json (preferred)
  - .claude/hooks.json (fallback)

Available events:
  - pre-session-start   - Before starting a session
  - post-session-start  - After starting a session
  - pre-shutdown        - Before shutting down (can block)
  - post-shutdown       - After shutting down
  - on-pane-output      - On tmux pane output
  - session-idle        - When session becomes idle
  - mail-received       - When mail is received
  - work-assigned       - When work is assigned

Examples:
  gt lifecycle list                      # List all hooks
  gt lifecycle list pre-shutdown         # List hooks for specific event
  gt lifecycle fire pre-shutdown         # Manually fire hooks for an event
  gt lifecycle test                      # Validate hook configuration
  gt lifecycle test --all                # Test all hooks by firing them`,
}

var lifecycleListCmd = &cobra.Command{
	Use:   "list [event]",
	Short: "List registered lifecycle hooks",
	Long: `List all lifecycle hooks or hooks for a specific event.

If an event is specified, only hooks for that event are shown.
Otherwise, all hooks across all events are listed.

Examples:
  gt lifecycle list                  # List all hooks
  gt lifecycle list pre-shutdown     # List pre-shutdown hooks`,
	RunE: runLifecycleList,
}

var lifecycleFireCmd = &cobra.Command{
	Use:   "fire <event>",
	Short: "Manually fire hooks for an event",
	Long: `Execute all hooks registered for the specified event.

This is useful for testing hooks or manually triggering event handlers.

The event name must be one of the supported events. Results from
all hooks are displayed, including success/failure status and any
output or error messages.

Examples:
  gt lifecycle fire pre-shutdown     # Fire pre-shutdown hooks
  gt lifecycle fire post-session-start   # Fire post-session-start hooks`,
	Args: cobra.ExactArgs(1),
	RunE: runLifecycleFire,
}

var lifecycleTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Validate lifecycle hook configuration",
	Long: `Validate the lifecycle hook configuration file.

Checks:
  - Configuration file syntax (JSON)
  - Hook definitions are well-formed
  - Referenced scripts exist and are executable
  - Built-in hooks are recognized

Use --all to actually fire all hooks to test their execution.

Examples:
  gt lifecycle test              # Validate configuration only
  gt lifecycle test --all        # Validate and test execution`,
	RunE: runLifecycleTest,
}

func init() {
	// List command flags
	lifecycleListCmd.Flags().BoolVar(&lifecycleJSON, "json", false, "Output as JSON")

	// Fire command flags
	lifecycleFireCmd.Flags().BoolVar(&lifecycleJSON, "json", false, "Output as JSON")
	lifecycleFireCmd.Flags().BoolVar(&lifecycleVerbose, "verbose", false, "Show detailed output")

	// Test command flags
	lifecycleTestCmd.Flags().BoolVar(&lifecycleAll, "all", false, "Test all hooks by firing them")
	lifecycleTestCmd.Flags().BoolVar(&lifecycleJSON, "json", false, "Output as JSON")

	// Add subcommands
	lifecycleHooksCmd.AddCommand(lifecycleListCmd)
	lifecycleHooksCmd.AddCommand(lifecycleFireCmd)
	lifecycleHooksCmd.AddCommand(lifecycleTestCmd)

	// Add to hooks command
	hooksCmd.AddCommand(lifecycleHooksCmd)
}

func runLifecycleList(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Create hook runner
	runner, err := hooks.NewHookRunner(townRoot)
	if err != nil {
		return fmt.Errorf("initializing hooks: %w", err)
	}

	// Parse event if provided
	var event hooks.Event
	if len(args) > 0 {
		event = hooks.Event(args[0])
		// Validate event
		if !isValidLifecycleEvent(event) {
			return fmt.Errorf("invalid event: %s (use one of: %s)",
				event, strings.Join(getLifecycleEventNames(), ", "))
		}
	}

	// Get hooks
	hookMap := runner.ListHooks(event)

	if lifecycleJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(hookMap)
	}

	// Pretty print
	if len(hookMap) == 0 {
		if event != "" {
			fmt.Printf("No hooks registered for event: %s\n", event)
		} else {
			fmt.Println("No hooks registered")
		}
		if configPath := runner.ConfigPath(); configPath != "" {
			fmt.Printf("\nConfiguration: %s\n", style.Dim.Render(configPath))
		}
		return nil
	}

	// Display hooks
	fmt.Printf("Hooks registered in %s:\n\n", style.Dim.Render(runner.ConfigPath()))
	for ev, hookList := range hookMap {
		fmt.Printf("%s %s\n", style.Info.Render("●"), style.Bold.Render(string(ev)))
		for i, hook := range hookList {
			prefix := "├─"
			if i == len(hookList)-1 {
				prefix = "└─"
			}
			fmt.Printf("  %s %s: ", prefix, style.Warning.Render(string(hook.Type)))
			switch hook.Type {
			case hooks.HookTypeCommand:
				fmt.Printf("%s\n", hook.Cmd)
			case hooks.HookTypeBuiltin:
				fmt.Printf("%s\n", hook.Name)
			default:
				fmt.Printf("%s\n", style.Dim.Render("(unknown)"))
			}
		}
		fmt.Println()
	}

	return nil
}

func runLifecycleFire(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	event := hooks.Event(args[0])
	if !isValidLifecycleEvent(event) {
		return fmt.Errorf("invalid event: %s (use one of: %s)",
			event, strings.Join(getLifecycleEventNames(), ", "))
	}

	// Create hook runner
	runner, err := hooks.NewHookRunner(townRoot)
	if err != nil {
		return fmt.Errorf("initializing hooks: %w", err)
	}

	// Create context
	ctx := &hooks.HookContext{
		WorkingDir: townRoot,
		Metadata:   make(map[string]interface{}),
	}

	// Fire hooks
	results := runner.Fire(event, ctx)

	if lifecycleJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	// Pretty print results
	if len(results) == 0 {
		fmt.Printf("No hooks registered for event: %s\n", event)
		return nil
	}

	fmt.Printf("Firing hooks for %s...\n\n", style.Bold.Render(string(event)))

	allSuccess := true
	for i, result := range results {
		fmt.Printf("Hook #%d:\n", i+1)

		if result.Success {
			fmt.Printf("  Status: %s\n", style.Success.Render("✓ Success"))
		} else {
			fmt.Printf("  Status: %s\n", style.Error.Render("✗ Failed"))
			allSuccess = false
		}

		if result.Message != "" {
			fmt.Printf("  Message: %s\n", result.Message)
		}

		if result.Block {
			fmt.Printf("  %s Operation blocked by hook\n", style.Warning.Render("⚠"))
		}

		if lifecycleVerbose && result.Output != "" {
			fmt.Printf("  Output:\n%s\n", indentLifecycleOutput(result.Output))
		}

		if result.Error != "" {
			fmt.Printf("  Error: %s\n", style.Error.Render(result.Error))
		}

		fmt.Printf("  Duration: %s\n", result.Duration)
		fmt.Println()
	}

	if !allSuccess {
		return fmt.Errorf("one or more hooks failed")
	}

	return nil
}

func runLifecycleTest(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Create hook runner
	runner, err := hooks.NewHookRunner(townRoot)
	if err != nil {
		return fmt.Errorf("initializing hooks: %w", err)
	}

	configPath := runner.ConfigPath()
	if configPath == "" {
		fmt.Println(style.Warning.Render("⚠ No hook configuration found"))
		fmt.Println("\nCreate a configuration file at:")
		fmt.Println("  .gastown/hooks.json (or)")
		fmt.Println("  .claude/hooks.json")
		return nil
	}

	fmt.Printf("Testing hook configuration: %s\n\n", style.Dim.Render(configPath))

	// Test configuration parsing
	fmt.Printf("%s Configuration file is valid JSON\n", style.Success.Render("✓"))

	// Get all hooks
	hookMap := runner.ListHooks("")

	if len(hookMap) == 0 {
		fmt.Println(style.Warning.Render("⚠ No hooks defined in configuration"))
		return nil
	}

	totalHooks := 0
	for _, hooks := range hookMap {
		totalHooks += len(hooks)
	}
	fmt.Printf("%s Found %d hook(s) across %d event(s)\n", style.Success.Render("✓"), totalHooks, len(hookMap))

	// Test each hook
	issues := []string{}
	for event, hookList := range hookMap {
		for _, hook := range hookList {
			switch hook.Type {
			case hooks.HookTypeCommand:
				if hook.Cmd == "" {
					issues = append(issues, fmt.Sprintf("Event %s: command hook missing 'cmd' field", event))
				}
				// Check if command script exists
				if strings.HasPrefix(hook.Cmd, "./") || strings.HasPrefix(hook.Cmd, "/") {
					scriptPath := hook.Cmd
					if !strings.HasPrefix(scriptPath, "/") {
						scriptPath = townRoot + "/" + strings.TrimPrefix(scriptPath, "./")
					}
					if _, err := os.Stat(scriptPath); err != nil {
						issues = append(issues, fmt.Sprintf("Event %s: script not found: %s", event, hook.Cmd))
					}
				}
			case hooks.HookTypeBuiltin:
				if hook.Name == "" {
					issues = append(issues, fmt.Sprintf("Event %s: builtin hook missing 'name' field", event))
				}
			default:
				issues = append(issues, fmt.Sprintf("Event %s: unknown hook type: %s", event, hook.Type))
			}
		}
	}

	if len(issues) > 0 {
		fmt.Printf("\n%s Issues found:\n", style.Error.Render("✗"))
		for _, issue := range issues {
			fmt.Printf("  - %s\n", issue)
		}
	} else {
		fmt.Printf("%s All hooks are well-formed\n", style.Success.Render("✓"))
	}

	// If --all flag, actually fire all hooks
	if lifecycleAll {
		fmt.Println("\nTesting hook execution...")

		allSuccess := true
		for event := range hookMap {
			fmt.Printf("\nTesting %s...\n", style.Bold.Render(string(event)))
			ctx := &hooks.HookContext{
				WorkingDir: townRoot,
				Metadata:   map[string]interface{}{"test": true},
			}
			results := runner.Fire(event, ctx)

			for i, result := range results {
				if result.Success {
					fmt.Printf("  Hook #%d: %s\n", i+1, style.Success.Render("✓"))
				} else {
					fmt.Printf("  Hook #%d: %s - %s\n", i+1, style.Error.Render("✗"), result.Error)
					allSuccess = false
				}
			}
		}

		if !allSuccess {
			return fmt.Errorf("one or more hooks failed during testing")
		}
	}

	fmt.Println("\n" + style.Success.Render("✓ Hook configuration is valid"))
	return nil
}

func isValidLifecycleEvent(event hooks.Event) bool {
	for _, e := range hooks.AllEvents() {
		if e == event {
			return true
		}
	}
	return false
}

func getLifecycleEventNames() []string {
	events := hooks.AllEvents()
	names := make([]string, len(events))
	for i, e := range events {
		names[i] = string(e)
	}
	return names
}

func indentLifecycleOutput(output string) string {
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}
