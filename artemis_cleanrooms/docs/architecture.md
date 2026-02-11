# Artemis Bridge Simulator - Architecture Document

**Version:** 0.1.0
**Date:** 2026-02-10
**Location:** `crew/ericfriday/game/`
**Runtime:** Bun 1.3.8
**Language:** TypeScript (ES2022, strict mode)

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Map](#2-component-map)
3. [Protocol Specification](#3-protocol-specification)
4. [Game Simulation Loop](#4-game-simulation-loop)
5. [Client-Server Communication Patterns](#5-client-server-communication-patterns)
6. [Gap Analysis](#6-gap-analysis)
7. [Technology Assessment](#7-technology-assessment)

---

## 1. Architecture Overview

The Artemis Bridge Simulator is a multiplayer starship bridge simulator where players take on different crew roles (Helm, Weapons, Engineering, Science, Communications) aboard a shared ship. The architecture follows a **server-authoritative model** with a centralized game server maintaining all world state and broadcasting updates to connected clients.

### High-Level Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Game Server                             │
│  ┌──────────────┐  ┌──────────────┐  ┌────────────────────┐   │
│  │  TCP Listener │  │  WS Listener │  │  Game Tick Loop    │   │
│  │  port 2010   │  │  port 8080   │  │  20 Hz (50ms)      │   │
│  └──────┬───────┘  └──────┬───────┘  └────────┬───────────┘   │
│         │                 │                    │               │
│         ▼                 ▼                    ▼               │
│  ┌──────────────────────────────────────────────────┐         │
│  │              GameServer (2133 lines)              │         │
│  │  - Client management (TCP + WS)                   │         │
│  │  - Command routing & validation                   │         │
│  │  - World state broadcasting                       │         │
│  │  - Heartbeat & timeout detection                  │         │
│  │  - Inline simulation logic                        │         │
│  └──────────────────────────────────────────────────┘         │
│                                                                │
│  ┌──────────────────────────────────────────────────┐         │
│  │           GameSimulation (1480 lines)             │         │
│  │  - Standalone simulation engine                   │         │
│  │  - Tick-based physics & combat                    │         │
│  │  - NPC AI (enemy pursue, neutral flee/wander)     │         │
│  │  - Engineering (heat, coolant, damage)            │         │
│  │  - Win/loss conditions                            │         │
│  │  - Change tracking (ChangedEntities)              │         │
│  └──────────────────────────────────────────────────┘         │
│                                                                │
│  ┌──────────────────────────────────────────────────┐         │
│  │              Shared Layer                         │         │
│  │  constants.ts (189 lines)                         │         │
│  │  protocol.ts  (392 lines)                         │         │
│  │  types.ts     (223 lines)                         │         │
│  └──────────────────────────────────────────────────┘         │
└─────────────────────────────────────────────────────────────────┘
         │ TCP Binary              │ WS JSON
         ▼                        ▼
┌──────────────┐     ┌───────────────────────┐
│ Native Artemis│     │   Web Client           │
│ Client (2010) │     │   (HTTP :3000)         │
│               │     │  ┌─────────────────┐  │
│ Binary packet │     │  │ index.html      │  │
│ protocol      │     │  │ (lobby)         │  │
│               │     │  ├─────────────────┤  │
│               │     │  │ bridge.html     │  │
│               │     │  │ (all consoles)  │  │
│               │     │  │ 1901 lines      │  │
│               │     │  └─────────────────┘  │
└──────────────┘     └───────────────────────┘
```

### Key Architectural Decisions

1. **Dual-protocol server** - TCP on port 2010 for native Artemis protocol compatibility, WebSocket on port 8080 for web clients using JSON.
2. **Server-authoritative** - All game state lives on the server. Clients send commands, server validates and broadcasts results.
3. **Two simulation implementations** - `GameServer.ts` contains inline simulation logic (the running version), while `GameSimulation.ts` is a standalone, testable simulation engine with richer features. These are **not currently integrated** - see [Gap Analysis](#6-gap-analysis).
4. **Single-file web client** - Both lobby (`index.html`, 614 lines) and bridge (`bridge.html`, 1901 lines) are self-contained HTML files with inline CSS and JavaScript. No build step, no framework.
5. **Zero runtime dependencies** - Only dev dependencies (`@types/bun`, `bun-types`). All functionality is hand-rolled.

---

## 2. Component Map

### Directory Structure

```
crew/ericfriday/game/
├── package.json           # Zero runtime deps, Bun scripts
├── tsconfig.json          # ES2022, strict, bundler resolution
├── bun.lock
├── dist/                  # Build output
│   ├── server/index.js    # 66KB bundled server
│   └── client/serve.js    # 1.55KB bundled client server
└── src/
    ├── shared/
    │   ├── constants.ts   # Protocol constants, enums, world config
    │   ├── protocol.ts    # Binary protocol: buffer, packets, parser
    │   ├── types.ts       # Entity types, factory functions
    │   └── __tests__/
    │       └── protocol.test.ts  # 37 tests
    ├── server/
    │   ├── GameServer.ts  # Main game server (TCP + WS + sim)
    │   └── index.ts       # Entry point with graceful shutdown
    ├── simulation/
    │   ├── GameSimulation.ts  # Standalone simulation engine
    │   └── __tests__/
    │       └── GameSimulation.test.ts  # 68 tests
    ├── client/
    │   ├── serve.ts       # Static file HTTP server (:3000)
    │   ├── index.html     # Lobby UI
    │   └── bridge.html    # Bridge console UI (all 6 stations)
    └── __tests__/
        └── integration.test.ts  # 6 tests (WS connection flow)
```

### Component Responsibilities

| Component | File | Lines | Responsibility |
|-----------|------|-------|----------------|
| **Constants** | `shared/constants.ts` | 189 | Magic bytes, packet types (JamCRC), object types, console types, ship systems, ordnance types, beam frequencies, main screen views, world dimensions, factions |
| **Protocol** | `shared/protocol.ts` | 392 | `PacketBuffer` (read/write LE primitives + UTF-16LE strings), packet header creation (24-byte header with magic `0xDEADBEEF`), object update builder (bitfield-based), `PacketStreamParser` (streaming TCP parser with magic-byte sync), client command parser |
| **Types** | `shared/types.ts` | 223 | `Vec3`, `PlayerShip` (46 fields), `NPCShip` (14 fields), `Base`, `Mine`, `Nebula`, `Torpedo`, `Anomaly`, `Creature`, `GameWorld` (8 entity maps), `ClientInfo`, factory functions for default entities |
| **GameServer** | `server/GameServer.ts` | 2133 | TCP/WS server lifecycle, client connect/disconnect, console occupation tracking, command routing (24 client subtypes for TCP, 23 commands for WS), inline simulation tick at 20Hz, world state broadcasting (TCP binary + WS JSON), heartbeat system |
| **Server Entry** | `server/index.ts` | 46 | Server instantiation, banner, graceful shutdown handlers (SIGINT/SIGTERM/uncaughtException) |
| **GameSimulation** | `simulation/GameSimulation.ts` | 1480 | Standalone tick engine with 14 update phases, scenario generation, player command API, visibility queries (sensor range), change tracking (`ChangedEntities`), richer combat model (nuke AOE, EMP disable, mines, frequency matching, surrender) |
| **Client Server** | `client/serve.ts` | 73 | Bun HTTP server on :3000, static file serving with MIME types, route `/` to `index.html`, `/bridge` to `bridge.html`, directory traversal protection |
| **Lobby UI** | `client/index.html` | 614 | Ship selection (8 ships), console selection (6 types with SVG icons), player name input, WebSocket status connection, localStorage persistence, redirect to `/bridge` with query params |
| **Bridge UI** | `client/bridge.html` | 1901 | Full bridge console for all 6 stations (Helm, Weapons, Engineering, Science, Comms, Main Screen), canvas-based compass and tactical map, WebSocket command/update loop, red alert animation, real-time status displays |

### Test Coverage

| Test File | Tests | Coverage |
|-----------|-------|----------|
| `protocol.test.ts` | 37 | PacketBuffer read/write, header creation, all packet builders, stream parser, client command parsing |
| `GameSimulation.test.ts` | 68 | Movement, combat (beams, torpedoes, nukes, EMP, PShock, mines), engineering (heat, coolant, overheat damage), NPC AI (enemy pursue, neutral flee/wander), docking (energy/shield recharge, ordnance restock, repair), scanning, shields, win/loss conditions |
| `integration.test.ts` | 6 | WebSocket connection, welcome message, join/console selection, ready/game start, world updates, multi-client support |
| **Total** | **114** | All passing (239 expect() calls) |

---

## 3. Protocol Specification

### 3.1 Binary Packet Format (TCP, Port 2010)

All packets share a **24-byte header** followed by a variable-length payload:

```
Offset  Size  Type     Field
──────  ────  ─────    ─────
0       4     uint32   Magic (0xDEADBEEF)
4       4     uint32   Total packet length (header + payload)
8       4     uint32   Origin (0x01 = server, 0x02 = client)
12      4     uint32   Padding (0x00000000)
16      4     uint32   Remaining bytes (total - 20)
20      4     uint32   Packet type (JamCRC hash)
24      var   bytes    Payload
```

**Data types are little-endian.** Strings are UTF-16LE encoded with a 4-byte char count prefix (including null terminator).

### 3.2 Packet Types (Server → Client)

| Packet Type | Hash | Payload |
|-------------|------|---------|
| `PLAIN_TEXT_GREETING` | `0x6D04B3DA` | ASCII text welcome message |
| `VERSION` | `0xE548E74A` | 3x uint32 (major, minor, patch) |
| `SERVER_HEARTBEAT` | `0xF5821226` | Empty |
| `GAME_START` | `0x3DE66711` | 2x uint32 (game type, subtype) |
| `GAME_OVER` | `0x15171C8E` | Empty |
| `CONSOLE_STATUS` | `0x19C6E261` | uint32 shipIndex + 11x uint32 (per-console taken flags) |
| `OBJECT_UPDATE` | `0x80803DF9` | Sequence of object update blocks (see 3.4) |
| `DESTROY_OBJECT` | `0xCC5A3E30` | uint8 objectType + uint32 objectId |
| `GAME_MESSAGE` | `0xF754C8FE` | UTF-16LE string |
| `INCOMING_MESSAGE` | `0xD672C35F` | (defined but unused) |

### 3.3 Client Command Format (Client → Server)

All client commands use packet type `0x4C821D3C` with a **uint32 subtype** as the first 4 bytes of payload:

| Subtype | Name | Additional Payload |
|---------|------|--------------------|
| `0x00` | `HELM_SET_IMPULSE` | float32 (0.0-1.0) |
| `0x01` | `HELM_SET_WARP` | uint32 (0-4) |
| `0x02` | `HELM_SET_STEERING` | float32 (-1.0 to 1.0) |
| `0x03` | `WEAPONS_SET_TARGET` | uint32 targetId |
| `0x04` | `WEAPONS_TOGGLE_AUTO_BEAMS` | (none) |
| `0x05` | `WEAPONS_FIRE_TUBE` | uint32 tubeIndex |
| `0x06` | `WEAPONS_LOAD_TUBE` | uint32 tubeIndex + uint32 ordnanceType |
| `0x07` | `HELM_REQUEST_DOCK` | (none) |
| `0x08` | `FIRE_BEAMS` | (defined, unused) |
| `0x09` | `ENG_SET_ENERGY` | uint32 systemIndex + float32 value (0.0-3.0) |
| `0x0A` | `ENG_SET_COOLANT` | uint32 systemIndex + uint32 value |
| `0x0B` | `SCIENCE_SCAN` | uint32 targetId |
| `0x0C` | `SCIENCE_SELECT` | uint32 targetId |
| `0x0D` | `SET_SHIP` | uint32 shipIndex |
| `0x0E` | `SET_CONSOLE` | uint32 consoleType |
| `0x0F` | `READY` | (none) |
| `0x11` | `TOGGLE_SHIELDS` | (none) |
| `0x12` | `COMMS_SEND` | uint32 targetId |
| `0x13` | `SET_MAIN_SCREEN` | uint32 viewType |
| `0x14` | `SET_BEAM_FREQUENCY` | uint32 frequency (0-4) |
| `0x17` | `SET_SHIP_SETTINGS` | (defined, unused) |
| `0x19` | `HELM_TOGGLE_REVERSE` | (none) |
| `0x1D` | `CLIMB_DIVE` | float32 value |
| `0x24` | `HEARTBEAT` | (none) |

### 3.4 Object Update Bitfield Format

Object update packets contain a sequence of entity updates terminated by object type `0x00` (END):

```
For each entity:
  uint8    objectType
  uint32   objectId
  N bytes  bitfield (ceil(totalBits/8) bytes)
  var      property values (only for set bits, in bit order)

Terminator:
  uint8    0x00 (END)
```

The bitfield indicates which properties are present. Each entity type has its own property layout:

**PlayerShip (type 0x01) - 18 bits:**

| Bit | Type | Field |
|-----|------|-------|
| 0 | int32 | targetId |
| 1 | float | impulse |
| 2 | float | heading |
| 3 | float | velocity |
| 4 | float | position.x |
| 5 | float | position.y |
| 6 | float | position.z |
| 7 | float | shieldsFore |
| 8 | float | shieldsAft |
| 9 | uint8 | shieldsActive |
| 10 | float | energy |
| 11 | uint8 | warpFactor |
| 12 | uint8 | reverse |
| 13 | uint8 | docked |
| 14 | uint8 | redAlert |
| 15 | uint8 | mainScreenView |
| 16 | uint8 | autoBeams |
| 17 | uint8 | beamFrequency |

**NPCShip (type 0x05) - 12 bits:**

| Bit | Type | Field |
|-----|------|-------|
| 0 | string | name |
| 1 | float | position.x |
| 2 | float | position.y |
| 3 | float | position.z |
| 4 | float | heading |
| 5 | float | velocity |
| 6 | uint8 | faction |
| 7 | float | shieldsFore |
| 8 | float | shieldsAft |
| 9 | uint8 | surrendered |
| 10 | uint8 | inNebula |
| 11 | int32 | scanState |

**Base (type 0x06) - 6 bits:**

| Bit | Type | Field |
|-----|------|-------|
| 0 | string | name |
| 1 | float | position.x |
| 2 | float | position.y |
| 3 | float | position.z |
| 4 | float | shields |
| 5 | float | shieldsMax |

**Nebula (type 0x0A) - 4 bits / Torpedo (type 0x0B) - 5 bits:** (position + type/heading fields)

### 3.5 Stream Parser

The `PacketStreamParser` handles TCP stream reassembly:
1. Accumulates incoming data into an internal buffer
2. Scans for the magic byte `0xDEADBEEF` at position 0
3. If magic doesn't match, skips one byte and retries (byte-level sync recovery)
4. Reads total packet length from offset 4
5. If insufficient data, waits for more
6. Extracts complete packet and calls the callback
7. Advances buffer past the consumed packet

---

## 4. Game Simulation Loop

### 4.1 Tick Architecture

The game runs at **20 ticks/second** (50ms interval) using `setInterval`. Each tick:

```
GameServer.tick()
├── tickCount++
├── dt = 1/20 = 0.05 seconds
├── updatePlayerShips(dt)      // Movement, steering, energy drain
├── updateTubeLoading()        // Countdown tube load timers
├── updateAutoBeams()          // Auto-fire beams at targets in range
├── updateTorpedoes(dt)        // Move torpedoes, check collisions
├── updateEngineering(dt)      // Heat build/cool, overheat damage
├── updateNPCAI(dt)            // Enemy pursue/attack, neutral flee/wander
├── checkNebulaProximity()     // Set inNebula flags
├── checkDocking(dt)           // Energy recharge, ordnance restock, repair
├── checkGameOver()            // All stations destroyed or all enemies destroyed
└── if (tickCount % 2 == 0)    // Broadcast every 2nd tick (10 Hz)
    └── broadcastWorldState()  // TCP binary + WS JSON to all clients
```

**State broadcast rate:** 10 updates/second (every other tick). Static objects (nebulae) broadcast only every 5 seconds.

### 4.2 GameSimulation (Standalone Engine) Tick Order

The standalone `GameSimulation.ts` has a richer 14-phase tick:

```
GameSimulation.tick(dt)
├── 1.  updateEngineering(dt)      // Energy drain per system allocation, heat gen/dissip
├── 2.  updatePlayerMovement(dt)   // Impulse/warp speed, heading, position, bounds
├── 3.  updateNebulaEffects()      // inNebula flags for all ships
├── 4.  updateDocking(dt)          // Recharge, repair, restock
├── 5.  updateNPCAI(dt)            // Enemy approach, neutral wander/flee
├── 6.  updatePlayerBeams(dt)      // Auto-beam with frequency matching bonus
├── 7.  updateNPCAttacks(dt)       // NPC beam attacks on players/stations
├── 8.  updateTorpedoLoading(dt)   // Performance-scaled load speed
├── 9.  updateTorpedoFlight(dt)    // Homing tracking, collision, lifetime
├── 10. updateMines(dt)            // Proximity detonation
├── 11. updateEMPDisables(dt)      // EMP timer countdown
├── 12. updateScans(dt)            // Scan progress tracking (5s duration)
├── 13. updateShieldDrain(dt)      // Shield energy cost (3x in nebula)
└── 14. checkWinLoss()             // Victory/defeat conditions
```

### 4.3 Entity Lifecycle

**Creation:**
- Player ships: created on-demand when a client selects a ship index (`ensurePlayerShipExists`)
- NPCs, bases, nebulae: spawned during `spawnDefaultScenario()` at server start
- Torpedoes: spawned when a player fires a loaded tube
- Mines: spawned when a mine-type torpedo is fired (GameSimulation only)

**Destruction:**
- NPCs: destroyed when total system damage >= 6.0 and both shields <= 0 (GameServer) or when both shields <= 0 (GameSimulation)
- Bases: destroyed when shields reach 0 from NPC attacks
- Torpedoes: destroyed on impact, out-of-bounds, or lifetime expiry
- Mines: destroyed on detonation (GameSimulation only)
- Player ships: never explicitly destroyed (game ends on defeat condition)

**ID Allocation:** Sequential counter starting at 1000, incremented per entity.

### 4.4 Movement Model

**Player Ship Movement (GameServer):**
- **Steering:** Rudder value (-1 to +1) stored as `_rudder`. Turn rate = 1.5 * rudder * maneuverMod * maneuverEnergy * dt
- **Impulse:** 0-100% → 0-100 units/sec, modified by system damage/energy
- **Warp:** Factor 0-4 → 0-2000 units/sec, modified by system damage/energy. Drains extra energy.
- **Reverse:** 50% of forward speed
- **Acceleration:** 50 units/s^2 toward target speed
- **Vertical:** Pitch * 20 * dt units/sec
- **Bounds:** Clamped to 0-100000 (X/Z), -100000 to 100000 (Y)

**NPC Movement:**
- Enemies: 30 units/sec approach (9 units/sec when in attack range)
- Neutrals: 20 units/sec wander, flee from enemies within 5000 units

### 4.5 Combat Model

**Beam Weapons (Player):**
- Range: 1000 units
- Base damage: 3 per shot (GameServer) / 5 per shot (GameSimulation)
- Cooldown: 0.5s (GameServer) / 3s (GameSimulation)
- Auto-fire when target is in range and auto-beams enabled
- Damage modified by beam system energy and damage level
- Shield hit direction based on angle relative to target's heading
- Frequency matching: 1.5x damage bonus when beam frequency matches enemy shield frequency (GameSimulation only)

**Torpedo System:**
- 6 tubes per ship
- 8 ordnance types: Homing (20-30 dmg), Nuke (80-200 dmg, AOE 500 radius), Mine (50 dmg proximity), EMP (15s system disable), PShock (30-100 dmg), Beacon, Probe, Tag
- Load time: 5s (GameServer) / 10s (GameSimulation, performance-scaled)
- Speed: 400 units/sec (GameServer) / 300 units/sec (GameSimulation)
- Homing torpedoes track nearest enemy or designated target
- Hit radius: 100 units (GameServer) / 50 units (GameSimulation)

**NPC Combat:**
- Attack range: 1200 units (GameServer) / 1500 units (GameSimulation)
- Beam damage: 2 per shot (GameServer) / 5 per shot (GameSimulation)
- Cooldown: 1s (GameServer) / 3s (GameSimulation)
- Enemies attack player ships and friendly stations
- NPCs can be EMP-disabled (GameSimulation only)
- Surrender mechanic: 0.1% chance per tick when hull < 30% (GameSimulation only)

### 4.6 Engineering Model

**Energy System:**
- Max energy: 1000
- 8 ship systems, each with energy allocation (0-300%), heat, coolant, and damage values
- Energy drain: base 0.02/tick + warp drain + shield drain
- At 0 energy, all systems shut down (GameSimulation)

**Heat/Coolant:**
- Heat builds when energy allocation > 100%
- Coolant units (pool of 8 total) reduce heat per system
- Overheat threshold: 80% (GameServer) / 100% (GameSimulation)
- Overheat causes progressive system damage
- Critical overheat (100%): system shuts down and resets to 80% heat (GameServer)

**Docking:**
- Range: 700 units (GameServer) / 800 units (GameSimulation)
- Speed limit: 5 units/sec (GameServer) / 10% impulse (GameSimulation)
- While docked: energy recharge, shield repair, system damage repair, ordnance restock
- Restock interval: 60 ticks (GameServer) / 10 seconds per unit (GameSimulation)
- Raising shields forces undock (GameServer)

### 4.7 Win/Loss Conditions

- **Victory:** All enemy NPCs destroyed (GameServer: after 5s grace period; GameSimulation: also counts surrendered enemies)
- **Defeat:** All stations destroyed (both), or all player ships destroyed with shields/energy/systems at 0 (GameSimulation only)

---

## 5. Client-Server Communication Patterns

### 5.1 Connection Flow

```
Client                          Server
  │                                │
  ├──── WebSocket connect ────────>│
  │                                │ Store client, assign ID
  │<──── welcome (version) ────────┤
  │                                │
  ├──── join (ship, console) ─────>│
  │                                │ Assign console, update occupation
  │<──── consoleStatus ────────────┤
  │                                │
  ├──── ready ────────────────────>│
  │                                │ Set client ready
  │<──── gameStart ────────────────┤
  │<──── worldUpdate (full) ───────┤
  │                                │
  │    ┌──── Game Loop ────────┐   │
  │    │                       │   │
  ├────┤ command (setImpulse)  ├──>│ Process command
  │    │                       │   │
  │<───┤ worldUpdate (10Hz)    ├───┤ Broadcast state
  │<───┤ shipUpdate (per-ship) ├───┤
  │    │                       │   │
  │<───┤ heartbeat (3s)        ├───┤ Heartbeat check
  ├────┤ heartbeat             ├──>│ Update timestamp
  │    │                       │   │
  │<───┤ gameMessage           ├───┤ In-game notifications
  │<───┤ destroyObject         ├───┤ Entity destruction
  │    │                       │   │
  │    └───────────────────────┘   │
  │                                │
  │<──── gameOver ─────────────────┤ (victory/defeat)
  │                                │
  ├──── disconnect ───────────────>│
  │                                │ Release console, broadcast
```

### 5.2 WebSocket JSON Message Format

**Client → Server:**

```typescript
// Join a ship/console
{ type: "join", shipIndex: 0, consoleType: 1, playerName: "Kirk" }

// Signal ready
{ type: "ready" }

// Heartbeat
{ type: "heartbeat" }

// Game command
{ type: "command", command: "setImpulse", params: { value: 0.5 } }
{ type: "command", command: "setWarp", params: { value: 3 } }
{ type: "command", command: "setSteering", params: { value: -0.5 } }
{ type: "command", command: "setTarget", params: { targetId: 1003 } }
{ type: "command", command: "fireTube", params: { tubeIndex: 0 } }
{ type: "command", command: "loadTube", params: { tubeIndex: 0, ordnanceType: 0 } }
{ type: "command", command: "toggleShields" }
{ type: "command", command: "toggleAutoBeams" }
{ type: "command", command: "requestDock" }
{ type: "command", command: "toggleReverse" }
{ type: "command", command: "setEnergy", params: { system: 0, value: 1.5 } }
{ type: "command", command: "setCoolant", params: { system: 0, value: 2 } }
{ type: "command", command: "setBeamFrequency", params: { frequency: 2 } }
{ type: "command", command: "scanTarget", params: { targetId: 1005 } }
{ type: "command", command: "selectTarget", params: { targetId: 1005 } }
{ type: "command", command: "setMainScreen", params: { view: 4 } }
{ type: "command", command: "sendComms", params: { targetId: 1008 } }
{ type: "command", command: "setRedAlert", params: { active: true } }
```

**Server → Client:**

```typescript
// Welcome
{ type: "welcome", version: { major: 2, minor: 7, patch: 0 } }

// Game lifecycle
{ type: "gameStart" }
{ type: "gameOver" }

// Heartbeat
{ type: "heartbeat" }

// Console status
{ type: "consoleStatus", shipIndex: 0, consoles: [false, true, false, ...] }

// Full world state (broadcast 10Hz)
{
  type: "worldUpdate",
  world: {
    playerShips: [{ id, shipIndex, name, position, heading, velocity, ... }],
    npcShips: [{ id, name, position, heading, faction, shields, ... }],
    bases: [{ id, name, position, shields, shieldsMax }],
    mines: [{ id, position, ownerId }],
    nebulae: [{ id, position, nebulaType }],
    torpedoes: [{ id, position, heading, ordnanceType }]
  }
}

// Per-client ship detail (sent alongside worldUpdate)
{ type: "shipUpdate", ship: { ...fullPlayerShipObject } }

// In-game messages
{ type: "gameMessage", message: "Artemis has docked with DS1 Deep Space" }

// Object destruction
{ type: "destroyObject", objectType: 5, objectId: 1003 }
```

### 5.3 Heartbeat Protocol

- Server sends heartbeat every **3 seconds**
- Client timeout: **10 seconds** without any message
- On timeout: server disconnects client and releases console
- TCP clients send heartbeat via packet subtype `0x24`
- WS clients update timestamp on any message (no explicit heartbeat required)

### 5.4 Broadcasting Strategy

- **Full world state** every 2 ticks (10 Hz) to all clients
- **TCP:** Separate binary object update packets per entity type. NPCs batched in groups of 10.
- **WS:** Single JSON `worldUpdate` message with all entities + per-client `shipUpdate` with their ship detail.
- **Static optimization:** Nebulae only broadcast every 5 seconds (100 ticks) via TCP.
- **No delta compression:** Every broadcast sends full property values, not deltas.

---

## 6. Gap Analysis

### 6.1 Critical Gaps (vs. Fully-Featured Bridge Sim)

| Gap | Severity | Description |
|-----|----------|-------------|
| **Dual simulation codebases** | High | `GameServer.ts` has inline simulation logic while `GameSimulation.ts` is a richer standalone engine. They duplicate functionality with different constants and behaviors. The server should delegate to `GameSimulation` for tick updates. |
| **No persistent game sessions** | High | No save/load, no session management. Server restart = full state loss. |
| **No lobby management** | High | No mechanism to create games, set difficulty, choose scenarios, or wait for all players before starting. Game starts immediately on first `ready`. |
| **Single scenario only** | High | Only one hardcoded scenario (4 stations, 6 enemies, 2 neutrals, 3 nebulae). No mission scripting or scenario editor. |
| **No player ship destruction** | High | Player ships can take infinite damage. GameSimulation tracks defeat condition but it's not wired through to the server. |
| **No authentication/authorization** | Medium | Any client can connect and claim any console. No passwords, no admin controls. |
| **Simplified comms** | Medium | Communications are one-way canned responses. No inter-ship messaging, no surrender acceptance, no station orders. |
| **No Game Master mode** | Medium | `GAME_MASTER` console type is defined but not implemented. No ability to spawn entities, adjust difficulty, or control scenarios in real-time. |
| **No Fighter console** | Medium | `FIGHTER` console type defined but not implemented. No fighter bay, launch, or piloting. |
| **No Data/Observer consoles** | Low | Defined but unimplemented. |
| **No audio** | Low | No sound effects, no ship computer voice, no ambient audio. |
| **No 3D visualization** | Low | Client uses 2D canvas tactical maps only. No forward viewscreen rendering. |

### 6.2 Implementation Gaps

| Gap | Description |
|-----|-------------|
| **GameSimulation not integrated** | The tested standalone engine is not used by GameServer. Server has its own inline sim with different balance/features. |
| **No delta updates** | Full world state broadcast every 100ms. No tracking of what changed between ticks for TCP. (GameSimulation has `ChangedEntities` but it's unused by the server.) |
| **No visibility filtering** | All entities broadcast to all clients regardless of sensor range or nebula occlusion. `GameSimulation.getVisibleObjects()` exists but unused. |
| **Rudder stored as `_rudder`** | Private ship property stored via `any` cast. Should be a proper field or separate state. |
| **No connection resilience** | No reconnection logic. If a client disconnects, they lose their console assignment with no way to reclaim it. |
| **No rate limiting** | Clients can send commands as fast as they want. No throttling or validation of command frequency. |
| **Bridge HTML is monolithic** | 1901-line single file with all 6 consoles. Should be componentized for maintainability. |
| **No unload tube** | Tubes can be loaded but never unloaded. Misloaded ordnance is wasted. |
| **Anomalies/Creatures unused** | Types defined but no spawning or interaction logic. |

### 6.3 Feature Parity with Artemis SBS

Compared to the original Artemis Spaceship Bridge Simulator:

| Feature | Status | Notes |
|---------|--------|-------|
| Multi-console bridge play | Partial | 6 of 11 consoles have UI |
| Ship movement (impulse/warp) | Complete | |
| Beam weapons | Complete | Auto-beams functional |
| Torpedo system | Complete | All 8 ordnance types defined, 4 functional |
| Engineering power/coolant | Complete | Heat, overheat, damage model |
| Science scanning | Partial | Scan level 1/2 via bitmask, no detailed readout |
| Communications | Minimal | Canned responses only |
| Docking | Complete | Repair, restock, recharge |
| NPC AI | Basic | Pursue/attack or flee/wander. No fleet tactics. |
| Multiple ship types | Missing | All ships identical |
| Ship customization | Missing | No hull/system variation |
| Multiple scenarios | Missing | Single hardcoded scenario |
| Difficulty levels | Missing | |
| Sector/map generation | Missing | Fixed positions |
| Jump drive | Missing | Type defined but not implemented |
| Tractor beams | Missing | |
| Black holes | Missing | |
| Asteroids | Missing | |
| Fleet formations | Missing | |
| Deep Strike mode | Missing | |
| PvP mode | Missing | |
| Game Master features | Missing | Console defined, no implementation |

---

## 7. Technology Assessment

### 7.1 Bun Runtime

**Advantages:**
- **Built-in TCP and WebSocket servers** via `Bun.listen()` and `Bun.serve()`. No need for `ws`, `socket.io`, or `net` packages.
- **Built-in test runner** (`bun test`). No Jest/Vitest dependency.
- **Fast startup** (< 100ms for the game server).
- **Native TypeScript execution** - no build step for development.
- **Fast bundling** - `bun build` produces optimized bundles in milliseconds (5 modules → 66KB in 6ms).
- **Zero runtime dependencies** achieved thanks to Bun's built-in APIs.

**Performance Characteristics:**
- Server bundle: 66KB (5 modules) - very small, fast loading
- Client server: 1.55KB (1 module)
- 114 tests run in ~6 seconds (includes 6 integration tests that start real servers)
- Single-threaded event loop handles game tick + client I/O efficiently at 20 Hz

**Concerns:**
- **Single-threaded** - Simulation and I/O share one thread. At scale (many clients, complex simulation), the 50ms tick budget may be exceeded. No worker thread delegation.
- **Platform availability** - Bun is less mature than Node.js. May not be available on all deployment targets.
- **Buffer API** - Uses Node.js `Buffer` compatibility layer. Some edge cases may differ from Node.js behavior.

### 7.2 WebSocket Performance

**Current Implementation:**
- Full JSON serialization of world state 10 times/second to each client
- World state includes all entity arrays regardless of relevance
- Per-client `shipUpdate` sent alongside `worldUpdate` (redundant data)
- No compression, no message batching, no binary WS frames

**Estimated Bandwidth (per client, per second):**
- World update: ~2-3 KB per message × 10 messages/sec = ~20-30 KB/sec
- Ship update: ~500 bytes × 10 messages/sec = ~5 KB/sec
- Total: **~25-35 KB/sec per client outbound**

**Scaling Projections:**
- 8 clients (1 ship): ~200-280 KB/sec total outbound
- 32 clients (4 ships): ~800 KB-1.1 MB/sec total outbound
- This is manageable on LAN but could stress WAN connections

**Optimization Opportunities:**
1. Delta-only updates (track `ChangedEntities` from GameSimulation)
2. Visibility filtering (use `getVisibleObjects()`)
3. Binary WebSocket frames (MessagePack or custom binary)
4. Per-entity update rates (static objects less frequently)
5. Client-side interpolation with less frequent server updates

### 7.3 Build & Deployment

```bash
# Development (hot reload)
bun run --watch src/server/index.ts    # Server with file watching
bun run --watch src/client/serve.ts    # Client HTTP server with watching

# Production build
bun build src/server/index.ts --outdir dist/server --target bun  # 66KB
bun build src/client/serve.ts --outdir dist/client --target bun  # 1.55KB

# Test
bun test                    # All 114 tests
bun test src/shared/        # Protocol tests only (37)
bun test src/simulation/    # Simulation tests only (68)
bun test src/__tests__/     # Integration tests only (6)

# Run production
bun run dist/server/index.js    # Game server
bun run dist/client/serve.js    # Client HTTP
```

**Deployment requirements:** Bun runtime only. No other system dependencies. Server runs on ports 2010 (TCP), 8080 (WS), 3000 (HTTP).

### 7.4 Code Quality Assessment

| Metric | Assessment |
|--------|------------|
| **Type safety** | Strong. Strict TypeScript throughout. Only `_rudder` uses `any` cast. |
| **Test coverage** | Good. 114 tests cover protocol, simulation, and integration. GameServer (the actual running server) has no direct unit tests. |
| **Separation of concerns** | Mixed. Shared layer is well-separated. Server mixes networking with simulation. Standalone `GameSimulation` exists but isn't used. |
| **Error handling** | Minimal. Socket/WS errors caught and logged. No client-facing error messages. No retry logic. |
| **Code organization** | Functional. Clear section headers and grouping within files. Long files (GameServer: 2133 lines) could benefit from decomposition. |
| **Constants management** | Good. All game constants centralized with descriptive names. Two sets of constants (server and simulation) with different values is a concern. |
| **Memory management** | Acceptable. Entity Maps cleaned up on destruction. Torpedo flight data cleaned up. No obvious memory leaks. |

---

## Appendix A: Entity Type Reference

| Object Type | Value | Entity | Fields |
|-------------|-------|--------|--------|
| `END` | 0x00 | Terminator | - |
| `PLAYER_SHIP` | 0x01 | Player ship | 46 fields (position, heading, velocity, energy, shields, systems, tubes, ordnance, etc.) |
| `WEAPONS_CONSOLE` | 0x02 | Weapons data | (unused) |
| `ENGINEERING_CONSOLE` | 0x03 | Engineering data | (unused) |
| `PLAYER_UPGRADES` | 0x04 | Upgrade state | (unused) |
| `NPC_SHIP` | 0x05 | NPC vessel | 14 fields (position, heading, faction, shields, scanState, etc.) |
| `BASE` | 0x06 | Friendly station | 6 fields (position, name, shields) |
| `MINE` | 0x07 | Deployed mine | 3 fields (position, ownerId) |
| `ANOMALY` | 0x08 | Space anomaly | 4 fields (position, type) |
| `NEBULA` | 0x0A | Nebula cloud | 4 fields (position, type) |
| `TORPEDO` | 0x0B | In-flight torpedo | 5 fields (position, heading, ordnanceType) |
| `CREATURE` | 0x0F | Space creature | 5 fields (position, heading, type, name) |
| `DRONE` | 0x10 | Drone | (unused) |

## Appendix B: Console Types

| Console | Value | Implemented | UI |
|---------|-------|-------------|-----|
| Main Screen | 0x00 | Yes | Tactical overview display |
| Helm | 0x01 | Yes | Compass, impulse/warp, dock/reverse |
| Weapons | 0x02 | Yes | Target selection, tubes, shields, beam frequency |
| Engineering | 0x03 | Yes | 8-system energy/coolant sliders, heat bars |
| Science | 0x04 | Yes | Target scanning, long-range map |
| Communications | 0x05 | Yes | Hail targets, message log |
| Fighter | 0x06 | No | - |
| Data | 0x07 | No | - |
| Observer | 0x08 | No | - |
| Captain's Map | 0x09 | No | - |
| Game Master | 0x0A | No | - |

## Appendix C: Simulation Constants Comparison

| Constant | GameServer | GameSimulation | Notes |
|----------|-----------|----------------|-------|
| Tick rate | 20 Hz | N/A (caller-driven dt) | |
| Impulse max speed | 100 u/s | 100 u/s | Match |
| Warp speed mult | 500 u/s/factor | 500 u/s/factor | Match |
| Beam range | 1000 u | 1000 u | Match |
| Beam damage | 3/shot | 5/shot | **Mismatch** |
| Beam cooldown | 0.5s (10 ticks) | 3s | **Mismatch** |
| Torpedo speed | 400 u/s | 300 u/s | **Mismatch** |
| Torpedo hit radius | 100 u | 50 u | **Mismatch** |
| Torpedo load time | 5s (100 ticks) | 10s | **Mismatch** |
| Homing damage | 20 | 30 | **Mismatch** |
| Nuke damage | 80 | 200 | **Mismatch** |
| NPC beam range | 1200 u | 1500 u | **Mismatch** |
| NPC beam damage | 2/shot | 5/shot | **Mismatch** |
| NPC beam cooldown | 1s | 3s | **Mismatch** |
| Dock range | 700 u | 800 u | **Mismatch** |
| Nebula radius | 3000 u | 2000 u | **Mismatch** |
| Energy max | 1000 | 1000 | Match |
| NPC destruction | 6.0 total sys dmg | shields <= 0 | **Different logic** |
| Beam freq matching | No | Yes (1.5x bonus) | **GameSim only** |
| Nuke AOE | No | Yes (500u radius) | **GameSim only** |
| EMP disable | No | Yes (15s) | **GameSim only** |
| Mine detonation | No | Yes (200u radius) | **GameSim only** |
| Surrender mechanic | No | Yes (0.1%/tick at <30%) | **GameSim only** |
| Scan duration | Instant | 5 seconds | **Different** |
| Sensor range/visibility | Not filtered | 50000u, performance-scaled | **GameSim only** |
