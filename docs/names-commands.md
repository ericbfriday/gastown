# Names Commands Documentation

The `gt names` command suite manages the polecat naming pool, which provides themed names for automatically spawned polecats.

## Overview

Polecats get names from a themed pool (mad-max, minerals, wasteland). The naming pool tracks:
- **Available names**: Ready for allocation to new polecats
- **In-use names**: Currently assigned to active polecats
- **Reserved names**: Infrastructure agents that cannot be allocated (witness, mayor, deacon, refinery)
- **Custom names**: User-added names that supplement the theme

## Commands

### gt names list

Shows all names in the pool with their status.

```bash
gt names list
```

**Output:**
```
Name Pool: gastown
Theme: mad-max

Reserved Names (Infrastructure)
  witness
  mayor
  deacon
  refinery

In Use (3)
  furiosa
  nux
  capable

Available (47)
  toast
  dag
  cheedo
  ...
```

**Use when:**
- Checking pool capacity before spawning polecats
- Seeing which names are currently in use
- Finding available names for manual polecat creation

---

### gt names add <name>

Adds a custom name to the pool.

```bash
gt names add obsidian
```

**Validation:**
- Alphanumeric characters, hyphens, and underscores only
- Maximum 32 characters
- Cannot be a reserved infrastructure name
- Cannot duplicate existing name

**Use when:**
- Adding project-specific names (e.g., service names, feature names)
- Supplementing theme names with custom names
- Reserving specific names for special-purpose polecats

**Example:**
```bash
gt names add prometheus
gt names add grafana
gt names add alertmanager
```

---

### gt names remove <name>

Removes a custom name from the pool.

```bash
gt names remove obsidian
```

**Safety checks:**
- Only custom names can be removed (themed names are immutable)
- Name must not be in use (unless `--force` is specified)
- Reserved names cannot be removed

**Flags:**
- `--force`: Remove even if currently in use

**Use when:**
- Cleaning up unused custom names
- Removing deprecated names
- Reorganizing the pool

**Example:**
```bash
# Safe removal (fails if in use)
gt names remove obsolete-name

# Force removal (removes even if in use)
gt names remove old-service --force
```

---

### gt names reserve <name>

Reserves a name to prevent allocation to polecats.

```bash
gt names reserve prometheus
```

**Purpose:**
- Prevent automatic allocation of names needed for infrastructure
- Reserve names for special-purpose agents
- Protect names from being used by transient polecats

**Persisted in:** `<rig>/settings/config.json` under `namepool.reserved_names`

**Use when:**
- Setting up infrastructure agents with stable names
- Preventing confusion between polecats and services
- Reserving names for future use

**Example:**
```bash
# Reserve monitoring service names
gt names reserve prometheus
gt names reserve grafana
gt names reserve alertmanager
```

---

### gt names stats

Shows detailed statistics about the naming pool.

```bash
gt names stats
```

**Output:**
```
Name Pool Statistics: gastown

Configuration
  Theme:          mad-max
  Max Pool Size:  50
  Custom Names:   3

Capacity
  Total Names:    53
  Available:      45 (84%)
  In Use:         8 (15%)
  Reserved:       4

Usage Patterns
  Pooled Names:   8
  Overflow Names: 0
```

**Metrics explained:**
- **Total Names**: All themed + custom names
- **Available**: Names ready for allocation
- **In Use**: Names currently assigned to polecats
- **Reserved**: Infrastructure names that cannot be allocated
- **Pooled Names**: In-use names from theme or custom pool
- **Overflow Names**: In-use auto-generated names (rigname-N format)

**Use when:**
- Monitoring pool capacity
- Detecting pool exhaustion
- Planning pool expansion

---

## Pool Exhaustion

When all themed/custom names are in use, the pool allocates overflow names in the format `<rigname>-<N>`.

**Example:**
```
gastown-51
gastown-52
gastown-53
```

**Overflow names:**
- Are NOT reusable (never return to the pool)
- Increment indefinitely (51, 52, 53, ...)
- Indicate the pool needs expansion

**Recovery:**
```bash
# Check overflow usage
gt names stats

# Add more custom names
gt names add name1 name2 name3

# Or switch to a different theme
gt namepool set minerals
```

---

## Configuration

### Rig Settings

Name pool configuration is stored in `<rig>/settings/config.json`:

```json
{
  "namepool": {
    "style": "mad-max",
    "names": ["custom1", "custom2"],
    "reserved_names": ["prometheus", "grafana"],
    "max_before_numbering": 50
  }
}
```

**Fields:**
- `style`: Theme name (mad-max, minerals, wasteland)
- `names`: Custom names (supplement theme names)
- `reserved_names`: Names protected from allocation
- `max_before_numbering`: Pool size threshold for overflow

### Runtime State

Pool state is tracked in `<rig>/.runtime/namepool-state.json`:

```json
{
  "rig_name": "gastown",
  "overflow_next": 51,
  "max_size": 50
}
```

**Note:** `InUse` status is NOT persisted - it's derived from existing polecat directories.

---

## Themes

Available themes:

| Theme | Names | Style |
|-------|-------|-------|
| `mad-max` | furiosa, nux, toast, capable, ... | Mad Max universe (default) |
| `minerals` | obsidian, quartz, jasper, onyx, ... | Minerals and gemstones |
| `wasteland` | rust, chrome, nitro, vault, ... | Post-apocalyptic |

**Change theme:**
```bash
gt namepool set minerals
```

See `gt namepool themes` for complete theme listings.

---

## Best Practices

### 1. Reserve Infrastructure Names

```bash
gt names reserve witness
gt names reserve mayor
gt names reserve deacon
gt names reserve refinery
```

### 2. Add Service Names

For service-oriented projects, add service names:
```bash
gt names add api-gateway
gt names add auth-service
gt names add user-service
```

### 3. Monitor Capacity

Check pool stats periodically:
```bash
gt names stats
```

If overflow names appear, add more custom names or switch themes.

### 4. Clean Up Unused Names

Remove deprecated custom names:
```bash
gt names list  # Review custom names
gt names remove obsolete-name
```

---

## Integration with Polecats

The naming pool is automatically used when spawning polecats:

```bash
# Automatic name allocation
gt polecat add  # Gets next available name from pool

# Manual name specification
gt polecat add prometheus  # Uses specific name (bypasses pool)
```

**Lifecycle:**
1. `gt polecat add` → Allocates name from pool
2. Polecat works → Name marked "in use"
3. `gt polecat nuke <name>` → Name returns to pool (if themed/custom)
4. Name available for next polecat

---

## Troubleshooting

### "Pool exhausted - using overflow naming"

**Symptom:** Polecats get names like `gastown-51`

**Solution:**
```bash
# Option 1: Add custom names
gt names add name1 name2 name3

# Option 2: Switch to different theme
gt namepool set minerals

# Option 3: Clean up unused polecats
gt polecat list
gt polecat nuke <unused-name>
```

### "Name already in pool"

**Symptom:** `gt names add` says name exists

**Solution:**
```bash
# Check if it's a themed name
gt namepool themes mad-max  # Shows all themed names

# If custom, no action needed (already added)
```

### "Cannot remove themed name"

**Symptom:** `gt names remove` fails for theme name

**Solution:**
Themed names are immutable. Only custom names can be removed.

---

## Examples

### Setup Infrastructure Names

```bash
# Reserve infrastructure agent names
gt names reserve witness
gt names reserve mayor
gt names reserve deacon
gt names reserve refinery

# Add monitoring service names
gt names add prometheus
gt names add grafana
gt names add alertmanager

# Verify configuration
gt names list
```

### Expand Pool Capacity

```bash
# Check current usage
gt names stats

# Add custom names for services
gt names add api-gateway
gt names add auth-service
gt names add user-service
gt names add payment-service
gt names add notification-service

# Verify capacity
gt names stats
```

### Switch Theme

```bash
# View current theme
gt namepool

# List available themes
gt namepool themes

# Switch to minerals theme
gt namepool set minerals

# Verify new theme
gt names list
```

---

## See Also

- `gt namepool` - Theme management
- `gt polecat add` - Spawn polecats with automatic naming
- `gt polecat list` - View active polecats and their names
