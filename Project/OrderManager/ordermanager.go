package ordermanager

import (
	hw "Driver-go/elevio"
	"sync"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

// --------------------------------------------------
// Internal state (owned ONLY by UpdaterRun goroutine)
// --------------------------------------------------

var (
	localWv   def.Worldview
	localWvMu sync.RWMutex // protects localWv for external reads via GetLocalWv
)

// peerWorldviews stores the last known worldview per peer ID,
// required for correct allAliveHallAtOrAbove consensus checks.
var peerWorldviews = make(map[string]def.Worldview)

// --------------------------------------------------
// Initialization
// --------------------------------------------------

func OrderManagerInit(localNodeID string, initialCabRequests [def.NumFloors]def.OrderState) {
	localWv = def.Worldview{
		Nodes: []def.Node{
			{
				ID:          localNodeID,
				CabRequests: initialCabRequests,
				Elevator:    def.Elevator{},
			},
		},
		HallRequests: [def.NumFloors][2]def.OrderState{},
	}
}

// --------------------------------------------------
// Main state owner goroutine
// --------------------------------------------------

func UpdaterRun(
	peerWvCh <-chan def.Worldview,
	orderCompleteCh <-chan def.OrderMessage,
	newOrderCh <-chan def.NewOrderMessage,
	networkWvCh chan<- def.Worldview,
	orderHandlerWvCh chan<- def.Worldview,
	getAliveList func() def.AliveList,
) {
	ticker := time.NewTicker(def.MsgFrq)
	defer ticker.Stop()

	for {
		select {

		// Periodic broadcast — non-blocking send to avoid stalling the loop
		// if the network layer is temporarily slow.
		case <-ticker.C:
			select {
			case networkWvCh <- deepCopyWorldview(localWv):
			default:
			}

		// Merge incoming peer worldview
		case peerWv := <-peerWvCh:
			reachedAck := mergePeerWorldview(peerWv, getAliveList())
			if reachedAck {
				sendToOrderHandler(orderHandlerWvCh)
			}

		// New button press from FSM (hall or cab)
		case newOrder := <-newOrderCh:
			applyNewOrder(newOrder)
			// Notify order handler immediately so it can (re)assign elevators
			sendToOrderHandler(orderHandlerWvCh)

		// Order completion from FSM
		case orderMsg := <-orderCompleteCh:
			applyCompletion(orderMsg.Floor, orderMsg.Direction)
		}
	}
}

// --------------------------------------------------
// Merge logic
// --------------------------------------------------

func mergePeerWorldview(peerWv def.Worldview, aliveList def.AliveList) bool {
	reachedAck := false

	// Store latest worldview per peer for consensus checks
	if len(peerWv.Nodes) > 0 {
		peerWorldviews[peerWv.Nodes[0].ID] = deepCopyWorldview(peerWv)
	}

	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {

			local := localWv.HallRequests[floor][dir]
			peer := peerWv.HallRequests[floor][dir]

			// Propagate existence: if any peer knows about a call, we adopt it
			if local == def.NoCall && peer >= def.Exist {
				localWv.HallRequests[floor][dir] = def.Exist
			}

			// Acknowledge once all alive peers have registered the call
			if localWv.HallRequests[floor][dir] == def.Exist {
				if allAliveHallAtOrAbove(floor, dir, def.Exist, aliveList) {
					localWv.HallRequests[floor][dir] = def.Acknowledged
					reachedAck = true
				}
			}

			// Adopt completion from peer once we are acknowledged
			if localWv.HallRequests[floor][dir] == def.Acknowledged && peer == def.Complete {
				localWv.HallRequests[floor][dir] = def.Complete
			}

			// Clear once all alive peers have seen the completion,
			// or if the peer has already cycled back to NoCall.
			if localWv.HallRequests[floor][dir] == def.Complete {
				if allAliveHallAtOrAbove(floor, dir, def.Complete, aliveList) ||
					peer == def.NoCall {
					localWv.HallRequests[floor][dir] = def.NoCall
				}
			}
		}
	}

	return reachedAck
}

// --------------------------------------------------
// New order from FSM
// --------------------------------------------------

func applyNewOrder(msg def.NewOrderMessage) {
    dir := motorDirToIndex(msg.Direction)

    if msg.CallType == def.Cabcall {
        // Cab calls are local-only — no peer consensus needed,
        // go straight to Acknowledged so the FSM can serve them immediately
        if localWv.Nodes[0].CabRequests[msg.Floor] == def.NoCall {
            localWv.Nodes[0].CabRequests[msg.Floor] = def.Acknowledged
        }
    } else {
        if localWv.HallRequests[msg.Floor][dir] == def.NoCall {
            localWv.HallRequests[msg.Floor][dir] = def.Exist
        }
    }
}

// --------------------------------------------------
// Completion from FSM
// --------------------------------------------------

// applyCompletion marks the hall request for the given direction — and the
// cab request — as Complete. Direction is required because stopping at a
// floor while travelling up should not clear a downward hall request
// (and vice versa). The cab request is always cleared since it is
// directionless.
func applyCompletion(floor int, direction hw.MotorDirection) {
    dir := motorDirToIndex(direction)

    if localWv.HallRequests[floor][dir] == def.Acknowledged {
        localWv.HallRequests[floor][dir] = def.Complete
    }

    // Cab requests are local-only — clear immediately on completion,
    // no peer consensus step needed (unlike hall requests)
    if localWv.Nodes[0].CabRequests[floor] == def.Acknowledged {
        localWv.Nodes[0].CabRequests[floor] = def.NoCall
    }
}

// --------------------------------------------------
// Alive consensus helper
// --------------------------------------------------

// allAliveHallAtOrAbove checks that every alive peer's last known worldview
// has the given hall request at or above `threshold`. This was previously
// broken — it only checked localWv for every peer instead of each peer's
// own worldview.
func allAliveHallAtOrAbove(
	floor int,
	dir int,
	threshold def.OrderState,
	aliveList def.AliveList,
) bool {
	for peerID, alive := range aliveList.Peers {
		if !alive {
			continue
		}

		// Local node: check localWv directly
		if peerID == localWv.Nodes[0].ID {
			if localWv.HallRequests[floor][dir] < threshold {
				return false
			}
			continue
		}

		// Remote peer: use last stored worldview
		peerWv, known := peerWorldviews[peerID]
		if !known {
			// We have no worldview from this peer yet — cannot confirm
			return false
		}
		if peerWv.HallRequests[floor][dir] < threshold {
			return false
		}
	}
	return true
}

// --------------------------------------------------
// External safe read
// --------------------------------------------------

func GetLocalWv() def.Worldview {
	localWvMu.RLock()
	defer localWvMu.RUnlock()
	return deepCopyWorldview(localWv)
}

// --------------------------------------------------
// Internal helpers
// --------------------------------------------------

// sendToOrderHandler sends a deep copy of localWv to the order handler.
// Non-blocking: if the handler is busy the worldview will arrive on the
// next state change instead, preventing the UpdaterRun loop from stalling.
func sendToOrderHandler(orderHandlerWvCh chan<- def.Worldview) {
	select {
	case orderHandlerWvCh <- deepCopyWorldview(localWv):
	default:
	}
}

func deepCopyWorldview(src def.Worldview) def.Worldview {
	copyWv := src

	nodesCopy := make([]def.Node, len(src.Nodes))
	copy(nodesCopy, src.Nodes)
	copyWv.Nodes = nodesCopy

	return copyWv
}

// motorDirToIndex converts MD_Up → 0, MD_Down → 1 for use as a
// HallRequests array index. MD_Stop should never be passed here.
func motorDirToIndex(dir hw.MotorDirection) int {
	if dir == hw.MD_Up {
		return 0
	}
	return 1
}
