package cleanup

import (
	"encoding/json"
	"fmt"

	"github.com/steveyegge/gastown/internal/hooks"
)

// RegisterCleanupHooks registers cleanup-related builtin hooks.
func RegisterCleanupHooks(runner *hooks.HookRunner) {
	runner.RegisterBuiltin("workspace-preflight", workspacePreflightHook)
	runner.RegisterBuiltin("workspace-postflight", workspacePostflightHook)
	runner.RegisterBuiltin("workspace-verify-clean", workspaceVerifyCleanHook)
	runner.RegisterBuiltin("workspace-backup", workspaceBackupHook)
}

// workspacePreflightHook performs workspace preflight checks.
func workspacePreflightHook(ctx *hooks.HookContext) (*hooks.HookResult, error) {
	workingDir := ctx.WorkingDir
	if workingDir == "" {
		return &hooks.HookResult{
			Success: false,
			Block:   true,
			Message: "working directory not specified",
		}, fmt.Errorf("working directory required")
	}

	// Determine workspace type from metadata
	workspaceType := WorkspaceTypeCrew // default
	if wsType, ok := ctx.Metadata["workspace_type"].(string); ok {
		workspaceType = WorkspaceType(wsType)
	}

	// Load configuration
	config, err := LoadConfig(workingDir, workspaceType)
	if err != nil {
		return &hooks.HookResult{
			Success: false,
			Block:   false,
			Message: fmt.Sprintf("failed to load config: %v", err),
		}, err
	}

	// Create cleaner
	cleaner := NewCleaner(workingDir, config)
	cleaner.SetDryRun(false)

	// Run preflight
	results, err := cleaner.RunPreflight()
	if err != nil {
		return &hooks.HookResult{
			Success: false,
			Block:   true,
			Message: fmt.Sprintf("preflight failed: %v", err),
		}, err
	}

	// Check if any critical checks failed
	for _, result := range results {
		if !result.Success && result.Action == ActionVerifyGitClean {
			data, _ := json.Marshal(result)
			return &hooks.HookResult{
				Success: false,
				Block:   true,
				Message: "git working directory is not clean",
				Output:  string(data),
			}, fmt.Errorf("git not clean")
		}
	}

	// Summarize results
	summary := fmt.Sprintf("Preflight completed: %d checks", len(results))
	data, _ := json.Marshal(results)

	return &hooks.HookResult{
		Success: true,
		Block:   false,
		Message: summary,
		Output:  string(data),
	}, nil
}

// workspacePostflightHook performs workspace postflight cleanup.
func workspacePostflightHook(ctx *hooks.HookContext) (*hooks.HookResult, error) {
	workingDir := ctx.WorkingDir
	if workingDir == "" {
		return &hooks.HookResult{
			Success: false,
			Message: "working directory not specified",
		}, fmt.Errorf("working directory required")
	}

	// Determine workspace type from metadata
	workspaceType := WorkspaceTypeCrew // default
	if wsType, ok := ctx.Metadata["workspace_type"].(string); ok {
		workspaceType = WorkspaceType(wsType)
	}

	// Load configuration
	config, err := LoadConfig(workingDir, workspaceType)
	if err != nil {
		return &hooks.HookResult{
			Success: false,
			Message: fmt.Sprintf("failed to load config: %v", err),
		}, err
	}

	// Create cleaner
	cleaner := NewCleaner(workingDir, config)
	cleaner.SetDryRun(false)

	// Run postflight
	results, err := cleaner.RunPostflight()
	if err != nil {
		// Postflight errors are warnings, not failures
		return &hooks.HookResult{
			Success: true,
			Message: fmt.Sprintf("postflight completed with warnings: %v", err),
		}, nil
	}

	// Summarize results
	var totalRemoved int
	var totalFreed int64
	for _, result := range results {
		totalRemoved += result.FilesRemoved
		totalFreed += result.BytesFreed
	}

	summary := fmt.Sprintf("Postflight: removed %d files, freed %.1fMB", totalRemoved, float64(totalFreed)/(1024*1024))
	data, _ := json.Marshal(results)

	return &hooks.HookResult{
		Success: true,
		Message: summary,
		Output:  string(data),
	}, nil
}

// workspaceVerifyCleanHook verifies workspace is clean.
func workspaceVerifyCleanHook(ctx *hooks.HookContext) (*hooks.HookResult, error) {
	workingDir := ctx.WorkingDir
	if workingDir == "" {
		return &hooks.HookResult{
			Success: false,
			Block:   true,
			Message: "working directory not specified",
		}, fmt.Errorf("working directory required")
	}

	// Create checker
	checker := NewPreflightChecker(workingDir)

	// Run check
	result, err := checker.Check()
	if err != nil {
		return &hooks.HookResult{
			Success: false,
			Block:   true,
			Message: fmt.Sprintf("preflight check failed: %v", err),
		}, err
	}

	if !result.Passed {
		data, _ := json.Marshal(result)
		return &hooks.HookResult{
			Success: false,
			Block:   true,
			Message: fmt.Sprintf("workspace not ready: %d issues", len(result.Issues)),
			Output:  string(data),
		}, fmt.Errorf("workspace check failed")
	}

	data, _ := json.Marshal(result)
	return &hooks.HookResult{
		Success: true,
		Block:   false,
		Message: "workspace is clean and ready",
		Output:  string(data),
	}, nil
}

// workspaceBackupHook creates a backup of uncommitted changes.
func workspaceBackupHook(ctx *hooks.HookContext) (*hooks.HookResult, error) {
	workingDir := ctx.WorkingDir
	if workingDir == "" {
		return &hooks.HookResult{
			Success: false,
			Message: "working directory not specified",
		}, fmt.Errorf("working directory required")
	}

	// Determine workspace type from metadata
	workspaceType := WorkspaceTypeCrew // default
	if wsType, ok := ctx.Metadata["workspace_type"].(string); ok {
		workspaceType = WorkspaceType(wsType)
	}

	// Load configuration
	config, err := LoadConfig(workingDir, workspaceType)
	if err != nil {
		return &hooks.HookResult{
			Success: false,
			Message: fmt.Sprintf("failed to load config: %v", err),
		}, err
	}

	// Create cleaner
	cleaner := NewCleaner(workingDir, config)

	// Backup uncommitted changes
	result, err := cleaner.backupUncommitted()
	if err != nil {
		return &hooks.HookResult{
			Success: false,
			Message: fmt.Sprintf("backup failed: %v", err),
		}, err
	}

	data, _ := json.Marshal(result)
	message := "no uncommitted changes"
	if len(result.Details) > 0 {
		message = fmt.Sprintf("backed up to %s", result.Details[0].BackupPath)
	}

	return &hooks.HookResult{
		Success: true,
		Message: message,
		Output:  string(data),
	}, nil
}
