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
	orderManagerCh := make(chan def.OrderMessage)
	networkWvCh := make(chan def.Worldview, 10)
	orderHandlerWvCh := make(chan def.Worldview, 10)

	peerID := "192.168.1.100"
	aliveList := &def.AliveList{
		Peers: map[string]bool{
			localID: true,
			peerID:  true,
		},
	}

	go om.UpdaterRun(peerWvCh, orderManagerCh, networkWvCh, orderHandlerWvCh, aliveList)
	go listenOrderHandler(orderHandlerWvCh)
	go drainNetwork(networkWvCh)
	go simulatePeerWorldviews(peerWvCh, peerID)
	go simulateSMCompletions(orderManagerCh)
	go printFinalWorldview()

	select {}
}

func simulatePeerWorldviews(peerWvCh chan<- def.Worldview, peerID string) {
	time.Sleep(1 * time.Second)

	fmt.Println("\n[PEER SIM] Worldview 1 — peer reports Exist on multiple calls")
	peerWvCh <- def.Worldview{
		Nodes: []def.Node{{
			ID:          peerID,
			CabRequests: [def.NumFloors]def.OrderState{def.NoCall, def.Exist, def.NoCall, def.NoCall},
			Elevator:    def.Elevator{CurrentFloor: 0},
		}},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.NoCall},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.NoCall},
		},
	}
	time.Sleep(2 * time.Second)

	fmt.Println("[PEER SIM] Worldview 2 — peer confirms Exist (expect Acknowledged + orderhandler trigger)")
	peerWvCh <- def.Worldview{
		Nodes: []def.Node{{
			ID:          peerID,
			CabRequests: [def.NumFloors]def.OrderState{def.NoCall, def.NoCall, def.NoCall, def.NoCall},
			Elevator:    def.Elevator{CurrentFloor: 0},
		}},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.Exist},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.Exist},
			{def.Exist, def.NoCall},
		},
	}
	time.Sleep(2 * time.Second)

	fmt.Println("[PEER SIM] Worldview 3 — peer reports Complete (expect reset to NoCall)")
	peerWvCh <- def.Worldview{
		Nodes: []def.Node{{
			ID:          peerID,
			CabRequests: [def.NumFloors]def.OrderState{def.NoCall, def.NoCall, def.NoCall, def.NoCall},
			Elevator:    def.Elevator{CurrentFloor: 2},
		}},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.NoCall},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.NoCall},
		},
	}
}

func simulateSMCompletions(orderManagerCh chan<- def.OrderMessage) {
	time.Sleep(4 * time.Second)
	for _, floor := range []int{0, 1, 2, 3} {
		fmt.Printf("\n[SM SIM] Signaling completion at floor %d\n", floor)
		orderManagerCh <- def.OrderMessage{Floor: floor, Direction: 1}
		time.Sleep(500 * time.Millisecond)
	}
}

func printFinalWorldview() {
	time.Sleep(12 * time.Second)
	fmt.Println("\n=== Final Local Worldview ===")
	printWorldview(om.GetLocalWv())
}
