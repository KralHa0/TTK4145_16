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

	peerWvCh         := make(chan def.Worldview)
	smUpdateCh       := make(chan om.SMUpdate)
	networkWvCh      := make(chan def.Worldview, 10)
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

func simulatePeerWorldviews(peerWvCh chan<- def.Worldview, peerID string) {
	time.Sleep(1 * time.Second)

	fmt.Println("\n[PEER SIM] Worldview 1 — peer reports Exist on multiple calls")
	peerWvCh <- def.Worldview{
		Nodes: []def.Node{{
			ID:          peerID,
			CabRequests: [def.NumFloors]def.OrderState{def.NoCall, def.Exist, def.NoCall, def.Exist},
			ElevState:   def.ElevState{Floor: 0},
		}},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.Exist},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.Exist},
			{def.Exist, def.NoCall},
		},
	}
	time.Sleep(2 * time.Second)

	fmt.Println("[PEER SIM] Worldview 2 — peer confirms Exist (expect Acknowledged + orderhandler trigger)")
	peerWvCh <- def.Worldview{
		Nodes: []def.Node{{
			ID:          peerID,
			CabRequests: [def.NumFloors]def.OrderState{def.NoCall, def.Exist, def.NoCall, def.Exist},
			ElevState:   def.ElevState{Floor: 0},
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
			CabRequests: [def.NumFloors]def.OrderState{def.NoCall, def.Complete, def.NoCall, def.Complete},
			ElevState:   def.ElevState{Floor: 2},
		}},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.Complete},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.Complete},
			{def.Complete, def.NoCall},
		},
	}
}

func simulateSMCompletions(smUpdateCh chan<- om.SMUpdate) {
	time.Sleep(4 * time.Second)
	for _, floor := range []int{0, 1, 2, 3} {
		fmt.Printf("\n[SM SIM] Signaling completion at floor %d\n", floor)
		smUpdateCh <- om.SMUpdate{Floor: floor, Direction: 1}
		time.Sleep(500 * time.Millisecond)
	}
}

func printFinalWorldview() {
	time.Sleep(12 * time.Second)
	fmt.Println("\n=== Final Local Worldview ===")
	printWorldview(om.GetLocalWv())
}