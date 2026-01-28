package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// HookRunner manages hook configuration and execution.
type HookRunner struct {
	config        *HookConfig
	builtins      map[string]BuiltinHookFunc
	configPath    string
	mu            sync.RWMutex
	defaultTimeout time.Duration
}

// NewHookRunner creates a new HookRunner instance.
// It attempts to load configuration from standard locations.
func NewHookRunner(workingDir string) (*HookRunner, error) {
	r := &HookRunner{
		builtins:       make(map[string]BuiltinHookFunc),
		defaultTimeout: 30 * time.Second,
	}

	// Register built-in hooks
	r.registerBuiltins()

	// Try to load configuration
	if err := r.LoadConfig(workingDir); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading hook config: %w", err)
	}

	return r, nil
}

// LoadConfig loads hook configuration from standard locations.
// Checks: .gastown/hooks.json, .claude/hooks.json
func (r *HookRunner) LoadConfig(workingDir string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Try standard locations in order of preference
	searchPaths := []string{
		filepath.Join(workingDir, ".gastown", "hooks.json"),
		filepath.Join(workingDir, ".claude", "hooks.json"),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading config from %s: %w", path, err)
			}

			var cfg HookConfig
			if err := json.Unmarshal(data, &cfg); err != nil {
				return fmt.Errorf("parsing config from %s: %w", path, err)
			}

			r.config = &cfg
			r.configPath = path
			return nil
		}
	}

	// No config found, use empty config
	r.config = &HookConfig{
		Hooks: make(map[Event][]HookDefinition),
	}
	return nil
}

// Fire executes all registered hooks for the given event.
// Returns a slice of results, one per hook executed.
func (r *HookRunner) Fire(event Event, ctx *HookContext) []HookResult {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if ctx == nil {
		ctx = &HookContext{
			Event:     event,
			Timestamp: time.Now(),
			Metadata:  make(map[string]interface{}),
		}
	} else {
		ctx.Event = event
		ctx.Timestamp = time.Now()
	}

	hooks, ok := r.config.Hooks[event]
	if !ok || len(hooks) == 0 {
		return nil
	}

	results := make([]HookResult, 0, len(hooks))
	for _, hook := range hooks {
		result := r.executeHook(hook, ctx)
		results = append(results, result)

		// If this is a pre-* event and the hook blocks, stop execution
		if isPreEvent(event) && result.Block {
			break
		}
	}

	return results
}

// executeHook executes a single hook definition.
func (r *HookRunner) executeHook(hook HookDefinition, ctx *HookContext) HookResult {
	start := time.Now()

	switch hook.Type {
	case HookTypeCommand:
		return r.executeCommand(hook, ctx, start)
	case HookTypeBuiltin:
		return r.executeBuiltin(hook, ctx, start)
	default:
		return HookResult{
			Success:  false,
			Error:    fmt.Sprintf("unknown hook type: %s", hook.Type),
			Duration: time.Since(start),
		}
	}
}

// executeCommand executes a command-type hook.
func (r *HookRunner) executeCommand(hook HookDefinition, ctx *HookContext, start time.Time) HookResult {
	if hook.Cmd == "" {
		return HookResult{
			Success:  false,
			Error:    "command hook missing 'cmd' field",
			Duration: time.Since(start),
		}
	}

	// Create context with timeout
	execCtx, cancel := context.WithTimeout(context.Background(), r.defaultTimeout)
	defer cancel()

	// Prepare command
	cmd := exec.CommandContext(execCtx, "sh", "-c", hook.Cmd)
	cmd.Dir = ctx.WorkingDir

	// Pass context as environment variables
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GT_HOOK_EVENT=%s", ctx.Event),
		fmt.Sprintf("GT_HOOK_TIMESTAMP=%s", ctx.Timestamp.Format(time.RFC3339)),
	)

	// Add metadata as env vars
	for k, v := range ctx.Metadata {
		cmd.Env = append(cmd.Env, fmt.Sprintf("GT_HOOK_%s=%v", k, v))
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	result := HookResult{
		Success:  err == nil,
		Output:   stdout.String(),
		ExitCode: exitCode,
		Duration: time.Since(start),
	}

	if err != nil {
		result.Error = fmt.Sprintf("%s\nStderr: %s", err.Error(), stderr.String())
	}

	// Check if hook wants to block (non-zero exit code for pre-* events)
	if isPreEvent(ctx.Event) && exitCode != 0 {
		result.Block = true
		result.Message = fmt.Sprintf("Hook blocked operation (exit code: %d)", exitCode)
	}

	return result
}

// executeBuiltin executes a builtin-type hook.
func (r *HookRunner) executeBuiltin(hook HookDefinition, ctx *HookContext, start time.Time) HookResult {
	if hook.Name == "" {
		return HookResult{
			Success:  false,
			Error:    "builtin hook missing 'name' field",
			Duration: time.Since(start),
		}
	}

	fn, ok := r.builtins[hook.Name]
	if !ok {
		return HookResult{
			Success:  false,
			Error:    fmt.Sprintf("unknown builtin hook: %s", hook.Name),
			Duration: time.Since(start),
		}
	}

	result, err := fn(ctx)
	if err != nil {
		return HookResult{
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}
	}

	if result == nil {
		result = &HookResult{Success: true}
	}
	result.Duration = time.Since(start)
	return *result
}

// RegisterBuiltin registers a built-in hook function.
func (r *HookRunner) RegisterBuiltin(name string, fn BuiltinHookFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builtins[name] = fn
}

// ListHooks returns all hooks registered for a specific event.
// If event is empty, returns all hooks.
func (r *HookRunner) ListHooks(event Event) map[Event][]HookDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if event == "" {
		// Return all hooks
		result := make(map[Event][]HookDefinition)
		for ev, hooks := range r.config.Hooks {
			result[ev] = append([]HookDefinition{}, hooks...)
		}
		return result
	}

	// Return hooks for specific event
	result := make(map[Event][]HookDefinition)
	if hooks, ok := r.config.Hooks[event]; ok {
		result[event] = append([]HookDefinition{}, hooks...)
	}
	return result
}

// SetTimeout sets the default timeout for command hooks.
func (r *HookRunner) SetTimeout(timeout time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultTimeout = timeout
}

// ConfigPath returns the path to the loaded configuration file.
func (r *HookRunner) ConfigPath() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.configPath
}

// isPreEvent returns true if the event is a "pre-*" event.
func isPreEvent(event Event) bool {
	return event == EventPreSessionStart || event == EventPreShutdown
}

// registerBuiltins registers all built-in hook functions.
func (r *HookRunner) registerBuiltins() {
	// Register built-in hooks from builtin.go
	registerBuiltinHooks(r)
}
