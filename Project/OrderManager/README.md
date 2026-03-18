# OrderManager

Single owner of the shared system state (the *worldview*). All state changes go through the `UpdaterRun` goroutine — no other part of the system writes to the worldview directly.

## What it does

- Merges incoming peer worldviews with the local state using a consensus protocol
- Advances hall and cab requests through a four-stage state machine: `NoCall → Exist → Acknowledged → Complete → NoCall`
- Forwards the worldview to the OrderAssigner when a new acknowledged order is ready
- Notifies the FSM when the assigned orders or elevator state change
- Broadcasts the local worldview to the network on a 100 ms tick

## State machine

Transitions require consensus from all alive peers before advancing:

| Transition | Trigger |
|------------|---------|
| `NoCall → Exist` | Button press received from FSM |
| `Exist → Acknowledged` | All alive peers have seen `Exist` |
| `Acknowledged → Complete` | FSM reports order served |
| `Complete → NoCall` | All alive peers have seen `Complete` |

Cab calls additionally use `cabCallKnown` to prevent a stale peer worldview from re-activating a call that was already served after a restart.

## Key files

| File | Description |
|------|-------------|
| `ordermanager.go` | State variables, init, and the `UpdaterRun` event loop |
| `ordermanagerLogic.go` | `mergePeerWorldview`, consensus helpers (`allAliveCabAtOrAbove`, `allAliveHallAtOrAbove`), `applyHallConsensus` |
| `ordermanagerHelpers.go` | `applyNewOrder`, `applyCompletion`, print utilities |

## Channels (UpdaterRun)

| Channel | Direction | Description |
|---------|-----------|-------------|
| `peerWvCh` | in | Worldviews from peers (via NetworkHandler) |
| `newOrderCh` | in | Button events from FSM |
| `orderCompleteCh` | in | Completed orders from FSM |
| `malfunctionCh` | in | Malfunction flag from FSM |
| `fsmElevStateCh` | in | Elevator position/state from FSM |
| `networkWvCh` | out | Local worldview for broadcast |
| `orderHandlerWvCh` | out | Worldview trigger for OrderAssigner |
| `omToFsmWvCh` | out | Worldview for FSM light control |

## Crash recovery

On a fresh restart (`cabCallKnown` all-false), if a peer's stored copy of this node has a cab call in `Exist` or `Acknowledged` state, `mergePeerWorldview` restores it. This covers the window between a button press and acknowledgement being propagated to peers.
