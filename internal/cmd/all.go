package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/git"
	"github.com/steveyegge/gastown/internal/polecat"
	"github.com/steveyegge/gastown/internal/rig"
	"github.com/steveyegge/gastown/internal/style"
	"github.com/steveyegge/gastown/internal/tmux"
)

// All command flags
var (
	allRig      string
	allStatus   string
	allPattern  string
	allDryRun   bool
	allForce    bool
	allJSON     bool
	allParallel int
)

var allCmd = &cobra.Command{
	Use:     "all <command> [specs...]",
	GroupID: GroupAgents,
	Short:   "Run commands on multiple polecats simultaneously",
	RunE:    requireSubcommand,
	Long: `Run commands on multiple polecats simultaneously.

Spec patterns:
  Toast              - Specific polecat (in default/current rig)
  gastown/Toast      - Specific rig/polecat
  gastown/*          - All polecats in rig
  *                  - All polecats everywhere

Filtering options:
  --rig <name>       - Filter by specific rig
  --status <state>   - Filter by status (working/idle/stuck/done)
  --pattern <regex>  - Filter by name pattern

Examples:
  gt all status                          # Status of all polecats
  gt all status gastown/*                # Status of gastown polecats
  gt all stop gastown/* --force          # Stop all gastown sessions
  gt all status --rig gastown            # Status filtered by rig
  gt all status --status working         # Only working polecats
  gt all status --pattern "Toast.*"      # Pattern match names`,
}

var allStatusCmd = &cobra.Command{
	Use:   "status [specs...]",
	Short: "Show status of multiple polecats",
	Long: `Show status of multiple polecats in table format.

If no specs provided, shows all polecats.
Use filters to narrow results.

Examples:
  gt all status
  gt all status gastown/*
  gt all status --rig gastown --status working
  gt all status --json`,
	RunE: runAllStatus,
}

var allStopCmd = &cobra.Command{
	Use:   "stop [specs...]",
	Short: "Stop sessions for multiple polecats",
	Long: `Stop tmux sessions for multiple polecats.

Use --force to kill sessions immediately without graceful shutdown.
Dry-run mode shows what would be stopped without actually stopping.

Examples:
  gt all stop gastown/*
  gt all stop --rig gastown --status working
  gt all stop gastown/* --force
  gt all stop --dry-run`,
	RunE: runAllStop,
}

var allStartCmd = &cobra.Command{
	Use:   "start [specs...]",
	Short: "Start sessions for multiple polecats",
	Long: `Start tmux sessions for multiple polecats.

Sessions are started in parallel using a worker pool.
Failed starts are reported but don't stop other operations.

Examples:
  gt all start gastown/*
  gt all start --rig gastown --status idle`,
	RunE: runAllStart,
}

var allAttachCmd = &cobra.Command{
	Use:   "attach [specs...]",
	Short: "Attach to multiple sessions in tmux panes/windows",
	Long: `Attach to multiple polecat sessions in tmux.

Creates a new tmux window with panes for each session.
If only one polecat matches, attaches directly.

Examples:
  gt all attach gastown/*
  gt all attach --rig gastown --status working`,
	RunE: runAllAttach,
}

var allRunCmd = &cobra.Command{
	Use:   "run <command> [specs...]",
	Short: "Run command in multiple polecat sessions",
	Long: `Run a command in multiple polecat sessions.

The command is sent to each session's tmux session.
Results are collected and reported.

Examples:
  gt all run "git status" gastown/*
  gt all run "gt status" --rig gastown`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAllRun,
}

func init() {
	// All command flags
	allCmd.PersistentFlags().StringVar(&allRig, "rig", "", "Filter by specific rig")
	allCmd.PersistentFlags().StringVar(&allStatus, "status", "", "Filter by status (working/idle/stuck/done)")
	allCmd.PersistentFlags().StringVar(&allPattern, "pattern", "", "Filter by name pattern (regex)")
	allCmd.PersistentFlags().BoolVar(&allDryRun, "dry-run", false, "Show what would happen without doing it")
	allCmd.PersistentFlags().IntVar(&allParallel, "parallel", 5, "Number of parallel workers")

	// Status flags
	allStatusCmd.Flags().BoolVar(&allJSON, "json", false, "Output as JSON")

	// Stop flags
	allStopCmd.Flags().BoolVarP(&allForce, "force", "f", false, "Force kill sessions")

	// Start flags - none specific

	// Attach flags - none specific

	// Run flags - none specific

	// Add subcommands
	allCmd.AddCommand(allStatusCmd)
	allCmd.AddCommand(allStopCmd)
	allCmd.AddCommand(allStartCmd)
	allCmd.AddCommand(allAttachCmd)
	allCmd.AddCommand(allRunCmd)

	rootCmd.AddCommand(allCmd)
}

// polecatSpec represents a polecat specification with its manager context.
type polecatSpec struct {
	rigName     string
	polecatName string
	mgr         *polecat.Manager
	r           *rig.Rig
	p           *polecat.Polecat
}

// expandSpecs converts spec patterns to actual polecat list.
func expandSpecs(specs []string) ([]polecatSpec, error) {
	var results []polecatSpec

	// If no specs, default to "*" (all polecats everywhere)
	if len(specs) == 0 {
		specs = []string{"*"}
	}

	// Process each spec
	for _, spec := range specs {
		expanded, err := expandSingleSpec(spec)
		if err != nil {
			return nil, fmt.Errorf("expanding spec '%s': %w", spec, err)
		}
		results = append(results, expanded...)
	}

	// Apply filters
	filtered := applyFilters(results)

	return filtered, nil
}

// expandSingleSpec expands a single spec pattern.
func expandSingleSpec(spec string) ([]polecatSpec, error) {
	var results []polecatSpec

	// Pattern: * (all polecats everywhere)
	if spec == "*" {
		allRigs, _, err := getAllRigs()
		if err != nil {
			return nil, fmt.Errorf("listing rigs: %w", err)
		}

		t := tmux.NewTmux()
		for _, r := range allRigs {
			polecatGit := git.NewGit(r.Path)
			mgr := polecat.NewManager(r, polecatGit, t)

			polecats, err := mgr.List()
			if err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to list polecats in %s: %v\n", r.Name, err)
				continue
			}

			for _, p := range polecats {
				results = append(results, polecatSpec{
					rigName:     r.Name,
					polecatName: p.Name,
					mgr:         mgr,
					r:           r,
					p:           p,
				})
			}
		}
		return results, nil
	}

	// Pattern: rig/* (all polecats in rig)
	if strings.HasSuffix(spec, "/*") {
		rigName := strings.TrimSuffix(spec, "/*")
		mgr, r, err := getPolecatManager(rigName)
		if err != nil {
			return nil, err
		}

		polecats, err := mgr.List()
		if err != nil {
			return nil, fmt.Errorf("listing polecats in rig '%s': %w", rigName, err)
		}

		for _, p := range polecats {
			results = append(results, polecatSpec{
				rigName:     rigName,
				polecatName: p.Name,
				mgr:         mgr,
				r:           r,
				p:           p,
			})
		}
		return results, nil
	}

	// Pattern: rig/polecat (specific polecat)
	if strings.Contains(spec, "/") {
		rigName, polecatName, err := parseAddress(spec)
		if err != nil {
			return nil, err
		}

		mgr, r, err := getPolecatManager(rigName)
		if err != nil {
			return nil, err
		}

		p, err := mgr.Get(polecatName)
		if err != nil {
			return nil, fmt.Errorf("polecat '%s' not found in rig '%s'", polecatName, rigName)
		}

		results = append(results, polecatSpec{
			rigName:     rigName,
			polecatName: polecatName,
			mgr:         mgr,
			r:           r,
			p:           p,
		})
		return results, nil
	}

	// Pattern: polecat (requires rig context via --rig flag)
	// For bare polecat names, require the --rig flag to avoid ambiguity
	return nil, fmt.Errorf("ambiguous spec '%s': use rig/polecat format (e.g., gastown/Toast) or provide --rig flag", spec)
}

// applyFilters applies command-line filters to polecat specs.
func applyFilters(specs []polecatSpec) []polecatSpec {
	var filtered []polecatSpec

	for _, spec := range specs {
		// Filter by rig
		if allRig != "" && spec.rigName != allRig {
			continue
		}

		// Filter by status
		if allStatus != "" {
			status := polecat.State(allStatus)
			if spec.p.State != status {
				continue
			}
		}

		// Filter by pattern (name regex)
		if allPattern != "" {
			// Simple pattern matching (could use regexp for more sophisticated matching)
			if !strings.Contains(spec.polecatName, allPattern) {
				continue
			}
		}

		filtered = append(filtered, spec)
	}

	return filtered
}

// runAllStatus shows status of multiple polecats.
func runAllStatus(cmd *cobra.Command, args []string) error {
	specs, err := expandSpecs(args)
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No polecats match the criteria.")
		return nil
	}

	// Get session status for each polecat
	t := tmux.NewTmux()
	type statusResult struct {
		Rig            string        `json:"rig"`
		Name           string        `json:"name"`
		State          polecat.State `json:"state"`
		Issue          string        `json:"issue,omitempty"`
		SessionRunning bool          `json:"session_running"`
		ClonePath      string        `json:"clone_path,omitempty"`
	}

	var results []statusResult
	for _, spec := range specs {
		polecatMgr := polecat.NewSessionManager(t, spec.r)
		running, _ := polecatMgr.IsRunning(spec.polecatName)

		results = append(results, statusResult{
			Rig:            spec.rigName,
			Name:           spec.polecatName,
			State:          spec.p.State,
			Issue:          spec.p.Issue,
			SessionRunning: running,
			ClonePath:      spec.p.ClonePath,
		})
	}

	// JSON output
	if allJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	// Table output
	fmt.Printf("%s\n\n", style.Bold.Render(fmt.Sprintf("Polecat Status (%d)", len(results))))
	fmt.Printf("%-20s %-15s %-12s %-8s %s\n", "RIG/POLECAT", "STATE", "SESSION", "ISSUE", "")
	fmt.Printf("%s\n", strings.Repeat("-", 80))

	for _, r := range results {
		// Session indicator
		sessionStatus := style.Dim.Render("○")
		if r.SessionRunning {
			sessionStatus = style.Success.Render("●")
		}

		// State color
		stateStr := string(r.State)
		switch r.State {
		case polecat.StateWorking:
			stateStr = style.Info.Render(stateStr)
		case polecat.StateStuck:
			stateStr = style.Warning.Render(stateStr)
		case polecat.StateDone:
			stateStr = style.Success.Render(stateStr)
		default:
			stateStr = style.Dim.Render(stateStr)
		}

		// Issue
		issueStr := "-"
		if r.Issue != "" {
			issueStr = r.Issue
		}

		fmt.Printf("%-20s %-15s %-12s %-8s\n",
			fmt.Sprintf("%s/%s", r.Rig, r.Name),
			stateStr,
			sessionStatus,
			issueStr)
	}

	return nil
}

// runAllStop stops sessions for multiple polecats.
func runAllStop(cmd *cobra.Command, args []string) error {
	specs, err := expandSpecs(args)
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No polecats match the criteria.")
		return nil
	}

	// Dry-run mode
	if allDryRun {
		fmt.Printf("Would stop %d polecat session(s):\n\n", len(specs))
		for _, spec := range specs {
			forceStr := ""
			if allForce {
				forceStr = " (--force)"
			}
			fmt.Printf("  - %s/%s%s\n", spec.rigName, spec.polecatName, forceStr)
		}
		return nil
	}

	// Confirm destructive operation
	if !allForce && len(specs) > 1 {
		fmt.Printf("About to stop %d polecat session(s). Continue? (y/N): ", len(specs))
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Stop sessions in parallel
	type stopResult struct {
		spec polecatSpec
		err  error
	}

	results := make(chan stopResult, len(specs))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, allParallel)

	for _, spec := range specs {
		wg.Add(1)
		go func(s polecatSpec) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			t := tmux.NewTmux()
			polecatMgr := polecat.NewSessionManager(t, s.r)

			// Check if running first
			running, _ := polecatMgr.IsRunning(s.polecatName)
			if !running {
				results <- stopResult{spec: s, err: nil}
				return
			}

			err := polecatMgr.Stop(s.polecatName, allForce)
			results <- stopResult{spec: s, err: err}
		}(spec)
	}

	wg.Wait()
	close(results)

	// Collect and report results
	var stopped, skipped int
	var errors []string

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s/%s: %v", result.spec.rigName, result.spec.polecatName, result.err))
		} else {
			stopped++
		}
	}

	// Summary
	fmt.Printf("\n%s Stopped %d session(s)", style.SuccessPrefix, stopped)
	if skipped > 0 {
		fmt.Printf(" (%d already stopped)", skipped)
	}
	fmt.Println()

	if len(errors) > 0 {
		fmt.Printf("\n%s Some operations failed:\n", style.Warning.Render("Warning:"))
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("%d operation(s) failed", len(errors))
	}

	return nil
}

// runAllStart starts sessions for multiple polecats.
func runAllStart(cmd *cobra.Command, args []string) error {
	specs, err := expandSpecs(args)
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No polecats match the criteria.")
		return nil
	}

	// Dry-run mode
	if allDryRun {
		fmt.Printf("Would start %d polecat session(s):\n\n", len(specs))
		for _, spec := range specs {
			fmt.Printf("  - %s/%s\n", spec.rigName, spec.polecatName)
		}
		return nil
	}

	// Start sessions in parallel
	type startResult struct {
		spec polecatSpec
		err  error
	}

	results := make(chan startResult, len(specs))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, allParallel)

	for _, spec := range specs {
		wg.Add(1)
		go func(s polecatSpec) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			t := tmux.NewTmux()
			polecatMgr := polecat.NewSessionManager(t, s.r)

			// Check if already running
			running, _ := polecatMgr.IsRunning(s.polecatName)
			if running {
				results <- startResult{spec: s, err: nil}
				return
			}

			// Start session
			opts := polecat.SessionStartOptions{
				WorkDir: s.p.ClonePath,
				Issue:   s.p.Issue,
			}
			err := polecatMgr.Start(s.polecatName, opts)
			results <- startResult{spec: s, err: err}
		}(spec)
	}

	wg.Wait()
	close(results)

	// Collect and report results
	var started int
	var errors []string

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s/%s: %v", result.spec.rigName, result.spec.polecatName, result.err))
		} else {
			started++
		}
	}

	// Summary
	fmt.Printf("\n%s Started %d session(s)\n", style.SuccessPrefix, started)

	if len(errors) > 0 {
		fmt.Printf("\n%s Some operations failed:\n", style.Warning.Render("Warning:"))
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("%d operation(s) failed", len(errors))
	}

	return nil
}

// runAllAttach attaches to multiple sessions in tmux.
func runAllAttach(cmd *cobra.Command, args []string) error {
	specs, err := expandSpecs(args)
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No polecats match the criteria.")
		return nil
	}

	// If only one polecat, attach directly
	if len(specs) == 1 {
		t := tmux.NewTmux()
		polecatMgr := polecat.NewSessionManager(t, specs[0].r)
		sessionName := polecatMgr.SessionName(specs[0].polecatName)

		// Check if running
		running, _ := polecatMgr.IsRunning(specs[0].polecatName)
		if !running {
			return fmt.Errorf("session not running: %s/%s", specs[0].rigName, specs[0].polecatName)
		}

		return t.AttachSession(sessionName)
	}

	// Multiple polecats - create list of sessions to attach to
	fmt.Printf("Attaching to %d polecat sessions.\n", len(specs))
	fmt.Println("\nNote: To attach to multiple sessions, use tmux directly:")
	fmt.Println("  tmux new-window -n 'polecats'")

	t := tmux.NewTmux()
	for _, spec := range specs {
		polecatMgr := polecat.NewSessionManager(t, spec.r)
		sessionName := polecatMgr.SessionName(spec.polecatName)

		// Check if running
		running, _ := polecatMgr.IsRunning(spec.polecatName)
		status := style.Dim.Render("not running")
		if running {
			status = style.Success.Render("running")
		}

		fmt.Printf("  %s/%s: %s (tmux attach -t %s)\n", spec.rigName, spec.polecatName, status, sessionName)
	}

	// Attach to first running session
	for _, spec := range specs {
		polecatMgr := polecat.NewSessionManager(t, spec.r)
		running, _ := polecatMgr.IsRunning(spec.polecatName)
		if running {
			sessionName := polecatMgr.SessionName(spec.polecatName)
			fmt.Printf("\nAttaching to %s/%s...\n", spec.rigName, spec.polecatName)
			return t.AttachSession(sessionName)
		}
	}

	return fmt.Errorf("no running sessions found")
}

// runAllRun runs a command in multiple polecat sessions.
func runAllRun(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("command required")
	}

	command := args[0]
	specs, err := expandSpecs(args[1:])
	if err != nil {
		return err
	}

	if len(specs) == 0 {
		fmt.Println("No polecats match the criteria.")
		return nil
	}

	// Dry-run mode
	if allDryRun {
		fmt.Printf("Would run command in %d polecat session(s):\n", len(specs))
		fmt.Printf("  Command: %s\n\n", command)
		for _, spec := range specs {
			fmt.Printf("  - %s/%s\n", spec.rigName, spec.polecatName)
		}
		return nil
	}

	// Run command in parallel
	type runResult struct {
		spec polecatSpec
		err  error
	}

	results := make(chan runResult, len(specs))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, allParallel)

	for _, spec := range specs {
		wg.Add(1)
		go func(s polecatSpec) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			t := tmux.NewTmux()
			polecatMgr := polecat.NewSessionManager(t, s.r)

			// Check if running
			running, _ := polecatMgr.IsRunning(s.polecatName)
			if !running {
				results <- runResult{spec: s, err: fmt.Errorf("session not running")}
				return
			}

			// Send command to session
			sessionName := polecatMgr.SessionName(s.polecatName)
			err := t.SendKeys(sessionName, command)
			results <- runResult{spec: s, err: err}
		}(spec)
	}

	wg.Wait()
	close(results)

	// Collect and report results
	var succeeded int
	var errors []string

	for result := range results {
		if result.err != nil {
			errors = append(errors, fmt.Sprintf("%s/%s: %v", result.spec.rigName, result.spec.polecatName, result.err))
		} else {
			succeeded++
		}
	}

	// Summary
	fmt.Printf("\n%s Command sent to %d session(s)\n", style.SuccessPrefix, succeeded)

	if len(errors) > 0 {
		fmt.Printf("\n%s Some operations failed:\n", style.Warning.Render("Warning:"))
		for _, e := range errors {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("%d operation(s) failed", len(errors))
	}

	return nil
}
