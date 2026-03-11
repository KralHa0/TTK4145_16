package tester

import (
	"Driver-go/elevio"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw  "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
	oa  "github.com/KralHa0/TTK4145_16/Project/OrderAssigner"
	om  "github.com/KralHa0/TTK4145_16/Project/OrderManager"
)

// RunORAOMTest tests the OM → ORA pipeline with hardcoded order scenarios.
// A self-feedback loop feeds OM's broadcast worldview back as a peer worldview,
// so single-node consensus (Exist → Acknowledged) is reached automatically.
func RunORAOMTest() {
	fmt.Println("=== ORA + OM Integration Test ===")

	nw.NetworkInit()
	localID := nw.GetIp()
	fmt.Println("Local ID:", localID)

	om.OrderManagerInit(localID, [def.NumFloors]def.OrderState{})

	peerWvCh        := make(chan def.Worldview, 10)
	orderCompleteCh := make(chan def.FsmClearOrderMessage, 10)
	newOrderCh      := make(chan elevio.ButtonEvent, 10)
	networkWvCh     := make(chan def.Worldview, 10)
	omToOraWvCh     := make(chan def.Worldview, 10)
	omToFsmWvCh     := make(chan def.Worldview, 10)
	oaToFsmCh       := make(chan def.AssignedOrders, 10)
	malfunctionCh   := make(chan bool, 10)

	// Single-node alive list: consensus is reached as soon as local node has Exist
	getAliveList := func() def.AliveList {
		return def.AliveList{Peers: map[def.NodeID]bool{localID: true}}
	}

	go om.UpdaterRun(
		peerWvCh,
		orderCompleteCh,
		newOrderCh,
		networkWvCh,
		omToOraWvCh,
		omToFsmWvCh,
		getAliveList,
		malfunctionCh,
	)

	// Self-feedback: OM's outgoing broadcast feeds back as a peer worldview.
	// This triggers Exist -> Acknowledged consensus for a single-node setup.
	go func() {
		for wv := range networkWvCh {
			select {
			case peerWvCh <- wv:
			default:
			}
		}
	}()

	orderAssigner := oa.NewOrderAssigner(localID, omToOraWvCh, oaToFsmCh)
	go orderAssigner.Run()

	go drainNetwork(omToFsmWvCh)
	go printAssignedOrders(oaToFsmCh)
	go oraOmScenarios(newOrderCh, orderCompleteCh)

	select {}
}

// printAssignedOrders prints the ORA output whenever a new assignment arrives.
func printAssignedOrders(oaToFsmCh <-chan def.AssignedOrders) {
	for orders := range oaToFsmCh {
		fmt.Println("\n[ORA OUTPUT] Assigned orders for this node:")
		for floor := 0; floor < def.NumFloors; floor++ {
			fmt.Printf("  Floor %d: up=%v  down=%v\n", floor, orders[floor][0], orders[floor][1])
		}
	}
}

// oraOmScenarios injects three hardcoded order sets at timed intervals.
func oraOmScenarios(
	newOrderCh      chan<- elevio.ButtonEvent,
	orderCompleteCh chan<- def.FsmClearOrderMessage,
) {
	// Allow OM and ORA to start up
	time.Sleep(500 * time.Millisecond)

	// --------------------------------------------------
	// Scenario A: 2 hall calls + 1 cab call
	// --------------------------------------------------
	fmt.Println("\n=== Scenario A: hall up@0, hall down@3, cab@2 ===")
	newOrderCh <- elevio.ButtonEvent{Floor: 0, Button: elevio.BT_HallUp}
	newOrderCh <- elevio.ButtonEvent{Floor: 3, Button: elevio.BT_HallDown}
	newOrderCh <- elevio.ButtonEvent{Floor: 2, Button: elevio.BT_Cab}

	time.Sleep(3 * time.Second)
	fmt.Println("\n[SIM] Completing Scenario A orders")
	orderCompleteCh <- def.FsmClearOrderMessage{Floor: 0, Dir: def.DirUp}
	orderCompleteCh <- def.FsmClearOrderMessage{Floor: 3, Dir: def.DirDown}
	orderCompleteCh <- def.FsmClearOrderMessage{Floor: 2, Dir: def.DirUp} // cab: dir ignored
	time.Sleep(500 * time.Millisecond)
	fmt.Println("\n--- Worldview after Scenario A ---")
	printWorldview(om.GetLocalWv())

	time.Sleep(1 * time.Second)

	// --------------------------------------------------
	// Scenario B: all floors hall up (stress test)
	// --------------------------------------------------
	fmt.Println("\n=== Scenario B: hall up on all floors ===")
	for floor := 0; floor < def.NumFloors; floor++ {
		newOrderCh <- elevio.ButtonEvent{Floor: floor, Button: elevio.BT_HallUp}
	}

	time.Sleep(3 * time.Second)
	fmt.Println("\n[SIM] Completing Scenario B orders")
	for floor := 0; floor < def.NumFloors; floor++ {
		orderCompleteCh <- def.FsmClearOrderMessage{Floor: floor, Dir: def.DirUp}
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Println("\n--- Worldview after Scenario B ---")
	printWorldview(om.GetLocalWv())

	time.Sleep(1 * time.Second)

	// --------------------------------------------------
	// Scenario C: only cab calls
	// --------------------------------------------------
	fmt.Println("\n=== Scenario C: cab calls at floor 1 and floor 3 ===")
	newOrderCh <- elevio.ButtonEvent{Floor: 1, Button: elevio.BT_Cab}
	newOrderCh <- elevio.ButtonEvent{Floor: 3, Button: elevio.BT_Cab}

	time.Sleep(3 * time.Second)
	fmt.Println("\n--- Worldview after Scenario C ---")
	printWorldview(om.GetLocalWv())

	fmt.Println("\n=== ORA + OM Test complete ===")
}
