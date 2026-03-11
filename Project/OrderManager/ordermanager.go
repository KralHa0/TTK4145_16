package ordermanager

import (
	"fmt"
	"sync"
	"time"

	"Driver-go/elevio"

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

// localNode returns a pointer to the local node, panicking if not initialised.
func localNode() *def.Node {
	if len(localWv.Nodes) == 0 {
		panic("ordermanager: localWv has no nodes — was OrderManagerInit called?")
	}
	return &localWv.Nodes[0]
}

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
	peerWvCh <-chan def.Worldview,
	orderCompleteCh <-chan def.FsmClearOrderMessage,
	newOrderCh <-chan elevio.ButtonEvent,
	networkWvCh chan<- def.Worldview,
	orderHandlerWvCh chan<- def.Worldview,
	omToFsmWvCh chan<- def.Worldview,
	getAliveList func() def.AliveList,
	malfunctionCh <-chan bool,
) {
	ticker := time.NewTicker(def.MsgFrq)
	defer ticker.Stop()
	oraTicker := time.NewTicker(def.OraFrq)
	defer oraTicker.Stop()

	for {
		select {

		// Periodic broadcast — non-blocking to avoid stalling the loop
		case <-ticker.C:
			localWvMu.RLock()
			wvCopy := deepCopyWorldview(localWv)
			localWvMu.RUnlock()
			select {
			case networkWvCh <- wvCopy:
			default:
			}

		// Periodic ORA re-evaluation
		case <-oraTicker.C:
			localWvMu.RLock()
			sendToOrderHandler(orderHandlerWvCh)
			localWvMu.RUnlock()

		// Merge incoming peer worldview
		case peerWv := <-peerWvCh:
			localWvMu.Lock()
			reachedAck := mergePeerWorldview(peerWv, getAliveList())
			if reachedAck {
				sendToOrderHandler(orderHandlerWvCh)
			}
			//fmt.Println("GetFromNW")
			sendToFsm(omToFsmWvCh)
			localWvMu.Unlock()

		// New button press from FSM
		case newOrder := <-newOrderCh:
			localWvMu.Lock()
			applyNewOrder(newOrder)
			sendToOrderHandler(orderHandlerWvCh)
			//sendToFsm(omToFsmWvCh)
			localWvMu.Unlock()

		// Order completion from FSM
		case orderMsg := <-orderCompleteCh:
			localWvMu.Lock()
			applyCompletion(orderMsg.Floor, orderMsg.Dir)
			//sendToFsm(omToFsmWvCh)
			localWvMu.Unlock()

		case malfunction := <-malfunctionCh:
			localWvMu.Lock()
			localNode().Elevator.Malfunctioned = malfunction
			localWvMu.Unlock()
		}

	}
}

// --------------------------------------------------
// Merge logic
// --------------------------------------------------

func mergePeerWorldview(peerWv def.Worldview, aliveList def.AliveList) bool {
	reachedAck := false

	// Store latest worldview per peer for consensus checks
	if len(peerWv.Nodes) > 0 && peerWv.Nodes[0].ID.IsValid() {
		peerWorldviews[peerWv.Nodes[0].ID] = deepCopyWorldview(peerWv)
	}

	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {

			local := localWv.HallRequests[floor][dir]
			peer := peerWv.HallRequests[floor][dir]

			// NoCall -> Exist: propagate existence from any peer
			if local == def.NoCall && (peer == def.Exist || peer == def.Acknowledged) {
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

			// Complete -> NoCall: all alive peers confirm Complete
			if localWv.HallRequests[floor][dir] == def.Complete {
				if allAliveHallAtOrAbove(floor, dir, def.Complete, aliveList) {
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

func applyNewOrder(msg elevio.ButtonEvent) {
	if msg.Button == elevio.BT_Cab {
		// Cab calls are local-only — set Acknowledged directly, no consensus needed
		if localNode().CabRequests[msg.Floor] == def.NoCall {
			localNode().CabRequests[msg.Floor] = def.Acknowledged
		}
	} else {
		// BT_HallUp=0 maps to dir index 0, BT_HallDown=1 maps to dir index 1
		dir := int(msg.Button)
		cur := localWv.HallRequests[msg.Floor][dir]
		if validTransition(cur, def.Exist) {
			localWv.HallRequests[msg.Floor][dir] = def.Exist
		}
	}
}

// --------------------------------------------------
// Completion from FSM
// --------------------------------------------------

func applyCompletion(floor int, dir def.DirectionUpDown) {
	cur := localWv.HallRequests[floor][dir]
	if validTransition(cur, def.Complete) {
		localWv.HallRequests[floor][dir] = def.Complete
	}

	// Cab requests are local-only — clear immediately, no peer consensus needed
	if localNode().CabRequests[floor] != def.NoCall {
		localNode().CabRequests[floor] = def.NoCall
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
	if len(aliveList.Peers) == 0 {
		return false
	}
	for peerID, alive := range aliveList.Peers {
		if !alive {
			continue
		}

		if peerID == localNode().ID {
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
		fmt.Println("Sending to ORA")
	default:
	}
}

func sendToFsm(omToFsmWvCh chan<- def.Worldview) {
	select {
	case omToFsmWvCh <- deepCopyWorldview(localWv):
		//fmt.Println("Sending to FSM")
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
