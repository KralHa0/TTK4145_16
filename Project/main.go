package main

import (
	"Network-go/network/peers"
	"fmt"
	"os"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	fsm "github.com/KralHa0/TTK4145_16/Project/FSM"
	nw "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
	oa "github.com/KralHa0/TTK4145_16/Project/OrderAssigner"
	om "github.com/KralHa0/TTK4145_16/Project/OrderManager"
	tester "github.com/KralHa0/TTK4145_16/Project/Tester"
)

func main() {
	if len(os.Args) < 2 {
		runFullSystem()
		return
	}
	runTester()
}

func runFullSystem() {
	fmt.Println("Starting elevator system...")

	// ------------------------------------------------
	// Network init — starts first to read node ID
	// ------------------------------------------------
	nw.NetworkInit()
	localID := nw.GetIp()
	if !localID.IsValid() {
		panic("runSystem: Invalid local ID - network unavailable")
	}
	fmt.Println("Local ID:", localID)

	// ------------------------------------------------
	// OrderManager init
	// ------------------------------------------------
	om.OrderManagerInit(localID, [def.NumFloors]def.OrderState{})

	// ------------------------------------------------
	// Channel declarations
	// Naming convention: <sender>To<Receiver>Ch
	// ------------------------------------------------

	// Network
	peerUpdateCh := make(chan peers.PeerUpdate, 10)
	omToNetworkWvCh := make(chan def.Worldview, 10) // OM -> Network
	localWvCh := make(chan def.Worldview, 10)       // Network bridge
	networkToOmWvCh := make(chan def.Worldview, 10) // Network -> OM

	// OrderManager -> OrderAssigner
	omToOraCh := make(chan def.Worldview, 10) // Worldview trigger

	// OrderAssigner -> FSM
	oaToFsmCh := make(chan def.AssignedOrders, 10) // Cost function output

	// FSM -> OrderManager
	fsmNewOrderCh := make(chan def.NewOrderMessage, 10) // New button calls
	fsmCompleteCh := make(chan def.OrderMessage, 10)    // Completed orders
	fsmElevStateCh := make(chan def.Elevator, 10)       // Current elevator state
	malfunctionCh := make(chan bool, 10)                // Malfunction toggles

	// ------------------------------------------------
	// Goroutines
	// ------------------------------------------------

	// Start network
	go nw.NetworkRun(localWvCh, networkToOmWvCh, peerUpdateCh)

	// Keep peer list current
	go func() {
		for update := range peerUpdateCh {
			nw.UpdateAliveList(update)
		}
	}()

	// Bridge: forward OM outbound worldviews to the network transmitter
	// Fix: was incorrectly reading networkToOmWvCh — must read omToNetworkWvCh
	go func() {
		for wv := range omToNetworkWvCh {
			localWvCh <- wv
		}
	}()

	// OrderManager: single owner of worldview state
	go om.UpdaterRun(
		networkToOmWvCh, // peerWvCh
		fsmCompleteCh,   // orderCompleteCh
		fsmNewOrderCh,   // newOrderCh
		omToNetworkWvCh, // networkWvCh
		omToOraCh,       // orderHandlerWvCh
		nw.GetAliveList, // getAliveList
		fsmElevStateCh,  // fsmElevStateCh
		malfunctionCh,   // malfunctionCh
	)

	// OrderAssigner: runs cost function on acknowledged worldview,
	// produces per-elevator order assignment for FSM
	go oa.Run(
		omToOraCh,
		oaToFsmCh,
		localID,
	)

	// FSM: drives hardware, reports new orders and completions to OM
	go fsm.Run(
		oaToFsmCh,
		fsmNewOrderCh,
		fsmCompleteCh,
		fsmElevStateCh,
		malfunctionCh,
		localID,
	)

	fmt.Println("System running.")
	select {}
}

func runTester() {
	fmt.Println("Starting tester...")
	switch os.Args[1] {
	case "nw":
		tester.RunNwTest()
	case "om":
		tester.RunOMTest()
	case "run":
		tester.RunTest()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println("Usage: ./main [test]")
	fmt.Println("  (no arg)  — run full elevator system")
	fmt.Println("  nw        — network test (single machine)")
	fmt.Println("  om        — order manager test (single machine, hardcoded)")
	fmt.Println("  run       — combined network + updater test (multi machine)")
}
