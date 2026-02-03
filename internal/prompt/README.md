# Prompt Package

Interactive prompt system for destructive operations in Gas Town CLI.

## Overview

The `prompt` package provides consistent, user-friendly confirmation prompts for destructive operations. It prevents accidental data loss by requiring explicit user confirmation before executing dangerous commands.

## Features

- **Yes/No Confirmations** - Simple yes/no prompts
- **Danger Confirmations** - Red-colored prompts requiring full "yes" for critical operations
- **Batch Operations** - Shows count of affected items
- **Multiple Choice** - Select from a list of options
- **Text Input** - Validated text input
- **Bypass Options** - `--yes/-y` flag, `--force` flag, or `GT_YES=1` environment variable
- **Timeout Support** - For automation scenarios
- **Non-Interactive Mode** - Respects piped input and automation contexts

## Usage

### Basic Confirmation

```go
import "github.com/steveyegge/gastown/internal/prompt"

if !prompt.Confirm("Delete this file?") {
    fmt.Println("Canceled")
    return nil
}
```

### Danger Confirmation

For critical operations that require explicit "yes" (not just "y"):

```go
if !prompt.ConfirmDanger("Delete entire workspace?") {
    fmt.Println("Canceled")
    return nil
}
```

### Batch Operations

Shows count of items being affected:

```go
if !prompt.ConfirmBatch("Delete", fileCount) {
    fmt.Println("Canceled")
    return nil
}
```

### With Force Flag

```go
if !prompt.Confirm("Delete file?", prompt.WithForce(force)) {
    fmt.Println("Canceled")
    return nil
}
```

### Multiple Options

```go
choice := prompt.Choice("Select action", []string{
    "Retry",
    "Skip",
    "Abort",
})

switch choice {
case 0:
    // Retry
case 1:
    // Skip
case 2:
    // Abort
default:
    // Canceled
}
```

### Text Input with Validation

```go
validator := func(s string) error {
    if len(s) < 3 {
        return fmt.Errorf("name must be at least 3 characters")
    }
    return nil
}

name, ok := prompt.Input("Enter name", validator)
if !ok {
    fmt.Println("Canceled")
    return nil
}
```

## Bypass Methods

### 1. Global --yes Flag

```bash
gt cleanup --yes           # Skip all prompts
gt cleanup -y              # Short form
```

### 2. Command-Specific --force Flag

```bash
gt cleanup --force         # Skip prompts for this command
```

### 3. Environment Variable

```bash
export GT_YES=1
gt cleanup                 # Automatically says yes to all prompts
```

### 4. Non-Interactive Mode

When stdin is not a terminal (piped input), prompts return the default response:

```bash
echo "data" | gt cleanup   # Returns default response
```

## Options

All prompt functions accept options:

- `WithForce(bool)` - Skip prompt if force is true
- `WithYes(bool)` - Skip prompt if yes is true
- `WithNonInteractive(bool)` - Enable non-interactive mode
- `WithTimeout(duration)` - Set timeout for automation
- `WithDefaultResponse(bool)` - Set default when timeout/non-interactive

## Integration with Commands

### Command Structure

```go
var (
    cleanupForce bool
    cleanupYes   bool
)

var cleanupCmd = &cobra.Command{
    Use:   "cleanup",
    Short: "Clean up orphaned resources",
    RunE:  runCleanup,
}

func init() {
    cleanupCmd.Flags().BoolVarP(&cleanupForce, "force", "f", false,
        "Skip confirmation prompts")
    cleanupCmd.Flags().BoolVarP(&cleanupYes, "yes", "y", false,
        "Skip confirmation prompts")
}

func runCleanup(cmd *cobra.Command, args []string) error {
    // Show what will be deleted
    fmt.Printf("Found %d orphaned resources\n", count)

    // Prompt for confirmation
    if !prompt.ConfirmDanger(
        fmt.Sprintf("Delete %d resource(s)?", count),
        prompt.WithForce(cleanupForce),
        prompt.WithYes(cleanupYes),
    ) {
        fmt.Println("Canceled")
        return nil
    }

    // Proceed with deletion
    deleteResources()
    return nil
}
```

### Global --yes Flag

The root command has a persistent `--yes/-y` flag that applies to all subcommands. This is handled automatically via `prompt.GlobalConfig` in the `persistentPreRun` hook.

## Color Coding

- **Regular prompts**: Default terminal colors
- **Danger prompts**: Red warning symbol (⚠) and bold text
- **Batch prompts**: Shows count in message

## Best Practices

1. **Use ConfirmDanger for irreversible operations**
   - Database deletions
   - Workspace removal
   - Batch deletions
   - Force operations

2. **Use Confirm for reversible operations**
   - Session stops
   - Temporary file cleanup
   - Cache clearing

3. **Use ConfirmBatch when affecting multiple items**
   - Shows count to user
   - Prevents accidental mass deletions

4. **Always show what will be affected before prompting**
   ```go
   // Good
   fmt.Printf("Will delete: %v\n", items)
   if !prompt.Confirm("Proceed?") { ... }

   // Bad - user doesn't know what they're confirming
   if !prompt.Confirm("Delete items?") { ... }
   ```

5. **Support both --force and --yes**
   ```go
   prompt.Confirm("...",
       prompt.WithForce(force),
       prompt.WithYes(yes))
   ```

## Examples from Codebase

### Cleanup Command

```go
// Show what will be deleted
for _, proc := range orphans {
    fmt.Printf("  PID %d: %s\n", proc.PID, proc.Cmd)
}

// Confirm deletion
if !prompt.ConfirmDanger(
    fmt.Sprintf("Kill these %d process(es)?", len(orphans)),
    prompt.WithForce(force),
) {
    fmt.Println("Aborted")
    return nil
}
```

### Uninstall Command

```go
// Show what will be removed
fmt.Println("Will remove:")
fmt.Printf("  • Shell integration\n")
fmt.Printf("  • State directory\n")

// Use danger prompt for workspace deletion
if workspace {
    if !prompt.ConfirmDanger("Continue?", prompt.WithForce(force)) {
        return nil
    }
} else {
    if !prompt.Confirm("Continue?", prompt.WithForce(force)) {
        return nil
    }
}
```

## Testing

The package includes comprehensive tests:

```bash
cd internal/prompt
go test -v
```

Tests cover:
- Force flag bypass
- Yes flag bypass
- Environment variable bypass
- Non-interactive mode
- Timeout handling
- Global config

## Environment Variables

- `GT_YES=1` - Skip all prompts (auto-yes)

## Related Issues

- gt-1u9: Interactive prompts for destructive operations
