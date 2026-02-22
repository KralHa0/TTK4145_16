package tester

import (
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
	om "github.com/KralHa0/TTK4145_16/Project/OrderManager"
)

func RunOMTest() {
	fmt.Println("=== OrderManager Test ===")

	localID := nw.CheckIP()
	fmt.Println("Local ID:", localID)

	om.OrderManagerInit(localID, [def.NumFloors]def.OrderState{})

	peerWvCh := make(chan def.Worldview)
	smUpdateCh := make(chan om.SMUpdate)
	networkWvCh := make(chan def.Worldview, 10)
	orderHandlerWvCh := make(chan def.Worldview, 10)

	peerID := "192.168.1.100"
	aliveList := &def.AliveList{
		Peers: map[string]bool{
			localID: true,
			peerID:  true,
		},
	}

	go om.UpdaterRun(peerWvCh, smUpdateCh, networkWvCh, orderHandlerWvCh, aliveList)
	go listenOrderHandler(orderHandlerWvCh)
	go drainNetwork(networkWvCh)
	go simulatePeerWorldviews(peerWvCh, peerID)
	go simulateSMCompletions(smUpdateCh)
	go printFinalWorldview()

	select {}
}

func listenOrderHandler(orderHandlerWvCh <-chan def.Worldview) {
	for wv := range orderHandlerWvCh {
		fmt.Println("\n[ORDER HANDLER] Triggered — requests reached Acknowledged:")
		fmt.Printf("  HallRequests:\n")
		for floor := 0; floor < def.NumFloors; floor++ {
			fmt.Printf("    Floor %d: up=%d down=%d\n", floor,
				wv.HallRequests[floor][1],
				wv.HallRequests[floor][0],
			)
		}
		for _, node := range wv.Nodes {
			fmt.Printf("  Node %s CabRequests: %v\n", node.ID, node.CabRequests)
		}
	}
}

// Drains network channel to prevent blocking the ticker
func drainNetwork(networkWvCh <-chan def.Worldview) {
	for range networkWvCh {
	}
}

func simulatePeerWorldviews(peerWvCh chan<- def.Worldview, peerID string) {
	time.Sleep(1 * time.Second)

	// Worldview 1: peer reports Exist on multiple hall and cab calls
	// Hall: floor 0 up, floor 2 up, floor 3 down
	// Cab: floor 1, floor 3
	fmt.Println("\n[PEER SIM] Worldview 1 — peer reports Exist on multiple calls")
	wv1 := def.Worldview{
		Nodes: []def.Node{
			{
				ID: peerID,
				CabRequests: [def.NumFloors]def.OrderState{
					def.NoCall, def.Exist, def.NoCall, def.Exist,
				},
				ElevState: def.ElevState{Floor: 0},
			},
		},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.Exist}, // floor 0: down=NoCall, up=Exist
			{def.NoCall, def.NoCall},
			{def.NoCall, def.Exist}, // floor 2: up=Exist
			{def.Exist, def.NoCall}, // floor 3: down=Exist
		},
	}
	peerWvCh <- wv1
	time.Sleep(2 * time.Second)

	// Worldview 2: peer sends Exist again on same calls
	// Since both nodes are now at Exist, this should push all of them to Acknowledged
	// and trigger the order handler
	fmt.Println("[PEER SIM] Worldview 2 — peer confirms Exist on same calls (expect Acknowledged + orderhandler trigger)")
	wv2 := def.Worldview{
		Nodes: []def.Node{
			{
				ID: peerID,
				CabRequests: [def.NumFloors]def.OrderState{
					def.NoCall, def.Exist, def.NoCall, def.Exist,
				},
				ElevState: def.ElevState{Floor: 0},
			},
		},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.Exist},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.Exist},
			{def.Exist, def.NoCall},
		},
	}
	peerWvCh <- wv2
	time.Sleep(2 * time.Second)

	// Worldview 3: peer reports Complete on all calls after SM signals completion
	// Should push all to Complete, then reset to NoCall
	fmt.Println("[PEER SIM] Worldview 3 — peer reports Complete on all calls (expect reset to NoCall)")
	wv3 := def.Worldview{
		Nodes: []def.Node{
			{
				ID: peerID,
				CabRequests: [def.NumFloors]def.OrderState{
					def.NoCall, def.Complete, def.NoCall, def.Complete,
				},
				ElevState: def.ElevState{Floor: 2},
			},
		},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.Complete},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.Complete},
			{def.Complete, def.NoCall},
		},
	}
	peerWvCh <- wv3
}

func simulateSMCompletions(smUpdateCh chan<- om.SMUpdate) {
	// Wait until calls are at Acknowledged, then signal completions
	time.Sleep(4 * time.Second)
	fmt.Println("\n[SM SIM] Signaling completion at floor 0")
	smUpdateCh <- om.SMUpdate{Floor: 0, Direction: 1}

	time.Sleep(500 * time.Millisecond)
	fmt.Println("[SM SIM] Signaling completion at floor 1")
	smUpdateCh <- om.SMUpdate{Floor: 1, Direction: 0}

	time.Sleep(500 * time.Millisecond)
	fmt.Println("[SM SIM] Signaling completion at floor 2")
	smUpdateCh <- om.SMUpdate{Floor: 2, Direction: 1}

	time.Sleep(500 * time.Millisecond)
	fmt.Println("[SM SIM] Signaling completion at floor 3")
	smUpdateCh <- om.SMUpdate{Floor: 3, Direction: 0}
}
func printFinalWorldview() {
	// Wait for the full test sequence to complete
	time.Sleep(12 * time.Second)
	wv := om.GetLocalWv()
	fmt.Println("\n=== Final Local Worldview ===")
	fmt.Println("HallRequests:")
	for floor := 0; floor < def.NumFloors; floor++ {
		fmt.Printf("  Floor %d: up=%d down=%d\n", floor,
			wv.HallRequests[floor][1],
			wv.HallRequests[floor][0],
		)
	}
	for _, node := range wv.Nodes {
		fmt.Printf("Node %s:\n", node.ID)
		fmt.Printf("  CabRequests: %v\n", node.CabRequests)
		fmt.Printf("  ElevState: Floor=%d Behavior=%d Malfunctioned=%v\n",
			node.ElevState.Floor,
			node.ElevState.Behavior,
			node.ElevState.Malfunctioned,
		)
	}
}
