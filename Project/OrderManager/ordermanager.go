package ordermanager

import (
	"fmt"
	"sync"
	"time"

	"Driver-go/elevio"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
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

	// Merge peer nodes into localWv so ORA sees all elevators
	for _, peerNode := range peerWv.Nodes {
		if !peerNode.ID.IsValid() {
			continue
		}
		if peerNode.ID == localNode().ID {
			// Recover our own cab calls from peer memory on reconnect.
			// Only apply if we have no local knowledge of that floor's state
			// (cabCallKnown is false only on fresh startup). This prevents
			// stale peer worldviews from re-activating calls we already served.
			for f := 0; f < def.NumFloors; f++ {
				if !cabCallKnown[f] && localNode().CabRequests[f] == def.NoCall && peerNode.CabRequests[f] == def.Acknowledged {
					localNode().CabRequests[f] = def.Acknowledged
					cabCallKnown[f] = true
				}
			}
			continue
		}
		found := false
		for i := range localWv.Nodes {
			if localWv.Nodes[i].ID == peerNode.ID {
				localWv.Nodes[i].Elevator = peerNode.Elevator
				localWv.Nodes[i].CabRequests = peerNode.CabRequests
				found = true
				break
			}
		}
		if !found {
			localWv.Nodes = append(localWv.Nodes, peerNode)
		}
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

			// Complete -> NoCall: adopt from peer that already cleared it.
			// When the first node to clear broadcasts NoCall, other nodes see
			// NoCall (0) < Complete (3) and allAliveHallAtOrAbove never passes —
			// this rule breaks that deadlock.
			if localWv.HallRequests[floor][dir] == def.Complete && peer == def.NoCall {
				localWv.HallRequests[floor][dir] = def.NoCall
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
			cabCallKnown[msg.Floor] = true
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
		cabCallKnown[floor] = true
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

// applyHallConsensus advances hall request states that can be resolved with
// the current alive list alone, without needing a peer worldview message.
// Handles Exist→Acknowledged and Complete→NoCall for isolated elevators.
// Caller must hold localWvMu write lock.
func applyHallConsensus(aliveList def.AliveList) bool {
	reachedAck := false
	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {
			if localWv.HallRequests[floor][dir] == def.Exist {
				if allAliveHallAtOrAbove(floor, dir, def.Exist, aliveList) {
					if validTransition(localWv.HallRequests[floor][dir], def.Acknowledged) {
						localWv.HallRequests[floor][dir] = def.Acknowledged
						reachedAck = true
					}
				}
			}
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
		fmt.Println("[OM] Sending to ORA")
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

// --------------------------------------------------
// Print helpers
// --------------------------------------------------

func orderStateStr(s def.OrderState) string {
	switch s {
	case def.NoCall:
		return "N"
	case def.Exist:
		return "E"
	case def.Acknowledged:
		return "A"
	case def.Complete:
		return "C"
	default:
		return "?"
	}
}

// PrintNode prints a single node's ID, elevator state, and cab requests.
func PrintNode(node def.Node) {
	stateStr := []string{"Moving", "Idle", "DoorOpen"}
	state := "?"
	if int(node.Elevator.ElevState) < len(stateStr) {
		state = stateStr[node.Elevator.ElevState]
	}
	fmt.Printf("Node %-20s floor=%-2d dir=%-5v state=%-8s malfunction=%v\n",
		node.ID, node.Elevator.CurrentFloor, node.Elevator.Direction, state, node.Elevator.Malfunctioned)
	fmt.Printf("  CabRequests: ")
	for f := 0; f < def.NumFloors; f++ {
		fmt.Printf("[%d:%s]", f, orderStateStr(node.CabRequests[f]))
	}
	fmt.Println()
}

// PrintHallCalls prints the hall requests table (floor, Up, Down).
func PrintHallCalls(hallRequests [def.NumFloors][2]def.OrderState) {
	fmt.Printf("%-6s  %-4s  %-4s\n", "Floor", "Up", "Down")
	for f := 0; f < def.NumFloors; f++ {
		fmt.Printf("%-6d  %-4s  %-4s\n", f,
			orderStateStr(hallRequests[f][def.DirUp]),
			orderStateStr(hallRequests[f][def.DirDown]))
	}
}

// PrintWv prints the full worldview as a table: hall calls (Up/Down) as the
// first column group, followed by one column per node (cab request per floor).
// An elevator-status row is printed below the header.
func PrintWv(wv def.Worldview) {
	const colW = 16
	dirStr := map[int]string{-1: "Dn", 0: "--", 1: "Up"}
	stateStr := []string{"Mov", "Idle", "Door"}

	// Header: node IDs
	fmt.Printf("%-6s  %-4s  %-4s", "Floor", "Up", "Dn")
	for _, node := range wv.Nodes {
		id := string(node.ID)
		if len(id) > colW {
			id = id[:colW]
		}
		fmt.Printf("  %-*s", colW, id)
	}
	fmt.Println()

	// Elevator status row
	fmt.Printf("%-6s  %-4s  %-4s", "", "", "")
	for _, node := range wv.Nodes {
		alivelist := nw.GetAliveList()
		f := "F"
		if alivelist.Peers[node.ID] == true {
			f = "T"
		}
		e := node.Elevator
		dir := dirStr[int(e.Direction)]
		state := "?"
		if int(e.ElevState) < len(stateStr) {
			state = stateStr[e.ElevState]
		}
		malf := "OK"
		if e.Malfunctioned {
			malf = "MF"
		}
		info := fmt.Sprintf("%s %s %s %s", f, dir, state, malf)
		fmt.Printf("  %-*s", colW, info)
	}
	fmt.Println()

	// Separator
	sepLen := 18 + len(wv.Nodes)*(colW+2)
	for i := 0; i < sepLen; i++ {
		fmt.Print("-")
	}
	fmt.Println()

	// One row per floor
	for f := 0; f < def.NumFloors; f++ {
		fmt.Printf("%-6d  %-4s  %-4s",
			f,
			orderStateStr(wv.HallRequests[f][def.DirUp]),
			orderStateStr(wv.HallRequests[f][def.DirDown]))
		for _, node := range wv.Nodes {
			fmt.Printf("  %-*s", colW, orderStateStr(node.CabRequests[f]))
		}
		fmt.Println()
	}
}

// printIncomingWv prints a peer worldview as it arrives — comment out to silence.
func printIncomingWv(wv def.Worldview) {
	if len(wv.Nodes) == 0 {
		return
	}
	fmt.Printf("[OM] Incoming WV from %s:\n", wv.Nodes[0].ID)
	PrintWv(wv)
}
