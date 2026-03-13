package ordermanager

import (
	"Driver-go/elevio"
	"fmt"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
)

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
		info := fmt.Sprintf("%s%d %s %s %s", f, e.CurrentFloor, dir, state, malf)
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
