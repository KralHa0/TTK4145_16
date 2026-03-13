package ordermanager

import (
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

// cabCallKnown tracks floors where we have definitive local knowledge of the
// cab call state (registered or cleared by us). Only false on fresh startup,
// which is the only time peer recovery should be trusted.
var cabCallKnown [def.NumFloors]bool

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
	cabCallKnown = [def.NumFloors]bool{}
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
	fsmElevStateCh <-chan def.Elevator,
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
			localWvMu.Lock()
			applyHallConsensus(getAliveList())
			sendToOrderHandler(orderHandlerWvCh)
			wvCopy := deepCopyWorldview(localWv)
			localWvMu.Unlock()
			PrintWv(wvCopy)

		// Merge incoming peer worldview
		case peerWv := <-peerWvCh:
			//printIncomingWv(peerWv)
			localWvMu.Lock()
			reachedAck := mergePeerWorldview(peerWv, getAliveList())
			if reachedAck {
				sendToOrderHandler(orderHandlerWvCh)
			}
			sendToFsm(omToFsmWvCh)
			localWvMu.Unlock()


		// New button press from FSM
		case newOrder := <-newOrderCh:
			localWvMu.Lock()
			applyNewOrder(newOrder)
			sendToOrderHandler(orderHandlerWvCh)
			localWvMu.Unlock()
			PrintWv(GetLocalWv())

		// Order completion from FSM
		case orderMsg := <-orderCompleteCh:
			localWvMu.Lock()
			applyCompletion(orderMsg.Floor, orderMsg.Dir)
			applyHallConsensus(getAliveList())
			sendToOrderHandler(orderHandlerWvCh)
			sendToFsm(omToFsmWvCh)
			localWvMu.Unlock()
			PrintWv(GetLocalWv())

		// Malfunction from fsm
		case malfunction := <-malfunctionCh:
			localWvMu.Lock()
			localNode().Elevator.Malfunctioned = malfunction
			localWvMu.Unlock()

		// Current elevator state from fsm
		case elevState := <-fsmElevStateCh:
			localWvMu.Lock()
			localNode().Elevator.CurrentFloor = elevState.CurrentFloor
			localNode().Elevator.Direction = elevState.Direction
			localNode().Elevator.ElevState = elevState.ElevState
			wvCopy := deepCopyWorldview(localWv)
			localWvMu.Unlock()
			select {
			case networkWvCh <- wvCopy:
			default:
			}
		}

	}
}
