# Distributed Multiplayer Architecture Design

**Issue:** ac-38j
**Date:** 2026-02-10
**Status:** Draft
**Game:** Artemis Bridge Simulator (TypeScript/Bun)

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Current Architecture](#2-current-architecture)
3. [State Synchronization Strategy](#3-state-synchronization-strategy)
4. [Network Topology](#4-network-topology)
5. [Session Management](#5-session-management)
6. [Scalability Plan](#6-scalability-plan)
7. [Latency Compensation & Conflict Resolution](#7-latency-compensation--conflict-resolution)
8. [Data Serialization & Bandwidth Optimization](#8-data-serialization--bandwidth-optimization)
9. [Implementation Roadmap](#9-implementation-roadmap)
10. [Appendix: Protocol Reference](#appendix-protocol-reference)

---

## 1. Executive Summary

This document specifies the distributed multiplayer architecture for the Artemis Bridge Simulator. The game supports 2-8 player ships with 6 consoles each (up to 48 concurrent players) in a cooperative starship bridge simulation.

### Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Authority model | **Authoritative dedicated server** | Cooperative game needs consistent physics; no competitive cheating concerns but consistency is critical for shared bridge instruments |
| Network topology | **Dedicated server with WebSocket transport** | Bun's native WebSocket support, NAT-friendly, supports both browser and native clients |
| State sync | **Server-reconciled delta compression** | Leverages existing bit-field protocol; add client-side interpolation for visual smoothness |
| Session model | **Lobby → Game → Results lifecycle** | Clean separation of pre-game coordination from gameplay |
| Serialization | **Binary protocol (existing) + MessagePack for metadata** | Keep existing efficient binary wire format for game state; use MessagePack for lobby/session control |

### Design Principles

1. **Consistency over speed** - Bridge crews share instruments. All 6 consoles on a ship must see identical state. A helm officer and weapons officer disagreeing on ship position breaks gameplay.
2. **Graceful degradation** - Players can disconnect and reconnect. The ship continues with AI filling gaps.
3. **Bandwidth efficiency** - Delta-compressed state updates. Players only receive updates relevant to their ship and nearby objects.
4. **Horizontal scaling** - One game session per server process. Scale by running multiple processes, not by distributing a single game.

---

## 2. Current Architecture

### What Exists

```
┌─────────────────────────────────────────────────┐
│                  GameServer                      │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐ │
│  │ TCP:2010 │  │ WS:8080  │  │ GameSimulation│ │
│  │ (binary) │  │ (JSON)   │  │ (20Hz tick)   │ │
│  └────┬─────┘  └────┬─────┘  └───────┬───────┘ │
│       │              │                │          │
│  ┌────▼──────────────▼────┐  ┌───────▼───────┐ │
│  │   Client Tracking      │  │   GameWorld    │ │
│  │   consoleOccupation    │  │   (all state)  │ │
│  └────────────────────────┘  └───────────────┘ │
└─────────────────────────────────────────────────┘
```

- **Single process** handles everything
- **No lobby** - clients join a default game immediately
- **No authentication** - clients identified by random clientId
- **No reconnection** - disconnect = lose your seat
- **No interest management** - all clients receive all updates
- **10Hz full-world broadcasts** to every connected client

### What Needs to Change

| Gap | Impact | Priority |
|-----|--------|----------|
| No session management | Can't run multiple games | P0 |
| No player identity | Can't reconnect or track players | P0 |
| No interest management | O(n²) bandwidth with 48 players | P1 |
| No client prediction | Helm feels laggy on >50ms RTT | P1 |
| No reconnection | Any network blip = lost seat | P1 |
| Single transport (TCP+WS separate) | Two code paths, inconsistent behavior | P2 |

---

## 3. State Synchronization Strategy

### 3.1 Authority Model: Server-Authoritative with Client Interpolation

**Decision: Keep server as sole authority. Add client-side interpolation and optional helm prediction.**

```
┌──────────┐          ┌──────────────────┐          ┌──────────┐
│  Client  │  intent  │     Server       │  truth   │  Client  │
│  (Helm)  │────────→ │  (authoritative) │────────→ │  (Sci)   │
│          │  ←────── │  tick @ 20Hz     │  ←────── │          │
│ predict  │  correct │  broadcast @10Hz │  interp  │ display  │
└──────────┘          └──────────────────┘          └──────────┘
```

**Why server-authoritative:**
- Bridge crews need *identical* state. If helm and weapons see different positions, they can't coordinate fire.
- Game is cooperative, not competitive PvP. No incentive to cheat.
- Physics simulation is deterministic at 20Hz - single source of truth prevents drift.
- NPCs and AI must behave consistently for all observers.

**What changes from current:**
- Add client-side **interpolation buffer** (100ms, covers 1 missed server tick)
- Add optional **helm prediction** for own-ship movement only (impulse, steering)
- Server corrections override predictions when they arrive

### 3.2 State Partitioning

Not all state changes at the same rate or matters to all players equally.

| State Category | Update Rate | Recipients | Encoding |
|---------------|-------------|------------|----------|
| **Own ship core** (position, heading, velocity, shields) | 10 Hz | Ship's crew only | Binary delta |
| **Own ship systems** (energy, coolant, heat, damage) | 5 Hz | Ship's crew only | Binary delta |
| **Own ship tubes** (loading state, ordnance counts) | On change | Ship's crew only | Binary delta |
| **Nearby entities** (<10k units) | 10 Hz | Ships within range | Binary delta |
| **Distant entities** (>10k units) | 2 Hz | Ships within range | Binary delta, position-only |
| **Bases** | 2 Hz | All ships | Binary delta |
| **Torpedoes** (in flight) | 20 Hz | Ships within 5k units | Binary delta |
| **Game events** (explosions, messages) | On occurrence | Relevant ships | Event packet |
| **Session/lobby** | On change | All connected | MessagePack |

### 3.3 Interest Management

Each ship has a **relevance sphere**. Objects are categorized by distance to the ship.

```
                    ┌─────────────────────────────┐
                    │      Far Zone (>20k)        │
                    │   Position only, 1 Hz       │
                    │  ┌───────────────────────┐  │
                    │  │   Mid Zone (10k-20k)  │  │
                    │  │   Reduced, 2 Hz       │  │
                    │  │  ┌─────────────────┐  │  │
                    │  │  │ Near Zone (<10k) │  │  │
                    │  │  │ Full state, 10Hz │  │  │
                    │  │  │   ┌─────────┐   │  │  │
                    │  │  │   │Own Ship │   │  │  │
                    │  │  │   │ 10-20Hz │   │  │  │
                    │  │  │   └─────────┘   │  │  │
                    │  │  └─────────────────┘  │  │
                    │  └───────────────────────┘  │
                    └─────────────────────────────┘
```

**Implementation:** Server maintains per-ship relevance lists, updated every 500ms. Broadcast loop sends filtered updates per ship rather than global broadcasts.

### 3.4 State Snapshot Format

Each tick, the simulation produces a `WorldDelta`:

```typescript
interface WorldDelta {
  tick: number;              // Monotonic tick counter
  timestamp: number;         // Server wall clock (ms)
  entities: EntityDelta[];   // Changed entities with bit-field properties
  destroyed: DestroyedEntity[];
  events: GameEvent[];       // Explosions, messages, scan results
}

interface EntityDelta {
  type: ObjectType;
  id: number;
  properties: BitFieldEncoded;  // Existing protocol format
}
```

This extends the existing `OBJECT_UPDATE` packet format. The key addition is the `tick` counter for ordering and the `timestamp` for interpolation.

---

## 4. Network Topology

### 4.1 Decision: Dedicated Server, Single Process per Game

```
┌─────────────────────────────────────────────────────────┐
│                    Deployment Host                       │
│                                                         │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ Game Server 1│  │ Game Server 2│  │ Game Server 3│ │
│  │ Session: abc │  │ Session: def │  │ Session: ghi │ │
│  │ 3 ships/18p  │  │ 8 ships/48p  │  │ 2 ships/12p  │ │
│  │ WS :8081     │  │ WS :8082     │  │ WS :8083     │ │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘ │
│         │                  │                  │         │
│  ┌──────▼──────────────────▼──────────────────▼──────┐ │
│  │              Lobby / Gateway Server                │ │
│  │              WS :8080 + HTTP :3000                 │ │
│  │  - Player authentication                          │ │
│  │  - Session listing & creation                     │ │
│  │  - Routes players to game servers                 │ │
│  └───────────────────────┬───────────────────────────┘ │
└──────────────────────────┼─────────────────────────────┘
                           │
              ─ ─ ─ ─ ─ ─ ┼ ─ ─ ─ ─ ─ ─
                           │  Internet
              ┌────────────┼────────────┐
              │            │            │
         ┌────▼───┐  ┌────▼───┐  ┌────▼───┐
         │Client 1│  │Client 2│  │Client N│
         │(browser│  │(browser│  │(native)│
         │  or    │  │  or    │  │        │
         │ native)│  │ native)│  │        │
         └────────┘  └────────┘  └────────┘
```

**Why dedicated server (not P2P):**

| Factor | Dedicated Server | P2P / Hybrid |
|--------|-----------------|--------------|
| NAT traversal | Not needed (server has public IP) | Complex, unreliable |
| Authority | Natural single authority | Must elect authority, handle migration |
| Cheating | Server validates all input | Peers can lie about state |
| Complexity | Simpler architecture | Needs STUN/TURN, NAT hole-punching |
| Hosting cost | Requires server infrastructure | Players host, but unreliable |
| Latency for co-located crew | All go through server | Could be lower with direct P2P |
| Player count | Scales linearly | Scales quadratically (mesh) |

**Why single process per game (not distributed simulation):**
- Game world is modest: ~100-200 entities max (8 ships, 50 NPCs, stations, torpedoes)
- 20Hz tick with simple physics is trivial for a single core
- Distributing the simulation adds complexity for no measurable gain at this scale
- A single Bun process can handle 48 WebSocket connections with zero pressure

### 4.2 Transport: Unified WebSocket

**Decision: Migrate to WebSocket-only transport. Drop raw TCP.**

**Rationale:**
- Current TCP transport exists for legacy Artemis protocol compatibility
- New architecture is greenfield - no legacy clients to support
- WebSocket provides: reliable delivery, framing, browser support, proxy/CDN compatibility
- Bun's WebSocket implementation is extremely fast (near-raw-TCP performance)
- Simplifies the codebase: one transport, one code path
- Binary WebSocket frames carry the existing binary protocol without modification

**Migration path:**
1. WebSocket carries the same binary packet format (ArrayBuffer frames)
2. Remove TCP server code path
3. Keep `PacketBuffer` and binary protocol unchanged
4. Add WebSocket ping/pong for connection health (replaces custom heartbeat)

### 4.3 Connection Flow

```
Client                        Gateway                      Game Server
  │                              │                              │
  ├── WS connect ──────────────→ │                              │
  │                              │                              │
  ├── AUTH {name, token} ──────→ │                              │
  │ ←── AUTH_OK {playerId} ────  │                              │
  │                              │                              │
  ├── LIST_SESSIONS ───────────→ │                              │
  │ ←── SESSIONS [{id,name,..}]  │                              │
  │                              │                              │
  ├── JOIN_SESSION {sessionId} → │                              │
  │                              ├── RESERVE {playerId} ──────→ │
  │                              │ ←── RESERVED {wsUrl} ──────  │
  │ ←── REDIRECT {wsUrl} ─────  │                              │
  │                              │                              │
  ├── WS connect (to game) ────────────────────────────────────→│
  ├── REJOIN {playerId, token} ────────────────────────────────→│
  │ ←── WELCOME + VERSION ──────────────────────────────────────│
  │                              │                              │
  │  ... normal game protocol ...                               │
```

### 4.4 Gateway Server Responsibilities

The Gateway is lightweight - it handles coordination, not gameplay:

- **Player registry**: Name → playerId mapping, simple token auth
- **Session catalog**: List active game sessions with metadata (ship count, player count, scenario name, game state)
- **Session lifecycle**: Create/destroy game server processes
- **Routing**: Direct players to the correct game server WebSocket URL
- **Health monitoring**: Detect crashed game servers, notify connected players

The Gateway does NOT proxy game traffic. After the redirect, clients connect directly to the game server. This avoids adding latency to every game packet.

---

## 5. Session Management

### 5.1 Session Lifecycle

```
     CREATE              START              END
  ┌──────────┐       ┌──────────┐       ┌──────────┐
  │          │       │          │       │          │
  │  LOBBY   │──────→│ RUNNING  │──────→│ RESULTS  │──→ CLOSED
  │          │       │          │       │          │
  └──────────┘       └──────────┘       └──────────┘
       │                   │                  │
       │ join/leave        │ reconnect        │ view stats
       │ pick ship         │ disconnect       │
       │ pick console      │                  │
       │ set ready         │                  │
       │ chat              │                  │
```

### 5.2 Session States

**LOBBY:**
- Creator sets: scenario, difficulty, ship count limit, password (optional)
- Players connect, choose ship and console
- Ship captain (first player on a ship) can rename the ship
- Chat available for coordination
- Game starts when: creator clicks start AND at least one player per ship is ready
- Timeout: lobby auto-closes after 30 minutes of inactivity

**RUNNING:**
- Normal gameplay via existing simulation engine
- New players can join mid-game (hot-join) - they see available consoles
- Players can switch consoles on their ship (releases old, claims new)
- Players cannot switch ships during gameplay (prevents chaos)
- Disconnected players get a 60-second reconnection window before their console is released

**RESULTS:**
- Victory or defeat screen with stats (damage dealt, torpedoes fired, etc.)
- Players can choose: return to lobby (new game) or disconnect
- Results persist for 5 minutes, then session auto-closes

### 5.3 Session Data Model

```typescript
interface GameSession {
  id: string;               // UUID v4
  name: string;             // Display name
  password?: string;        // Optional, hashed
  creatorId: string;        // Player who created it
  state: "lobby" | "running" | "results";

  // Configuration (set in lobby)
  scenario: ScenarioConfig;
  maxShips: number;         // 1-8
  difficulty: number;       // 1-11

  // Runtime
  ships: ShipAssignment[];
  connectedPlayers: Map<string, PlayerConnection>;

  // Timestamps
  createdAt: number;
  startedAt?: number;
  endedAt?: number;
}

interface ShipAssignment {
  shipIndex: number;        // 0-7
  name: string;             // "Artemis", "Intrepid", etc.
  consoles: Map<ConsoleType, string | null>;  // console → playerId
}

interface PlayerConnection {
  playerId: string;
  displayName: string;
  shipIndex: number;
  consoleType: ConsoleType;
  connected: boolean;
  lastSeen: number;         // For reconnection window
  reconnectToken: string;   // Opaque token for reconnection
}
```

### 5.4 Join/Leave Protocol

**Joining a session:**
```
1. Client → Gateway: JOIN_SESSION { sessionId, password? }
2. Gateway validates: session exists, not full, password matches
3. Gateway → Game Server: PLAYER_JOINING { playerId, displayName }
4. Game Server reserves a slot, generates reconnect token
5. Gateway → Client: REDIRECT { wsUrl, reconnectToken }
6. Client connects directly to Game Server via WebSocket
7. Client → Game Server: REJOIN { playerId, reconnectToken }
8. Game Server → Client: SESSION_STATE { ships, consoles, players, gameState }
```

**Leaving (graceful):**
```
1. Client → Game Server: LEAVE
2. Game Server releases console, broadcasts updated roster
3. Client disconnects WebSocket
```

**Disconnecting (ungraceful):**
```
1. WebSocket closes unexpectedly
2. Game Server marks player as disconnected, starts 60s timer
3. If player reconnects within 60s → restore to same ship/console
4. If timer expires → release console, broadcast roster update
5. If no players left on a ship → ship continues under AI autopilot
```

### 5.5 Reconnection

Reconnection is critical for a game with 48 players - someone's WiFi will hiccup.

```
Client                                    Game Server
  │                                            │
  │  (connection lost)                         │
  │  ✕ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ✕ │
  │                                            │ mark disconnected
  │                                            │ start 60s timer
  │  ... network recovers ...                  │
  │                                            │
  ├── WS connect (to same game server) ──────→ │
  ├── REJOIN { playerId, reconnectToken } ───→ │
  │                                            │ validate token
  │                                            │ cancel timer
  │                                            │ restore assignment
  │ ←── SESSION_STATE (full snapshot) ────────  │
  │ ←── GAME_START (if running) ──────────────  │
  │                                            │
  │  ... resume normal gameplay ...            │
```

**Reconnect token:**
- Generated when player first joins
- 128-bit random, stored server-side mapped to playerId
- Valid for the lifetime of the session
- Invalidated when session closes or player is explicitly kicked

---

## 6. Scalability Plan

### 6.1 Scale Target

| Metric | Minimum | Maximum | Design Target |
|--------|---------|---------|---------------|
| Ships per game | 2 | 8 | 8 |
| Consoles per ship | 1 | 6 | 6 |
| Players per game | 2 | 48 | 48 |
| Concurrent games | 1 | ~50 | 10 |
| Total concurrent players | 2 | ~2400 | 480 |
| Entities per game | ~20 | ~200 | 150 |
| Tick rate | 20 Hz | 20 Hz | 20 Hz |
| State update rate | 10 Hz | 10 Hz | 10 Hz |

### 6.2 Single Game Server Performance Budget

At maximum load (8 ships, 48 players, 150 entities):

**CPU budget per tick (50ms budget at 20Hz):**
- Physics update: ~2ms (simple 2D movement, no complex collisions)
- NPC AI: ~3ms (50 NPCs with pathfinding)
- Combat resolution: ~1ms (beam/torpedo hit checks)
- State delta computation: ~1ms (bit-field diff)
- Interest management: ~0.5ms (distance checks per ship)
- Serialization + send: ~2ms (48 binary packets)
- **Total: ~10ms** → 80% headroom on 50ms budget

**Memory budget:**
- Game world state: ~500KB (150 entities × ~3KB average)
- Player connections: ~50KB (48 connections × ~1KB)
- Interest management: ~20KB (8 relevance lists)
- Buffers (send/receive): ~200KB
- **Total: ~1MB** → negligible

**Bandwidth budget (per second, outbound):**
- Per-ship update (own ship, 10Hz): ~200 bytes × 10 = 2 KB/s
- Near entities (20 entities, 10Hz): ~100 bytes × 20 × 10 = 20 KB/s
- Mid entities (30 entities, 2Hz): ~40 bytes × 30 × 2 = 2.4 KB/s
- Events (occasional): ~0.5 KB/s
- **Per player: ~25 KB/s outbound**
- **48 players: ~1.2 MB/s outbound** → well within server capabilities

**Inbound:**
- Per player input: ~50 bytes × 10 inputs/s = 0.5 KB/s
- **48 players: ~24 KB/s inbound** → trivial

### 6.3 Scaling Strategy

**Vertical (single host):**
- One game server process per game session
- Bun process handles 48 WS connections easily
- Single-threaded event loop is sufficient (no need for workers)
- A modest server (4 cores, 8GB RAM) can run 10+ concurrent games

**Horizontal (multiple hosts):**
- Gateway distributes sessions across hosts
- No shared state between game servers (each game is independent)
- Add hosts to support more concurrent games
- Load balancing at the Gateway level (round-robin session assignment)

```
┌─────────────────────────────────────┐
│            Gateway (1)              │
│  - Session catalog                  │
│  - Player routing                   │
│  - Health checks                    │
└──────┬──────────┬──────────┬────────┘
       │          │          │
┌──────▼───┐ ┌───▼──────┐ ┌▼─────────┐
│ Host A   │ │ Host B   │ │ Host C   │
│ 4 games  │ │ 4 games  │ │ 2 games  │
│ ~192 p   │ │ ~192 p   │ │ ~96 p    │
└──────────┘ └──────────┘ └──────────┘
```

### 6.4 Process Management

Game server processes are managed by the Gateway:

```typescript
interface GameProcess {
  sessionId: string;
  pid: number;
  port: number;           // Assigned from pool (8081-8180)
  host: string;           // Hostname/IP
  startedAt: number;
  playerCount: number;
  state: "starting" | "ready" | "running" | "stopping";
}
```

**Lifecycle:**
1. Player creates session → Gateway spawns `bun run src/server/index.ts --port=808X --session=<id>`
2. Game server opens WebSocket on assigned port, reports ready
3. Gateway routes players to `ws://host:808X`
4. Game ends → Game server sends results to Gateway, exits
5. Gateway cleans up session catalog entry

---

## 7. Latency Compensation & Conflict Resolution

### 7.1 Latency Profile

Bridge simulators have different latency requirements than FPS games:

| Action | Acceptable Latency | Rationale |
|--------|-------------------|-----------|
| Helm steering | 100-150ms | Ship turns slowly; small delay is masked by inertia |
| Impulse/warp changes | 200ms | Speed changes are gradual |
| Torpedo fire | 150ms | Player fires, expects visual confirmation within a few frames |
| Shield toggle | 100ms | Tactical decision, needs responsive feedback |
| Beam weapons (auto) | N/A | Server-controlled, no player input lag |
| Science scan | 500ms | Scan takes seconds anyway |
| Engineering energy | 200ms | Adjustments are incremental |
| Comms messages | 500ms | Text-based, inherently slow |

**Key insight:** Most actions are either gradual (helm, engineering) or have built-in delays (torpedo loading, scan time). This means we don't need aggressive client prediction for most consoles.

### 7.2 Compensation Strategies

**Strategy 1: Input Timestamping (All Consoles)**

Every client command includes a local timestamp. Server uses this for ordering when multiple commands arrive in the same tick.

```typescript
interface ClientCommand {
  type: CommandSubtype;
  timestamp: number;      // Client-side DOMHighResTimeStamp
  sequence: number;       // Monotonic per-client sequence number
  payload: ArrayBuffer;   // Command-specific data
}
```

Server processes commands in `(timestamp, sequence)` order within each tick. This prevents reordering artifacts when two players on the same ship send conflicting commands.

**Strategy 2: Client-Side Prediction (Helm Console Only)**

The helm console has the tightest latency requirements. We apply client-side prediction for own-ship movement only:

```
Client (Helm)                          Server
  │                                      │
  │  set impulse to 0.8                  │
  │  ├─ predict locally: ship.impulse = 0.8
  │  ├─ predict velocity, position       │
  │  ├─ render predicted state           │
  │  │                                   │
  │  ├── HELM_SET_IMPULSE(0.8) ────────→ │
  │  │                                   │ process next tick
  │  │                                   │ ship.impulse = 0.8
  │  │                                   │ compute authoritative state
  │  │                                   │
  │  │ ←── OBJECT_UPDATE (tick N) ─────  │
  │  │                                   │
  │  ├─ compare prediction vs server     │
  │  ├─ if close: smooth blend           │
  │  ├─ if far: snap to server           │
  │  │                                   │
```

**Prediction scope (helm only):**
- Position (x, y, z)
- Heading
- Velocity
- Warp factor

**Not predicted:**
- Shield state, energy levels (engineering manages these)
- Torpedo state (weapons officer's domain)
- NPC positions (server-only)
- Scan results (server-only)

**Reconciliation:**
- Small error (<50 units position, <0.1 radians heading): smooth interpolation over 100ms
- Large error (>50 units, >0.1 radians): instant snap to server state
- Prediction buffer: keep last 10 predicted states for comparison

**Strategy 3: Visual Interpolation (All Consoles)**

Non-helm consoles display server state with visual interpolation between updates:

```
Server updates at 10Hz (every 100ms):
  T=0ms    T=100ms   T=200ms
  State A  State B   State C
     │        │         │
Client interpolates between received states:
  T=0  T=25  T=50  T=75  T=100
  A    A+25% A+50% A+75% B
```

**Implementation:**
- Client maintains a 100ms interpolation buffer (one server tick)
- Entities smoothly move between last two known positions
- Buffer absorbs jitter: if a packet arrives late (up to 150ms), interpolation continues smoothly
- If buffer runs dry (packet loss > 150ms), hold last known position (freeze, don't extrapolate wildly)

### 7.3 Conflict Resolution

**Same-ship conflicts (two players, one ship):**

In a bridge simulator, console roles partition authority. Conflicts are rare because each console controls different systems. But edge cases exist:

| Conflict | Resolution |
|----------|-----------|
| Two players claim same console | First-come-first-served. Second player gets rejection. |
| Helm and Captain both set course | Last-write-wins within same tick. Captain override is a future feature. |
| Engineering sets energy while weapons fires | No conflict - different systems. Energy change applies next tick, weapon fires this tick. |
| Shield toggle from multiple sources | Idempotent toggle - result is consistent regardless of order. |

**Cross-ship conflicts:**

Ships are independent entities. No cross-ship conflicts exist in the game model. Ships can't modify each other's state directly. Combat (beams, torpedoes) is resolved by the server simulation with no player-to-player authority disputes.

**Entity ownership:**
```
Entity Type     │ Authority
────────────────┼──────────────────────
Player Ship     │ Server (inputs from crew)
NPC Ship        │ Server (AI-controlled)
Base/Station    │ Server
Torpedo         │ Server (fired by player, then autonomous)
Mine            │ Server
Nebula          │ Server (static)
```

No entity ownership transfers. The server is always authoritative. This eliminates an entire class of distributed systems problems.

---

## 8. Data Serialization & Bandwidth Optimization

### 8.1 Protocol Layers

```
┌─────────────────────────────────────────────┐
│  Application Layer                          │
│  ┌───────────────┐  ┌───────────────────┐  │
│  │ Game Protocol  │  │ Session Protocol  │  │
│  │ (binary, fast) │  │ (MessagePack)     │  │
│  └───────┬───────┘  └───────┬───────────┘  │
│          │                  │               │
│  ┌───────▼──────────────────▼───────────┐  │
│  │      Framing (packet header)          │  │
│  │      magic + length + origin + type   │  │
│  └───────────────────┬──────────────────┘  │
│                      │                      │
│  ┌───────────────────▼──────────────────┐  │
│  │      WebSocket (binary frames)        │  │
│  └──────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

**Game Protocol (existing, optimized for bandwidth):**
- Binary encoding with `PacketBuffer`
- Bit-field delta compression for entity updates
- Little-endian, fixed-width numeric fields
- Used for: all real-time game state and commands

**Session Protocol (new, optimized for developer ergonomics):**
- MessagePack encoding (binary JSON, ~30% smaller than JSON)
- Used for: lobby operations, chat, session management, player roster
- These messages are infrequent and small - developer convenience > raw performance

### 8.2 Bandwidth Optimization Techniques

**Technique 1: Bit-field Delta Compression (existing)**

Only changed properties are sent. A bit-field indicates which properties are present.

```
Full PlayerShip state:  ~120 bytes (18 properties)
Typical delta update:   ~30 bytes  (4-5 changed properties)
Compression ratio:      ~4:1 for steady-state updates
```

**Technique 2: Interest-Based Filtering (new)**

Players only receive updates for entities within their relevance sphere (see Section 3.3).

```
Without filtering:  150 entities × 48 players = 7,200 update messages per tick
With filtering:     ~30 entities × 48 players = 1,440 update messages per tick
Reduction:          ~5x fewer messages
```

**Technique 3: Update Rate Tiering (new)**

Different entity categories update at different rates (see Section 3.2).

```
Without tiering:  All entities at 10Hz
With tiering:     Own ship 10Hz, near 10Hz, mid 2Hz, far 1Hz
Effective rate:   ~4Hz average across all entities
Reduction:        ~2.5x fewer updates
```

**Technique 4: Packet Batching (existing, enhanced)**

Multiple entity updates are batched into a single WebSocket frame:

```
Current:   One OBJECT_UPDATE packet per entity type per tick
Enhanced:  One OBJECT_UPDATE packet per ship per tick (all relevant entities)
```

This reduces WebSocket frame overhead (2-14 bytes per frame) and system call overhead.

**Technique 5: Quantization (new)**

Reduce precision where full precision isn't needed:

| Field | Current | Optimized | Savings |
|-------|---------|-----------|---------|
| Position (x, z) | float64 | float32 | 50% (32-bit sufficient for 200k unit map) |
| Heading | float64 | uint16 (0-65535 → 0-2π) | 75% |
| Velocity | float64 | float32 | 50% |
| Shield values | float64 | uint16 (0-65535) | 75% |
| Energy | float64 | uint16 | 75% |

**Combined bandwidth savings estimate:**

```
Baseline (current):        ~50 KB/s per player (no filtering, full precision)
+ Interest filtering:      ~10 KB/s per player
+ Update rate tiering:     ~6 KB/s per player
+ Quantization:            ~4 KB/s per player
+ Batching improvements:   ~3.5 KB/s per player

48 players × 3.5 KB/s = ~170 KB/s total outbound (vs ~2.4 MB/s baseline)
```

### 8.3 Packet Type Registry (Extended)

New packet types for session management:

```typescript
// Session Protocol (Gateway ↔ Client)
const SESSION_PACKETS = {
  // Client → Gateway
  AUTH:            0x50000001,  // { name, token }
  LIST_SESSIONS:   0x50000002,  // {}
  CREATE_SESSION:  0x50000003,  // { name, scenario, maxShips, password? }
  JOIN_SESSION:    0x50000004,  // { sessionId, password? }

  // Gateway → Client
  AUTH_OK:         0x50000011,  // { playerId }
  AUTH_FAIL:       0x50000012,  // { reason }
  SESSION_LIST:    0x50000013,  // { sessions[] }
  SESSION_CREATED: 0x50000014,  // { sessionId, wsUrl }
  REDIRECT:        0x50000015,  // { wsUrl, reconnectToken }
  ERROR:           0x50000016,  // { code, message }
} as const;

// Session Protocol (Game Server ↔ Client, extending existing)
const GAME_SESSION_PACKETS = {
  // Client → Game Server
  REJOIN:          0x60000001,  // { playerId, reconnectToken }
  LEAVE:           0x60000002,  // {}
  CHAT:            0x60000003,  // { message }

  // Game Server → Client
  SESSION_STATE:   0x60000011,  // { ships, players, gameState, fullSnapshot }
  PLAYER_JOINED:   0x60000012,  // { playerId, name, ship, console }
  PLAYER_LEFT:     0x60000013,  // { playerId, reason }
  PLAYER_MOVED:    0x60000014,  // { playerId, newShip?, newConsole? }
  CHAT_MESSAGE:    0x60000015,  // { from, message }
} as const;
```

---

## 9. Implementation Roadmap

### Phase 1: Foundation (Sessions & Lobby)

**Goal:** Players can create/join games through a lobby.

| Task | Description | Effort |
|------|-------------|--------|
| 1.1 | Add session state machine (lobby/running/results) to GameServer | S |
| 1.2 | Implement player identity (name, playerId, reconnect token) | S |
| 1.3 | Build Gateway server (session catalog, player routing) | M |
| 1.4 | Add session protocol packets (MessagePack) | S |
| 1.5 | Update web client with lobby UI (create/join/ship-console picker) | M |
| 1.6 | Migrate from TCP+WS to WebSocket-only | M |

**Deliverable:** Multi-game server with lobby. Players choose games and consoles through a web UI.

### Phase 2: Resilience (Reconnection & Interest Management)

**Goal:** Players can reconnect after drops. Bandwidth scales efficiently.

| Task | Description | Effort |
|------|-------------|--------|
| 2.1 | Implement reconnection protocol (token-based, 60s window) | M |
| 2.2 | Add interest management (per-ship relevance spheres) | M |
| 2.3 | Implement update rate tiering (near/mid/far zones) | S |
| 2.4 | Add tick counter and timestamps to state updates | S |
| 2.5 | Client-side interpolation buffer (100ms) | M |
| 2.6 | Quantize network fields (float64 → float32/uint16) | S |

**Deliverable:** Robust multiplayer that handles network issues gracefully and scales to 48 players.

### Phase 3: Polish (Prediction & Optimization)

**Goal:** Responsive controls, optimized performance.

| Task | Description | Effort |
|------|-------------|--------|
| 3.1 | Helm client-side prediction with server reconciliation | L |
| 3.2 | Input timestamping and ordering | S |
| 3.3 | Packet batching optimization (per-ship bundles) | S |
| 3.4 | Performance profiling and optimization at 48-player load | M |
| 3.5 | Stress testing harness (simulated 48 clients) | M |
| 3.6 | Monitoring and metrics (tick time, bandwidth, player count) | S |

**Deliverable:** Production-quality multiplayer with good "feel" and measurable performance characteristics.

### Effort Key
- **S** (Small): < 1 day
- **M** (Medium): 1-3 days
- **L** (Large): 3-5 days

---

## Appendix: Protocol Reference

### Existing Packet Header (unchanged)

```
Offset  Size  Field         Value
0       4     Magic         0xdeadbeef
4       4     TotalLength   Header (24) + Payload
8       4     Origin        0x01 (server) | 0x02 (client)
12      4     Padding       0x00000000
16      4     Remaining     TotalLength - 20
20      4     PacketType    CRC32 hash identifier
24      var   Payload       Type-specific data
```

### Entity Update Bit-Field (unchanged)

```
PlayerShip (18 bits):
  0: targetId      1: impulse       2: heading       3: velocity
  4: posX          5: posY          6: posZ          7: shieldsFore
  8: shieldsAft    9: shieldsActive 10: energy       11: warpFactor
  12: reverse      13: docked       14: redAlert     15: mainScreenView
  16: autoBeams    17: beamFrequency

NPCShip (12 bits):
  0: name          1: posX          2: posY          3: posZ
  4: heading       5: velocity      6: faction       7: shieldsFore
  8: shieldsAft    9: surrendered   10: inNebula     11: scanState
```

### Interest Management Ranges

```
Zone        Range      Update Rate   Properties Sent
──────────  ─────────  ───────────   ──────────────────────
Own Ship    N/A        10-20 Hz      All properties
Near        < 10,000   10 Hz         All properties
Mid         10k-20k    2 Hz          Position, heading, shields
Far         20k-50k    1 Hz          Position only
Out of Range > 50,000  0 Hz          Not sent (removed from client)
```

### Reconnection State Machine

```
CONNECTED ──(ws close)──→ DISCONNECTED ──(60s timeout)──→ RELEASED
                               │
                          (ws reconnect + valid token)
                               │
                               ▼
                          RECONNECTING ──(session state sent)──→ CONNECTED
```

---

*End of design document.*
