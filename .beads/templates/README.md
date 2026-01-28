# Beads Epic Templates

Epic templates for creating batch work patterns with predefined subtask structures and variable substitution.

## Overview

Templates enable creating epics with multiple subtasks using a simple pattern-based approach. They're especially useful for:

- **Cross-rig work**: "Add feature X to all rigs"
- **Refactoring patterns**: "Replace pattern Y everywhere"
- **Feature rollouts**: Phased implementation with design, implementation, testing
- **Security audits**: Systematic review across components

## Usage

### List available templates
```bash
gt template list
```

### View template details
```bash
gt template show cross-rig-feature
gt template show refactor-pattern
```

### Create epic from template
```bash
gt template create cross-rig-feature \
  --var feature="logging middleware" \
  --var rigs="gastown,beads,aardwolf_snd"

gt template create refactor-pattern \
  --var pattern="manual error handling" \
  --var replacement="error wrapper" \
  --var locations="beads.go,routes.go,catalog.go"
```

## Template Format

Templates are TOML files with `.template.toml` extension.

### Basic Structure

```toml
name = "template-name"
description = "Template description"
version = 1

# Variables (required and optional)
[vars]
[vars.feature]
description = "Feature name"
required = true

[vars.priority]
description = "Priority level"
default = "2"

# Epic definition
[epic]
title = "Add {{feature}}"
description = "Implement {{feature}} across the project"
priority = "{{priority}}"
type = "epic"

# Subtasks
[[subtasks]]
title = "Design {{feature}}"
description = "Create design specification"
type = "task"
priority = "{{priority}}"

[[subtasks]]
title = "Implement {{feature}}"
description = "Build the feature"
type = "task"
priority = "{{priority}}"
```

### Variable Expansion

Variables are substituted using `{{varname}}` syntax:
- `{{feature}}` - Simple substitution
- `{{priority}}` - Can be used in any field
- Variables with defaults are optional

### List Expansion

Use `expand_over` to create multiple subtasks from a comma-separated list:

```toml
[vars.rigs]
description = "Comma-separated rig list"
required = true

[[subtasks]]
title = "{{rig}}: Implement feature"
description = "Add feature to {{rig}}"
expand_over = "rigs"  # Creates one subtask per rig
```

If `rigs="gastown,beads,aardwolf_snd"`, this creates 3 subtasks:
- `gastown: Implement feature`
- `beads: Implement feature`
- `aardwolf_snd: Implement feature`

The singular form of the variable is automatically derived:
- `rigs` → `rig`
- `components` → `component`
- `files` → `file`

## Built-in Templates

### cross-rig-feature
Add a feature across multiple rigs.

**Variables:**
- `feature` (required) - Feature name
- `rigs` (required) - Comma-separated rig list
- `priority` (optional) - Default: 2

**Example:**
```bash
gt template create cross-rig-feature \
  --var feature="metrics collection" \
  --var rigs="gastown,beads"
```

### refactor-pattern
Refactor a code pattern across multiple locations.

**Variables:**
- `pattern` (required) - Pattern to refactor
- `replacement` (required) - What to replace it with
- `locations` (required) - Comma-separated file/component list
- `rig` (optional) - Target rig
- `priority` (optional) - Default: 3

**Example:**
```bash
gt template create refactor-pattern \
  --var pattern="sync.Mutex" \
  --var replacement="sync.RWMutex" \
  --var locations="beads.go,catalog.go,routes.go"
```

### security-audit
Conduct security audit across components.

**Variables:**
- `scope` (required) - Security focus area
- `components` (required) - Comma-separated component list
- `priority` (optional) - Default: 1

**Example:**
```bash
gt template create security-audit \
  --var scope="input validation" \
  --var components="API,CLI,daemon"
```

### feature-rollout
Phased feature rollout with design, implementation, testing, documentation.

**Variables:**
- `feature` (required) - Feature name
- `rig` (required) - Target rig
- `priority` (optional) - Default: 2

**Example:**
```bash
gt template create feature-rollout \
  --var feature="rate limiting" \
  --var rig="gastown"
```

## Creating Custom Templates

1. Create a new `.template.toml` file in this directory
2. Define variables, epic, and subtasks
3. Use `{{varname}}` for substitution
4. Use `expand_over` for list expansion
5. Test with `gt template show <name>`
6. Create epics with `gt template create <name>`

## Tips

- **Keep subtasks focused**: Each subtask should be a single unit of work
- **Use descriptive variables**: `feature` is better than `f`
- **Provide defaults**: Optional variables with sensible defaults improve UX
- **Document examples**: Include example usage in template description
- **Test expansion**: Verify list expansion creates expected subtasks

## Integration with Workflows

Templates integrate with beads workflow:

```bash
# Create epic from template
EPIC=$(gt template create cross-rig-feature \
  --var feature="logging" \
  --var rigs="gastown,beads" | grep "Epic created" | awk '{print $4}')

# View epic and subtasks
gt show $EPIC
bd list --parent $EPIC

# Start working on subtasks
bd ready --parent $EPIC
```

## See Also

- [Formulas](.beads/formulas/) - Molecule workflow templates
- [Beads Documentation](https://github.com/steveyegge/beads)
- Issue `gt-l9g` - Epic templates implementation
