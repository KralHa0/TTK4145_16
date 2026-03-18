# TTK4145 Group 16 — Elevator Project

A distributed elevator control system written in Go. Multiple elevator nodes communicate over UDP, agree on a shared worldview, and cooperatively serve hall and cab calls.

## Approach

Each node runs an identical copy of the software. There is no central coordinator — nodes reach consensus by broadcasting their state and waiting until all alive peers have acknowledged each order transition before acting on it.

**Order lifecycle:** `NoCall → Exist → Acknowledged → Complete → NoCall`

A hall call is only served once all alive peers have confirmed they know about it (`Acknowledged`). Completion is only cleared once all peers have confirmed the order was served. This ensures no call is lost if a node disconnects mid-order, and no call is double-served.

Cab calls follow the same protocol but are local to each elevator. Peer memory is used to recover cab calls after a crash/restart.

Hall call assignment (which elevator takes which call) is computed by an external cost-function binary (HRA) run locally on each node.

## File structure

```
Project/
├── main.go                  — Entry point; wires all modules together
├── Definitions/             — Shared types: Worldview, Elevator, OrderState, NodeID, etc.
├── NetworkHandler/          — UDP broadcast and peer-discovery wrapper
├── OrderManager/            — Worldview owner; consensus state machine for all orders
├── OrderAssigner/           — Wraps the HRA binary to assign hall calls to elevators
├── StateMachine/            — Elevator FSM: Moving / Idle / DoorOpen states + hardware I/O
│   ├── fsm.go               — Init and main control loop
│   ├── state_moving.go      — Moving state handler
│   ├── state_idle.go        — Idle state handler
│   ├── state_dooropen.go    — DoorOpen state handler
│   ├── destination.go       — Next-floor destination calculation
│   ├── timers.go            — Door, watchdog, and floor timers
│   └── malfunction.go       — Malfunction detection and reporting
├── Hardware/                — Low-level elevator I/O driver (elevio) [borrowed]
├── Network/                 — Low-level UDP broadcast and peer-heartbeat library [borrowed]
└── Tester/                  — Integration test harnesses
```

## Data flow

```
Hardware buttons
      ↓
  StateMachine (FSM)
      ↓ button events / completions
  OrderManager  ←→  NetworkHandler  ←→  peers on network
      ↓ acknowledged worldview
  OrderAssigner (HRA binary)
      ↓ assigned orders
  StateMachine (FSM)
      ↓
  Hardware motor / lights
```

## Running

```bash
go run .           # start full elevator system
go run . nw        # network connectivity test
go run . om        # order manager unit test
go run . ora       # ORA + order manager integration test
go run . run       # multi-machine network + updater test
```

Requires the elevator simulator (or real hardware) listening on `localhost:15657`.

## Third-party code

The `Hardware/` (elevio driver) and `Network/` (UDP broadcast + peer heartbeat) packages are borrowed from the [TTK4145 course repository](https://github.com/TTK4145).
