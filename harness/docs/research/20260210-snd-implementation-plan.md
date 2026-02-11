# Search and Destroy Mudlet Port - Implementation Plan

**Issue:** as-ckq
**Date:** 2026-02-10
**Based on:** Architecture analysis (as-1tn), existing PRD, porting blueprint
**Status:** Planning

---

## 1. Porting Strategy

### 1.1 Approach: WinkleWinkle-First, Crowley-Enhanced

The port should target **WinkleWinkle's modular architecture** (~3,500 LOC across 3 plugins) as the primary source, augmented by specific features from **Crowley's comprehensive version** (10,110 LOC). Rationale:

1. **WinkleWinkle's modular split** (Search_Destroy_2 + Mapper_Extender_2 + Extender_GUI_2) maps directly to the port's existing module structure
2. **Crowley's monolith** is harder to decompose but has superior mob keyword learning, combat tracking, and GQ support
3. **Feature parity with WinkleWinkle** = a usable tool for daily Aardwolf play. Crowley features are enhancements, not requirements

### 1.2 Technical Principles

| Principle | Rationale |
|-----------|-----------|
| **Direct port** for core algorithms | Hunt trick, quick where, campaign — the logic is correct in the original, just needs API translation |
| **Redesign** for triggers | MUSHclient XML triggers → Mudlet `tempRegexTrigger()` / `tempExactMatchTrigger()`. 1:1 mapping, but lifecycle differs |
| **Redesign** for GUI | MUSHclient miniwindows → Mudlet Geyser framework. Completely different API, same UX goals |
| **Preserve** existing architecture | The port's event-driven, decoupled design is correct. Build within it, don't restructure |
| **Mock-first development** | Continue using `core.lua` mock layer. Every feature must work in mock mode for testing |

---

## 2. Priority-Ordered Module Plan

### Phase 1: Core Hunt Loop (P1 Critical)

The hunt trick is the #1 daily-use feature. Getting this working end-to-end proves the entire trigger/alias/event pipeline.

#### M1.1 — Hunt Triggers & Direction Parsing

**Source:** WinkleWinkle `Search_Destroy_2.xml` hunt trick section + Crowley `do_hunt_trick()`
**Target:** `hunt.lua` (expand from 63 → ~200 lines)
**Approach:** Direct port with Mudlet API translation

**Work items:**
1. Parse all Aardwolf hunt responses:
   - `"You seem unable to hunt that target."` — mob doesn't exist
   - `"You couldn't find a path to <mob> from here."` — unreachable
   - `"<mob> is in this room!"` — found, same room
   - `"You can smell <mob> nearby to the <direction>."` — nearby
   - `"You sense the trail of <mob> leading <direction> from here."` — distant
   - `"The trail seems to disappear..."` — lost trail
2. Add direction extraction and confidence levels (here/nearby/distant/lost)
3. Implement auto-advance: on "hunting" success, increment index and re-hunt after configurable delay
4. Add combat abort: detect combat entry via GMCP `char.status.state == 8` and pause hunt
5. Add `SnD.Hunt.onTrigger(response)` as central dispatch for all hunt response types
6. Use `tempRegexTrigger` with one-shot (destroy after fire) for each hunt attempt

**Testing strategy:**
- Mock tests: Feed each hunt response string through `SnD.Hunt.onTrigger()`, verify state transitions
- Live test: Hunt for common mob (e.g., "guard" in Aylor), verify index cycling works
- Edge case: Hunt for nonexistent mob, verify clean abort

**Estimated LOC:** ~140 new lines in hunt.lua

#### M1.2 — Auto-Hunt Movement

**Source:** WinkleWinkle `do_auto_hunt()` + Crowley `auto_hunt()`
**Target:** `hunt.lua` (add auto-hunt section, or new `auto_hunt.lua` ~150 lines)
**Approach:** Direct port

**Work items:**
1. Directional movement: parse "nearby to the north" → `send("north")`
2. Direction map: n/s/e/w/ne/nw/se/sw/u/d
3. Re-hunt after movement with configurable delay (default 0.5s)
4. Combat detection: stop auto-hunt when entering combat
5. Stuck detection: abort after N consecutive "trail disappears" responses
6. Add `ah <mob>` and `ah abort` aliases
7. Integration with hunt trick: auto-hunt can call `SnD.Hunt.execute()` internally

**Testing strategy:**
- Mock: Simulate direction response → verify correct `send()` call
- Mock: Simulate combat entry → verify hunt pauses
- Live: `ah guard` in a safe area, verify automated movement to target

**Estimated LOC:** ~150 new lines

#### M1.3 — Critical Aliases

**Target:** `aliases.lua` (expand from 23 → ~60 lines)
**Approach:** New code

**Work items:**
1. `ht [N.]<mob>` — hunt trick with optional index prefix
2. `ht abort` — abort current hunt
3. `ah <mob>` — start auto-hunt
4. `ah abort` — stop auto-hunt
5. `ak` / `kk` — quick kill (send configured kill command)
6. `qs` — quick scan (send "scan")

**Testing strategy:**
- Mock: Verify alias regex patterns match expected inputs
- Mock: Verify correct function dispatch

**Estimated LOC:** ~40 new lines

**Phase 1 Acceptance Criteria:**
- [ ] `ht guard` cycles through 1.guard, 2.guard, ... until "not found" or "found here"
- [ ] `ah guard` auto-walks toward target, re-hunting after each move
- [ ] Combat entry pauses hunt/auto-hunt
- [ ] `ht abort` / `ah abort` cleanly stop active operations
- [ ] `ak` sends kill command
- [ ] All mock tests pass
- [ ] Manual test against live Aardwolf confirms hunt trick works

---

### Phase 2: Campaign & Quick Where (P1 High)

Campaign navigation is the #2 daily-use feature. Quick where supports both campaigns and general mob finding.

#### M2.1 — Campaign GMCP Parsing

**Source:** Crowley `do_campaign()` + WinkleWinkle campaign handling
**Target:** `gmcp.lua` (expand from 17 → ~80 lines) + `campaign.lua` (expand from 48 → ~150 lines)
**Approach:** Direct port of GMCP parsing, redesign display

**Work items:**
1. Parse `gmcp.comm.quest` for campaign data:
   - Action field: "accepted", "completed", "failed", "target"
   - Target fields: name, room, area, level
2. Parse `gmcp.char.status` for character state tracking:
   - State codes: 1=login, 2=motd, 3=active, 5=sleeping, 6=resting, 7=fighting, 8=stunned, 9=AFK, 11=note, 12=edit
3. Track campaign targets: alive vs dead, area vs room
4. `cp check` alias: send "cp check" and parse multi-line response
5. Level range detection per area
6. Integrate with database: persist campaign state across sessions

**Testing strategy:**
- Mock: Feed sample GMCP campaign JSON → verify target list populated
- Mock: Feed "target killed" event → verify dead tracking
- Mock: Verify database persistence round-trip
- Live: Accept a campaign, verify `xcp` shows targets correctly

**Estimated LOC:** ~120 new lines across gmcp.lua + campaign.lua

#### M2.2 — Quick Where Enhancement

**Source:** WinkleWinkle `do_quick_where()` + Crowley `where_mob()`
**Target:** `quick_where.lua` (expand from 43 → ~120 lines)
**Approach:** Direct port

**Work items:**
1. Parse all Aardwolf "where" response formats:
   - Single result: `"<mob> is in <room> [<area>]"`
   - Multiple results: multi-line output with room list
   - Not found: `"Nobody by that name around"`
2. Extract room name and match to Mudlet mapper room IDs
3. Multi-result handling: store results, support `go N` / `nx` navigation
4. Clickable links in output (Mudlet `cechoLink()`)
5. Area filtering: show only results in current area optionally

**Testing strategy:**
- Mock: Feed single/multi/not-found responses → verify parsing
- Mock: Verify room ID matching against mock map data
- Live: `qw guard` in Aylor, verify room identification

**Estimated LOC:** ~80 new lines

#### M2.3 — Room Search (xm/xmall)

**Source:** WinkleWinkle `do_xm()` / `do_xmall()` + Crowley `xm_room()` / `xmall_room()`
**Target:** `mapper.lua` (expand from 45 → ~150 lines)
**Approach:** Direct port

**Work items:**
1. `xm <room>` — search for room by name in current area
2. `xmall <room>` — search for room by name across all areas
3. `go [N]` — navigate to Nth search result (default: first)
4. `nx` — navigate to next search result
5. Store search results in `SnD.Mapper.searchResults[]`
6. Display formatted results with room IDs and area names

**Testing strategy:**
- Mock: Search for room name → verify results list populated
- Mock: `go 2` → verify navigation to correct room
- Live: `xm recall` in Aylor, verify room match

**Estimated LOC:** ~100 new lines

**Phase 2 Acceptance Criteria:**
- [ ] Campaign targets display correctly from GMCP data
- [ ] `xcp 1` navigates to first campaign target (xrunto area + quick where)
- [ ] Dead targets tracked and displayed with status
- [ ] `qw <mob>` shows results with room identification
- [ ] `xm <room>` finds rooms in current area
- [ ] `xmall <room>` finds rooms globally
- [ ] `go` / `nx` navigate search results
- [ ] All mock tests pass

---

### Phase 3: GUI & Display (P2 Medium)

The GUI makes the tool visually accessible. Port targets WinkleWinkle's miniwindow design.

#### M3.1 — Campaign GUI

**Source:** WinkleWinkle `Extender_GUI_2.xml`
**Target:** `campaign_gui.lua` (expand from 28 → ~120 lines) + `gui.lua` (expand from 42 → ~100 lines)
**Approach:** Redesign for Geyser

**Work items:**
1. Campaign target list with color-coded status (alive=green, dead=red, navigating=yellow)
2. Clickable targets (click to navigate)
3. Area grouping in display
4. Level range display per target
5. GQ mode (switch display for Global Quest targets)
6. Window resize persistence (save to config DB)
7. Minimize/maximize toggle

**Testing strategy:**
- Mock: Verify Geyser API calls for window creation
- Mock: Feed campaign data → verify label updates
- Visual: Manual inspection in Mudlet

**Estimated LOC:** ~150 new lines

#### M3.2 — Button Panel

**Source:** WinkleWinkle `Extender_GUI_2.xml` buttons
**Target:** `buttons.lua` (expand from 43 → ~80 lines)
**Approach:** Redesign for Geyser

**Work items:**
1. Settings panel with toggles: auto-hunt on/off, kill command selection, speed walk
2. Quick action buttons: CP Check, GQ Check, Hunt Abort
3. Status indicator: current hunt state, campaign progress

**Testing strategy:**
- Mock: Verify button click callbacks dispatch correctly
- Visual: Manual inspection in Mudlet

**Estimated LOC:** ~40 new lines

**Phase 3 Acceptance Criteria:**
- [ ] Campaign targets display in GUI window with correct status colors
- [ ] Clicking a target navigates to it
- [ ] Window remembers size/position across sessions
- [ ] Buttons trigger correct actions
- [ ] GQ and CP modes switchable

---

### Phase 4: Advanced Features (P2-P3)

#### M4.1 — Mob Keyword Learning

**Source:** Crowley `gmkw()` + WinkleWinkle `guess_mob_name()`
**Target:** New `mob_keywords.lua` (~150 lines) + database.lua additions
**Approach:** Port Crowley's approach (SQLite-backed)

**Work items:**
1. Extract mob keywords from GMCP `char.status.enemy` on combat start
2. Store keyword → full name mapping in database
3. Use learned keywords for hunt/quick where when exact name unknown
4. Manual `gmkw <keyword> <mobname>` alias for corrections
5. Export/import keyword database

**Estimated LOC:** ~150 new lines

#### M4.2 — State Machine Enhancement

**Source:** Crowley's comprehensive state tracking
**Target:** `gmcp.lua` (add ~80 lines)
**Approach:** Direct port of state codes

**Work items:**
1. Track full character state from GMCP `char.status`
2. Activity detection: idle, hunting, fighting, questing, camping
3. Block conflicting operations (e.g., don't hunt while fighting)
4. State change events for other modules to subscribe to

**Estimated LOC:** ~80 new lines

#### M4.3 — GQ System

**Source:** Crowley GQ tracking
**Target:** New `global_quest.lua` (~100 lines)
**Approach:** Direct port

**Work items:**
1. Parse `gmcp.comm.quest` for GQ data
2. `gq check` / `gq info` aliases
3. GQ target tracking (similar to campaign)
4. Timer display for GQ deadline
5. Integration with GUI (GQ mode toggle)

**Estimated LOC:** ~100 new lines

#### M4.4 — Settings System

**Target:** `database.lua` additions + new `settings.lua` (~80 lines)
**Approach:** New code

**Work items:**
1. `xset` alias framework for all settings
2. Settings: kill_command, hunt_delay, speed_walk, pk_rooms, vidblain_handling
3. Persist all settings in database config table (already exists)
4. Settings panel in GUI

**Estimated LOC:** ~80 new lines

**Phase 4 Acceptance Criteria:**
- [ ] Mob keywords learned from combat and reused in hunts
- [ ] Character state tracked, conflicting operations blocked
- [ ] GQ tracking works like campaign tracking
- [ ] All settings configurable via `xset` and persisted

---

### Phase 5: Polish & Edge Cases (P3)

#### M5.1 — Room Notes
- `roomnote <text>` — add note to current room
- `roomnote area` — list all room notes in current area
- Store in room user data via Mapper API

#### M5.2 — Vidblain Handling
- Detect vidblain portal rooms
- Adjusted navigation logic

#### M5.3 — Combat Damage Tracking
- Port subset of Crowley's 50+ damage verb triggers
- Track last-attacked mob for hunt targeting

#### M5.4 — Auto No-Exp
- Monitor TNL threshold
- Auto-toggle noexp when close to leveling during runs

#### M5.5 — Sound Alerts
- Port target_nearby.wav and other_target_here.wav
- Play on hunt found / campaign target nearby

**Phase 5 Acceptance Criteria:**
- [ ] Room notes persist and display
- [ ] Edge case navigation works (vidblain, unmapped areas)
- [ ] Optional sound alerts for key events

---

## 3. Testing Strategy

### 3.1 Test Infrastructure Decision

**Problem:** Current project has vestigial Nx/pnpm scaffolding but no working test runner. All code is Lua targeting Mudlet's Lua 5.1 runtime.

**Decision:** Use the existing mock framework in `core.lua` + `mock_data.lua` as the test harness. Extend it rather than introducing a new Lua test framework (like busted).

**Rationale:**
- `core.lua` already mocks all Mudlet APIs (send, cecho, tempRegexTrigger, etc.)
- `mock_data.lua` provides sample GMCP data, area tables, room data
- Adding busted adds a dependency and requires Lua 5.1 compatibility verification
- The mock approach tests the actual code that runs in Mudlet

### 3.2 Test Layers

| Layer | What | How | When |
|-------|------|-----|------|
| **Unit tests** | Individual functions in isolation | Call function with mock data, check return/state | Every module |
| **Integration tests** | Multi-module workflows | Simulate trigger→function→event→handler chain | Each phase |
| **Manual Mudlet tests** | End-to-end in Mudlet runtime | Load package, test against Aardwolf | Each phase milestone |
| **Regression tests** | Prevent breakage | Re-run all unit/integration tests on changes | Before each commit |

### 3.3 Test File Structure

```
src/
  tests/
    test_hunt.lua          -- Hunt trick unit tests
    test_auto_hunt.lua     -- Auto-hunt unit tests
    test_campaign.lua      -- Campaign parsing tests
    test_quick_where.lua   -- Quick where parsing tests
    test_mapper.lua        -- Room search tests
    test_gmcp.lua          -- GMCP event handling tests
    test_database.lua      -- Database CRUD tests
    test_integration.lua   -- Cross-module workflow tests
    test_runner.lua        -- Simple test runner (assert-based)
```

### 3.4 Test Runner

Simple Lua test runner using assert:

```lua
-- test_runner.lua pattern:
local pass, fail = 0, 0
function test(name, fn)
    local ok, err = pcall(fn)
    if ok then pass = pass + 1
    else fail = fail + 1; print("FAIL: " .. name .. " - " .. err) end
end
-- Run all test files, report pass/fail
```

Runnable with: `lua5.1 src/tests/test_runner.lua` (outside Mudlet) or loaded via Mudlet's script editor.

### 3.5 Mock Testing Patterns

**Trigger simulation:**
```lua
-- Simulate hunt response
SnD.Hunt.execute("guard")
-- Fire the trigger callback directly
SnD.Hunt.onTrigger("You can smell guard nearby to the north.")
assert(SnD.Hunt.lastDirection == "north")
assert(SnD.Hunt.confidence == "nearby")
```

**GMCP simulation:**
```lua
-- Simulate campaign GMCP
gmcp.comm.quest = { action = "target", name = "a goblin", area = "Goblin Caverns" }
SnD.handlers.questUpdated()
assert(#SnD.state.campaignTargets == 1)
```

---

## 4. Integration Plan

### 4.1 Module Load Order

Mudlet loads scripts in order. The port must maintain this load order:

```
1. core.lua           -- Namespace init, mocks, event system
2. database.lua       -- Database init (depends on core)
3. gmcp.lua           -- GMCP handlers (depends on core)
4. hunt.lua           -- Hunt logic (depends on core)
5. quick_where.lua    -- Quick where (depends on core)
6. mapper.lua         -- Mapper extensions (depends on core)
7. campaign.lua       -- Campaign logic (depends on mapper, database)
8. realtime.lua       -- Quest routing (depends on campaign, database)
9. gui.lua            -- GUI framework (depends on core)
10. buttons.lua       -- Button panel (depends on gui)
11. campaign_gui.lua  -- Campaign display (depends on gui)
12. aliases.lua       -- All aliases (depends on all modules)
13. [test files]      -- Test only, not loaded in production
```

### 4.2 Integration Points

| Source Module | Event/Call | Target Module | Description |
|---------------|-----------|---------------|-------------|
| gmcp.lua | `raiseEvent("roomUpdated")` | mapper.lua, campaign.lua | Room change detection |
| gmcp.lua | `raiseEvent("questUpdated")` | campaign.lua, realtime.lua | Campaign/GQ data received |
| gmcp.lua | `raiseEvent("stateChanged")` | hunt.lua | Combat detection for hunt abort |
| hunt.lua | `raiseEvent("huntCompleted")` | gui.lua, campaign.lua | Hunt result notification |
| campaign.lua | `raiseEvent("campaignsUpdated")` | campaign_gui.lua | Target list refresh |
| quick_where.lua | `raiseEvent("whereResult")` | mapper.lua, campaign.lua | Room found from where |
| mapper.lua | `raiseEvent("navigationStarted")` | gui.lua | Navigation status update |
| aliases.lua | Direct calls | All modules | User command dispatch |

### 4.3 Mudlet Package Structure

```xml
<!-- Search_and_Destroy.xml (Mudlet package format) -->
<MudletPackage version="1.001">
  <TriggerPackage />
  <TimerPackage />
  <AliasPackage>
    <!-- All aliases from aliases.lua -->
  </AliasPackage>
  <KeyPackage />
  <ScriptPackage>
    <!-- All .lua files in load order -->
    <Script name="SnD-Core">core.lua</Script>
    <Script name="SnD-Database">database.lua</Script>
    <!-- ... etc -->
  </ScriptPackage>
</MudletPackage>
```

---

## 5. Milestones & Timeline

### Milestone 1: Hunting Works (Phase 1)

**Goal:** Hunt trick and auto-hunt fully functional
**Modules:** hunt.lua, aliases.lua (hunt aliases), gmcp.lua (combat state)
**Estimated effort:** ~330 new LOC
**Definition of done:**
- `ht guard` finds guard via indexed hunt trick
- `ah guard` auto-walks to guard
- Combat aborts hunt/auto-hunt
- `ak` sends kill command
- 10+ unit tests passing

### Milestone 2: Campaign Navigation (Phase 2)

**Goal:** Campaign and quick where fully functional
**Modules:** campaign.lua, gmcp.lua, quick_where.lua, mapper.lua, aliases.lua
**Estimated effort:** ~300 new LOC
**Definition of done:**
- `xcp` lists campaign targets from GMCP
- `xcp 1` navigates to target (xrunto + quick where)
- `qw <mob>` shows results with room IDs
- `xm` / `xmall` / `go` / `nx` room search works
- 10+ unit tests passing

### Milestone 3: GUI Display (Phase 3)

**Goal:** Visual campaign/hunt status
**Modules:** gui.lua, campaign_gui.lua, buttons.lua
**Estimated effort:** ~190 new LOC
**Definition of done:**
- Campaign targets display in Geyser window
- Color-coded status (alive/dead/navigating)
- Clickable targets
- Window resize persistence

### Milestone 4: Advanced Features (Phase 4)

**Goal:** Mob learning, GQ, settings
**Modules:** mob_keywords.lua, global_quest.lua, settings.lua, gmcp.lua
**Estimated effort:** ~410 new LOC
**Definition of done:**
- Mob keywords learned from combat
- GQ tracking works
- All settings via `xset`

### Milestone 5: Polish (Phase 5)

**Goal:** Edge cases and nice-to-haves
**Modules:** Various additions across existing files
**Estimated effort:** ~200 new LOC
**Definition of done:**
- Room notes, vidblain handling
- Sound alerts
- All documented edge cases handled

### Total Estimated New Code: ~1,430 LOC

Combined with existing 906 LOC, the complete port would be ~2,300 LOC — roughly equivalent to WinkleWinkle's total (3,500) accounting for Mudlet's more concise API patterns vs MUSHclient's verbose XML format.

---

## 6. Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| Hunt response regex doesn't match actual Aardwolf output | Breaks core feature | Collect actual hunt output samples from live play before implementation |
| Mudlet mapper API differences between versions | Room search broken | Test against Mudlet 4.17+ (current stable), document minimum version |
| GMCP data format differs from expected | Campaign parsing fails | Capture actual GMCP data via `echo(json.encode(gmcp.comm.quest))` before implementation |
| Mock layer diverges from actual Mudlet behavior | Tests pass, live fails | Maintain mock parity; compare mock signatures against Mudlet API docs |
| Trigger lifecycle issues (one-shot cleanup) | Memory leak or missed triggers | Use tempRegexTrigger return value for cleanup; add timeout-based fallback |
| Multiple simultaneous hunts/campaigns | State corruption | Enforce single-operation state; queue or reject concurrent requests |

---

## 7. Build & Test Setup

### 7.1 Current State

The project has no working build/test system. The Nx/pnpm scaffolding is vestigial.

### 7.2 Recommended Setup

**Option A (Minimal):** Lua-only test runner
```bash
# Install Lua 5.1 (matches Mudlet runtime)
brew install lua@5.1

# Run tests
lua5.1 src/tests/test_runner.lua
```

**Option B (Keep Bun/TS):** Add Bun script that shells out to Lua
```json
// package.json
{
  "scripts": {
    "test": "lua5.1 src/tests/test_runner.lua",
    "build": "echo 'No build step - pure Lua project'"
  }
}
```

**Recommended:** Option B — maintains compatibility with harness expectations (`bun test` / `bun run build`) while using Lua for actual testing.

---

## 8. File Deliverables Summary

| Phase | New Files | Modified Files |
|-------|-----------|----------------|
| Phase 1 | `tests/test_hunt.lua`, `tests/test_runner.lua` | `hunt.lua`, `aliases.lua`, `gmcp.lua` |
| Phase 2 | `tests/test_campaign.lua`, `tests/test_qw.lua`, `tests/test_mapper.lua` | `campaign.lua`, `quick_where.lua`, `mapper.lua`, `gmcp.lua`, `aliases.lua` |
| Phase 3 | — | `gui.lua`, `campaign_gui.lua`, `buttons.lua` |
| Phase 4 | `mob_keywords.lua`, `global_quest.lua`, `settings.lua`, `tests/test_*.lua` | `gmcp.lua`, `database.lua`, `aliases.lua` |
| Phase 5 | — | Various existing files |

---

## 9. Definition of Done (Overall)

The SnD Mudlet port is **feature-complete** when:

1. All Phase 1-3 acceptance criteria met (daily-use features working)
2. Test suite with 30+ unit tests passing
3. Manual validation against live Aardwolf (hunt, campaign, quick where, xrunto, room search)
4. Mudlet package XML generated and installable
5. README with installation instructions and command reference
6. At minimum WinkleWinkle feature parity achieved

Phase 4-5 are enhancements that improve the tool but are not required for a usable release.
