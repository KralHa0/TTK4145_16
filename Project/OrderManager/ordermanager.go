package ordermanager

import (
	"sync"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

// --------------------------------------------------
// Internal state (owned ONLY by UpdaterRun goroutine)
// --------------------------------------------------

var (
	localWv   def.Worldview
	localWvMu sync.RWMutex
)

// peerWorldviews stores the last known worldview per peer ID
// for use in allAliveHallAtOrAbove consensus checks.
var peerWorldviews = make(map[def.NodeID]def.Worldview)

// --------------------------------------------------
// Initialization
// --------------------------------------------------

func OrderManagerInit(localNodeID def.NodeID, initialCabRequests [def.NumFloors]def.OrderState) {
	if !localNodeID.IsValid() {
		panic("OrderManagerInit: invalid node ID — check NetworkInit ran before OrderManagerInit")
	}

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
	peerWvCh         <-chan def.Worldview,
	orderCompleteCh  <-chan def.OrderMessage,
	newOrderCh       <-chan def.NewOrderMessage,
	networkWvCh      chan<- def.Worldview,
	orderHandlerWvCh chan<- def.Worldview,
	getAliveList     func() def.AliveList,
) {
	ticker := time.NewTicker(def.MsgFrq)
	defer ticker.Stop()

	for {
		select {

		// Periodic broadcast — non-blocking to avoid stalling the loop
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

		// New button press from FSM
		case newOrder := <-newOrderCh:
			applyNewOrder(newOrder)
			sendToOrderHandler(orderHandlerWvCh)

		// Order completion from FSM
		case orderMsg := <-orderCompleteCh:
			applyCompletion(orderMsg.Floor, orderMsg.Dir)
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

			// NoCall -> Exist: propagate existence from any peer
			if local == def.NoCall && peer >= def.Exist {
				if validTransition(local, def.Exist) {
					localWv.HallRequests[floor][dir] = def.Exist
				}
			}

			// Exist -> Acknowledged: all alive peers must confirm Exist
			if localWv.HallRequests[floor][dir] == def.Exist {
				if allAliveHallAtOrAbove(floor, dir, def.Exist, aliveList) {
					if validTransition(localWv.HallRequests[floor][dir], def.Acknowledged) {
						localWv.HallRequests[floor][dir] = def.Acknowledged
						reachedAck = true
					}
				}
			}

			// Acknowledged -> Complete: adopt completion signal from peer
			if localWv.HallRequests[floor][dir] == def.Acknowledged && peer == def.Complete {
				if validTransition(localWv.HallRequests[floor][dir], def.Complete) {
					localWv.HallRequests[floor][dir] = def.Complete
				}
			}

			// Complete -> NoCall: all alive peers confirm Complete, or peer already reset
			if localWv.HallRequests[floor][dir] == def.Complete {
				if allAliveHallAtOrAbove(floor, dir, def.Complete, aliveList) || peer == def.NoCall {
					if validTransition(localWv.HallRequests[floor][dir], def.NoCall) {
						localWv.HallRequests[floor][dir] = def.NoCall
					}
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
	if msg.CallType == def.Cabcall {
		cur := localWv.Nodes[0].CabRequests[msg.Floor]
		if validTransition(cur, def.Acknowledged) {
			localWv.Nodes[0].CabRequests[msg.Floor] = def.Acknowledged
		}
	} else {
		cur := localWv.HallRequests[msg.Floor][msg.Dir]
		if validTransition(cur, def.Exist) {
			localWv.HallRequests[msg.Floor][msg.Dir] = def.Exist
		}
	}
}

// --------------------------------------------------
// Completion from FSM
// --------------------------------------------------

func applyCompletion(floor int, dir def.Direction) {
	cur := localWv.HallRequests[floor][dir]
	if validTransition(cur, def.Complete) {
		localWv.HallRequests[floor][dir] = def.Complete
	}

	// Cab requests are local-only — clear immediately, no peer consensus needed
	cabCur := localWv.Nodes[0].CabRequests[floor]
	if validTransition(cabCur, def.NoCall) {
		localWv.Nodes[0].CabRequests[floor] = def.NoCall
	}
}

// --------------------------------------------------
// Alive consensus helper
// --------------------------------------------------

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

		if peerID == localWv.Nodes[0].ID {
			if localWv.HallRequests[floor][dir] < threshold {
				return false
			}
			continue
		}

		peerWv, known := peerWorldviews[peerID]
		if !known {
			return false
		}
		if peerWv.HallRequests[floor][dir] < threshold {
			return false
		}
	}
	return true
}

// --------------------------------------------------
// Transition guard
// --------------------------------------------------

func validTransition(from, to def.OrderState) bool {
	return to == from+1 || (from == def.Complete && to == def.NoCall)
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