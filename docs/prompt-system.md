# Interactive Prompt System

**Issue**: gt-1u9
**Status**: Implemented
**Date**: 2026-02-03

## Overview

Implemented a comprehensive interactive prompt system to protect users from accidental data loss during destructive operations.

## Implementation

### New Package: `internal/prompt/`

Created a reusable prompt package with the following components:

#### Core Functions

1. **`Confirm(message, opts...)`** - Basic yes/no confirmation
   - Default: [y/N]
   - Returns true if user confirms

2. **`ConfirmDanger(message, opts...)`** - Danger confirmation for critical ops
   - Requires full "yes" (not just "y")
   - Red warning symbol (⚠) and bold text
   - For irreversible operations

3. **`ConfirmBatch(operation, count, opts...)`** - Batch operation confirmation
   - Shows count of affected items
   - Prevents accidental mass operations

4. **`Choice(message, options, opts...)`** - Multiple choice selection
   - Returns index of selected option
   - Returns -1 if canceled

5. **`Input(message, validator, opts...)`** - Text input with validation
   - Custom validation function
   - Allows empty to cancel

#### Bypass Options

All prompts can be bypassed via:

1. **Global `--yes/-y` flag** - Added to root command
   - `gt cleanup --yes`
   - `gt cleanup -y`

2. **Command-specific `--force` flag** - Per-command override
   - `gt cleanup --force`

3. **`GT_YES=1` environment variable** - For automation
   - `export GT_YES=1`

4. **Non-interactive mode** - Auto-detected
   - Piped input
   - Non-terminal stdin

#### Configuration Options

- `WithForce(bool)` - Skip if force flag set
- `WithYes(bool)` - Skip if yes flag set
- `WithNonInteractive(bool)` - Enable non-interactive mode
- `WithTimeout(duration)` - Timeout for automation
- `WithDefaultResponse(bool)` - Default when timeout/non-interactive

### Integrated Commands

Updated the following commands to use the prompt system:

1. **`gt cleanup`** - Clean orphaned Claude processes
   - Uses `ConfirmDanger` for process killing
   - Shows PIDs, commands, and ages before prompting

2. **`gt orphans`** - Multiple subcommands updated
   - `gt orphans kill` - Orphaned commits
   - `gt orphans procs` - Orphaned processes
   - `gt orphans zombies` - Zombie processes
   - All use `ConfirmDanger` with batch counts

3. **`gt start --shutdown`** - Shutdown sessions
   - Uses `ConfirmBatch` showing session count
   - Supports `--yes` and `--force` flags

4. **`gt uninstall`** - Remove Gas Town
   - Uses `Confirm` for normal uninstall
   - Uses `ConfirmDanger` when `--workspace` flag set
   - Extra warning for workspace deletion

5. **`gt formula`** - Formula commands
   - Replaced custom `promptYesNo` with `prompt.Confirm`
   - Maintains backward compatibility

## UX Features

### Color Coding

- **Regular prompts**: Default colors with [y/N] format
- **Danger prompts**: Red ⚠ symbol, bold text, requires "yes"
- **Batch prompts**: Shows count in message

### Preview Before Confirm

All destructive commands show what will be affected:

```
Found 3 orphaned Claude process(es):

  PID 12345 claude --mode code (age: 2h15m, tty: ttys001)
  PID 12346 claude --mode code (age: 1h05m, tty: ttys002)
  PID 12347 claude --mode code (age: 45m, tty: ttys003)

⚠ Kill these 3 process(es)? [yes/NO]
```

### Safe Defaults

- Default is always NO (safe option)
- Must type "yes" for danger operations
- Clear abort messages

### Timeout Support

For automation scenarios:

```go
prompt.Confirm("Continue?",
    prompt.WithTimeout(30*time.Second),
    prompt.WithDefaultResponse(false))
```

## Testing

Comprehensive test suite in `internal/prompt/prompt_test.go`:

- Force flag bypass
- Yes flag bypass
- Environment variable bypass
- Non-interactive mode
- Global config
- Timeout handling

All tests passing ✓

## Documentation

1. **Package documentation**: `internal/prompt/README.md`
   - Usage examples
   - Integration guide
   - Best practices

2. **API documentation**: Go doc comments
   - Exported functions
   - Configuration options
   - Examples

## Examples

### Basic Confirmation

```go
if !prompt.Confirm("Delete this file?") {
    fmt.Println("Canceled")
    return nil
}
```

### Danger Confirmation with Force

```go
if !prompt.ConfirmDanger(
    "Delete entire workspace?",
    prompt.WithForce(force),
) {
    fmt.Println("Canceled")
    return nil
}
```

### Batch Operation

```go
if !prompt.ConfirmBatch("Delete", len(files),
    prompt.WithForce(force),
    prompt.WithYes(yes),
) {
    fmt.Println("Canceled")
    return nil
}
```

### Multiple Choice

```go
choice := prompt.Choice("Select action", []string{
    "Retry",
    "Skip",
    "Abort",
})
```

## Future Enhancements

Potential additions identified but not implemented:

1. **Undo information** - Show if operation can be undone
2. **Dry-run preview** - Integration with dry-run flags
3. **Progress indicators** - For long-running confirmations
4. **Sound/visual alerts** - For critical prompts
5. **Confirmation history** - Track what was confirmed

## Acceptance Criteria

✅ Destructive ops prompt by default
✅ --force or --yes bypasses prompts
✅ Clear prompt text explaining action
✅ Non-interactive mode (piped input) auto-fails
✅ Preview of what will be affected
✅ Color coding for danger operations
✅ Confirmation count for batch ops
✅ Global --yes/-y flag
✅ GT_YES environment variable
✅ Tests and documentation

## Files Changed

### New Files
- `/Users/ericfriday/gt/internal/prompt/prompt.go` - Main implementation
- `/Users/ericfriday/gt/internal/prompt/prompt_test.go` - Test suite
- `/Users/ericfriday/gt/internal/prompt/README.md` - Package documentation
- `/Users/ericfriday/gt/docs/prompt-system.md` - This document

### Modified Files
- `/Users/ericfriday/gt/internal/cmd/root.go` - Added global --yes flag
- `/Users/ericfriday/gt/internal/cmd/cleanup.go` - Integrated prompts
- `/Users/ericfriday/gt/internal/cmd/orphans.go` - Integrated prompts
- `/Users/ericfriday/gt/internal/cmd/start.go` - Integrated prompts
- `/Users/ericfriday/gt/internal/cmd/uninstall.go` - Integrated prompts
- `/Users/ericfriday/gt/internal/cmd/formula.go` - Integrated prompts

## Related Issues

- gt-1u9: Interactive prompts for destructive operations ✅ RESOLVED
