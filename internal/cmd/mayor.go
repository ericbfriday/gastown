package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/mayor"
	"github.com/steveyegge/gastown/internal/session"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/tmux"
	"github.com/steveyegge/gastown/internal/workspace"
)

// getMayorSessionName returns the Mayor session name.
func getMayorSessionName() string {
	return session.MayorSessionName()
}

// Mayor command flags
var (
	mayorContinue    bool
	mayorAgent       string
	mayorGracePeriod int
	mayorStatusJSON  bool
)

var mayorCmd = &cobra.Command{
	Use:     "mayor",
	GroupID: GroupAgents,
	Short:   "Manage the Mayor session",
	RunE:    requireSubcommand,
	Long: `Manage the Mayor global coordinator session.

The Mayor is the top-level coordinator agent that manages Gas Town infrastructure.
One Mayor per machine - multi-town requires containers/VMs for isolation.

Commands:
  start   Launch Mayor session
  attach  Attach to running Mayor session
  stop    Stop Mayor session
  status  Show Mayor session status`,
}

var mayorStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Mayor session",
	Long: `Start the Mayor global coordinator session.

Creates a tmux session for the Mayor agent. The session is created in the
town root directory with appropriate environment and theming.

Examples:
  gt mayor start                  # Start with default agent
  gt mayor start --agent claude   # Start with specific agent type
  gt mayor start --continue       # Resume from handoff mail`,
	RunE: runMayorStart,
}

var mayorAttachCmd = &cobra.Command{
	Use:     "attach",
	Aliases: []string{"at"},
	Short:   "Attach to the Mayor session",
	Long: `Attach to the running Mayor tmux session.

Attaches the current terminal to the Mayor session. Detach with Ctrl-B D.

Examples:
  gt mayor attach
  gt mayor at`,
	RunE: runMayorAttach,
}

var mayorStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the Mayor session",
	Long: `Stop the Mayor session.

Attempts graceful shutdown first (Ctrl-C), then kills the tmux session.

Examples:
  gt mayor stop
  gt mayor stop --grace-period 5`,
	RunE: runMayorStop,
}

var mayorStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Mayor session status",
	Long: `Show detailed status for the Mayor session.

Displays running state, session info, uptime, and activity.

Examples:
  gt mayor status
  gt mayor status --json`,
	RunE: runMayorStatus,
}

func init() {
	// Start flags
	mayorStartCmd.Flags().BoolVar(&mayorContinue, "continue", false, "Resume from handoff mail")
	mayorStartCmd.Flags().StringVar(&mayorAgent, "agent", "claude", "Agent type to use")

	// Stop flags
	mayorStopCmd.Flags().IntVar(&mayorGracePeriod, "grace-period", 0, "Grace period in seconds before force shutdown")

	// Status flags
	mayorStatusCmd.Flags().BoolVar(&mayorStatusJSON, "json", false, "Output as JSON")

	// Add subcommands
	mayorCmd.AddCommand(mayorStartCmd)
	mayorCmd.AddCommand(mayorAttachCmd)
	mayorCmd.AddCommand(mayorStopCmd)
	mayorCmd.AddCommand(mayorStatusCmd)

	rootCmd.AddCommand(mayorCmd)
}

func runMayorStart(cmd *cobra.Command, args []string) error {
	// Find town root
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Create mayor manager
	mgr := mayor.NewManager(townRoot)

	// Check if already running
	running, err := mgr.IsRunning()
	if err != nil {
		return fmt.Errorf("checking mayor status: %w", err)
	}
	if running {
		return fmt.Errorf("mayor already running. Use 'gt mayor attach' to connect.")
	}

	fmt.Println("Starting Mayor session...")

	// Start the mayor
	agentOverride := ""
	if mayorAgent != "" && mayorAgent != "claude" {
		agentOverride = mayorAgent
	}

	if err := mgr.Start(agentOverride); err != nil {
		return fmt.Errorf("starting mayor: %w", err)
	}

	fmt.Printf("%s Mayor started. Attach with: %s\n",
		style.Bold.Render("✓"),
		style.Dim.Render("gt mayor attach"))

	if mayorContinue {
		// TODO: Implement handoff mail continuation
		fmt.Println(style.Dim.Render("Note: --continue flag not yet implemented"))
	}

	return nil
}

func runMayorAttach(cmd *cobra.Command, args []string) error {
	// Find town root
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Create mayor manager
	mgr := mayor.NewManager(townRoot)

	// Check if running
	running, err := mgr.IsRunning()
	if err != nil {
		return fmt.Errorf("checking mayor status: %w", err)
	}
	if !running {
		return fmt.Errorf("mayor not running. Start with: gt mayor start")
	}

	// Attach to session
	t := tmux.NewTmux()
	sessionID := mgr.SessionName()
	return t.AttachSession(sessionID)
}

func runMayorStop(cmd *cobra.Command, args []string) error {
	// Find town root
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Create mayor manager
	mgr := mayor.NewManager(townRoot)

	// Check if running
	running, err := mgr.IsRunning()
	if err != nil {
		return fmt.Errorf("checking mayor status: %w", err)
	}
	if !running {
		return fmt.Errorf("mayor not running")
	}

	// Apply grace period if specified
	if mayorGracePeriod > 0 {
		fmt.Printf("Waiting %d seconds before shutdown...\n", mayorGracePeriod)
		time.Sleep(time.Duration(mayorGracePeriod) * time.Second)
	}

	fmt.Println("Stopping Mayor session...")

	// Stop the mayor
	if err := mgr.Stop(); err != nil {
		return fmt.Errorf("stopping mayor: %w", err)
	}

	fmt.Printf("%s Mayor stopped.\n", style.Bold.Render("✓"))
	return nil
}

func runMayorStatus(cmd *cobra.Command, args []string) error {
	// Find town root
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Create mayor manager
	mgr := mayor.NewManager(townRoot)

	// Check if running
	running, err := mgr.IsRunning()
	if err != nil {
		return fmt.Errorf("checking mayor status: %w", err)
	}

	if !running {
		if mayorStatusJSON {
			status := map[string]interface{}{
				"running":    false,
				"session_id": mgr.SessionName(),
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(status)
		}
		fmt.Printf("%s Mayor: %s\n", style.Bold.Render("🎩"), style.Dim.Render("stopped"))
		return nil
	}

	// Get detailed status
	info, err := mgr.Status()
	if err != nil {
		return fmt.Errorf("getting status: %w", err)
	}

	// Output format
	if mayorStatusJSON {
		status := map[string]interface{}{
			"running":    true,
			"session_id": info.Name,
			"windows":    info.Windows,
			"created":    info.Created,
			"attached":   info.Attached,
			"activity":   info.Activity,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	// Human-readable output
	fmt.Printf("%s Mayor Session\n\n", style.Bold.Render("🎩"))
	fmt.Printf("  State: %s\n", style.Bold.Render("● running"))
	fmt.Printf("  Session ID: %s\n", info.Name)

	if info.Attached {
		fmt.Printf("  Attached: yes\n")
	} else {
		fmt.Printf("  Attached: no\n")
	}

	if info.Created != "" {
		fmt.Printf("  Created: %s\n", info.Created)
	}

	if info.Windows > 0 {
		fmt.Printf("  Windows: %d\n", info.Windows)
	}

	fmt.Printf("\nAttach with: %s\n", style.Dim.Render("gt mayor attach"))
	return nil
}
