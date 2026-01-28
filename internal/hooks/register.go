package hooks

// RegisterWorkspaceCleanupHooks is a placeholder for workspace cleanup hooks registration.
// The actual registration is done via cleanup.RegisterCleanupHooks() to avoid circular imports.
// This function exists to document the integration point.
//
// Usage:
//   import "github.com/steveyegge/gastown/internal/workspace/cleanup"
//
//   runner, _ := hooks.NewHookRunner("/path/to/workspace")
//   cleanup.RegisterCleanupHooks(runner)
//
// Available hooks:
//   - workspace-preflight: Run preflight checks
//   - workspace-postflight: Run postflight cleanup
//   - workspace-verify-clean: Verify workspace is clean
//   - workspace-backup: Backup uncommitted changes
func RegisterWorkspaceCleanupHooks(r *HookRunner) {
	// Note: Actual registration is done in workspace/cleanup package
	// to avoid circular imports. See cleanup.RegisterCleanupHooks().
	_ = r
}
