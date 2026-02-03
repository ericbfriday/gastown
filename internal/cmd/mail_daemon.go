package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/daemon"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/workspace"
)

var mailDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the mail orchestrator daemon",
	Long: `Manage the mail orchestrator daemon for async mail delivery.

The mail orchestrator monitors mail queues and delivers messages
automatically based on priority and routing rules.`,
	RunE: requireSubcommand,
}

var mailDaemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the mail orchestrator",
	Long: `Start the mail orchestrator daemon.

The orchestrator will run as part of the main daemon and monitor
mail queues for delivery.`,
	RunE: runMailDaemonStart,
}

var mailDaemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the mail orchestrator",
	Long:  `Stop the mail orchestrator daemon.`,
	RunE:  runMailDaemonStop,
}

var mailDaemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mail orchestrator status",
	Long:  `Show the current status of the mail orchestrator daemon.`,
	RunE:  runMailDaemonStatus,
}

var mailDaemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View mail orchestrator logs",
	Long:  `View the mail orchestrator logs from the daemon log file.`,
	RunE:  runMailDaemonLogs,
}

var mailDaemonQueueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Show mail queue status",
	Long:  `Show the current status of mail queues (inbound, outbound, dead letter).`,
	RunE:  runMailDaemonQueue,
}

func init() {
	mailDaemonCmd.AddCommand(mailDaemonStartCmd)
	mailDaemonCmd.AddCommand(mailDaemonStopCmd)
	mailDaemonCmd.AddCommand(mailDaemonStatusCmd)
	mailDaemonCmd.AddCommand(mailDaemonLogsCmd)
	mailDaemonCmd.AddCommand(mailDaemonQueueCmd)

	mailCmd.AddCommand(mailDaemonCmd)
}

func runMailDaemonStart(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Check if main daemon is running
	running, _, err := daemon.IsRunning(townRoot)
	if err != nil {
		return fmt.Errorf("checking daemon status: %w", err)
	}
	if !running {
		fmt.Printf("%s Main daemon is not running. Start with 'gt daemon start'\n",
			style.Bold.Render("!"))
		return nil
	}

	// Enable mail orchestrator in config
	configPath := daemon.PatrolConfigFile(townRoot)
	config := daemon.LoadPatrolConfig(townRoot)
	if config == nil {
		config = &daemon.DaemonPatrolConfig{
			Type:    "daemon_config",
			Version: 1,
			Patrols: &daemon.PatrolsConfig{},
		}
	}
	if config.Patrols == nil {
		config.Patrols = &daemon.PatrolsConfig{}
	}

	config.Patrols.MailOrchestrator = &daemon.PatrolConfig{
		Enabled: true,
	}

	// Save config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("%s Mail orchestrator enabled in daemon config\n", style.Bold.Render("✓"))
	fmt.Printf("Restart daemon to apply: %s\n",
		style.Dim.Render("gt daemon stop && gt daemon start"))

	return nil
}

func runMailDaemonStop(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Disable mail orchestrator in config
	configPath := daemon.PatrolConfigFile(townRoot)
	config := daemon.LoadPatrolConfig(townRoot)
	if config == nil || config.Patrols == nil || config.Patrols.MailOrchestrator == nil {
		fmt.Printf("%s Mail orchestrator is not configured\n", style.Dim.Render("○"))
		return nil
	}

	config.Patrols.MailOrchestrator.Enabled = false

	// Save config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Printf("%s Mail orchestrator disabled in daemon config\n", style.Bold.Render("✓"))
	fmt.Printf("Restart daemon to apply: %s\n",
		style.Dim.Render("gt daemon stop && gt daemon start"))

	return nil
}

func runMailDaemonStatus(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Check if daemon is running
	running, pid, err := daemon.IsRunning(townRoot)
	if err != nil {
		return fmt.Errorf("checking daemon status: %w", err)
	}

	if !running {
		fmt.Printf("%s Main daemon is not running\n", style.Dim.Render("○"))
		fmt.Printf("Start with: %s\n", style.Dim.Render("gt daemon start"))
		return nil
	}

	// Check if mail orchestrator is enabled
	config := daemon.LoadPatrolConfig(townRoot)
	enabled := daemon.IsPatrolEnabled(config, "mail-orchestrator")

	if enabled {
		fmt.Printf("%s Mail orchestrator is %s (daemon PID %d)\n",
			style.Bold.Render("●"),
			style.Bold.Render("enabled"),
			pid)
	} else {
		fmt.Printf("%s Mail orchestrator is %s (daemon PID %d)\n",
			style.Dim.Render("○"),
			style.Dim.Render("disabled"),
			pid)
		fmt.Printf("Enable with: %s\n", style.Dim.Render("gt mail daemon start"))
	}

	return nil
}

func runMailDaemonLogs(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	logFile := filepath.Join(townRoot, "daemon", "daemon.log")
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		return fmt.Errorf("no daemon log file found")
	}

	// Use grep to filter for mail orchestrator logs
	// Grep for "mail" (case insensitive) and pipe to tail
	fmt.Printf("Showing mail orchestrator logs from: %s\n\n", logFile)

	// Just tail the whole daemon log for now - user can grep themselves
	return runDaemonLogs(cmd, args)
}

func runMailDaemonQueue(cmd *cobra.Command, args []string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Load queue files
	queueDir := filepath.Join(townRoot, "daemon", "mail-queues")

	// Load and display each queue
	inbound := loadQueueSize(filepath.Join(queueDir, "inbound.json"))
	outbound := loadQueueSize(filepath.Join(queueDir, "outbound.json"))
	deadLetter := loadQueueSize(filepath.Join(queueDir, "dead-letter.json"))

	fmt.Println(style.Bold.Render("Mail Queue Status:"))
	fmt.Printf("  Inbound:     %s\n", formatQueueSize(inbound))
	fmt.Printf("  Outbound:    %s\n", formatQueueSize(outbound))
	fmt.Printf("  Dead Letter: %s\n", formatQueueSize(deadLetter))

	if deadLetter > 0 {
		fmt.Printf("\n%s Dead letter queue has %d messages requiring attention\n",
			style.Bold.Render("⚠"),
			deadLetter)
	}

	return nil
}

func loadQueueSize(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}

	var queue []interface{}
	if err := json.Unmarshal(data, &queue); err != nil {
		return 0
	}

	return len(queue)
}

func formatQueueSize(size int) string {
	if size == 0 {
		return style.Dim.Render("0 messages")
	}
	return fmt.Sprintf("%d messages", size)
}
