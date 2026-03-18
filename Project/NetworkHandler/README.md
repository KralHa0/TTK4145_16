# NetworkHandler

Thin wrapper around the Network-go library that manages UDP broadcast and peer discovery for the elevator system.

## What it does

- Broadcasts the local worldview to all peers at a fixed interval via UDP
- Receives worldviews from peers and forwards them to the OrderManager
- Tracks which elevators are currently alive using a peer-discovery heartbeat
- Exposes the local IP address as the node's unique ID

## Channels

| Channel | Direction | Description |
|---------|-----------|-------------|
| `localWvCh` | in | Worldviews to broadcast on the network |
| `peerWvCh` | out | Worldviews received from peers |
| `peerUpdateCh` | out | Peer connect/disconnect events |

## Key functions

- `NetworkInit()`: starts all transmitter/receiver goroutines; must be called before anything else
- `NetworkRun(...)`: main loop; bridges the above channels to the underlying bcast/peers library
- `UpdateAliveList(update)`: updates the alive-peer map from a `PeerUpdate` event
- `GetAliveList()`: returns a safe copy of the current alive-peer map (used by OrderManager for consensus)
- `GetIp()`: returns the local IP as a `NodeID`

## Notes

- Node identity is the local IP address. Two processes on the same machine will have the same ID.
- If the network is unavailable at startup, `CheckIP()` returns `"DISCONNECTED"` and the system panics in `main`.
- Outbound sends are non-blocking (drop on full buffer) to prevent the network loop from stalling state updates.
