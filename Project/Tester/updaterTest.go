package tester

import (
	"Driver-go/elevio"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw  "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
	om  "github.com/KralHa0/TTK4145_16/Project/OrderManager"
)

func RunOMTest() {
	fmt.Println("=== OrderManager Test ===")

	nw.NetworkInit()
	localID := nw.GetIp() // now def.NodeID
	fmt.Println("Local ID:", localID)

	om.OrderManagerInit(localID, [def.NumFloors]def.OrderState{})

	peerWvCh         := make(chan def.Worldview, 10)
	orderCompleteCh  := make(chan def.OrderMessage, 10)
	newOrderCh       := make(chan elevio.ButtonEvent, 10)
	networkWvCh      := make(chan def.Worldview, 10)
	orderHandlerWvCh := make(chan def.Worldview, 10)
	omToFsmWvCh      := make(chan def.Worldview, 10)
	fsmElevStateCh   := make(chan def.Elevator, 10)
    malfunctionCh	 := make(chan bool, 10)

	peerID := def.NodeID("192.168.1.100")

	aliveList := def.AliveList{
		Peers: map[def.NodeID]bool{
			localID: true,
			peerID:  true,
		},
	}
	getAliveList := func() def.AliveList {
		copyMap := make(map[def.NodeID]bool)
		for k, v := range aliveList.Peers {
			copyMap[k] = v
		}
		return def.AliveList{Peers: copyMap}
	}

	go om.UpdaterRun(
		peerWvCh,
		orderCompleteCh,
		newOrderCh,
		networkWvCh,
		orderHandlerWvCh,
		omToFsmWvCh,
		getAliveList,
		fsmElevStateCh,
		malfunctionCh,
	)
	go drainNetwork(omToFsmWvCh)

	go listenOrderHandler(orderHandlerWvCh)
	go drainNetwork(networkWvCh)
	go simulateNewOrders(newOrderCh)
	go simulatePeerWorldviews(peerWvCh, peerID)
	go simulateSMCompletions(orderCompleteCh)
	go printFinalWorldview()

	select {}
}

func simulateNewOrders(newOrderCh chan<- elevio.ButtonEvent) {
	time.Sleep(500 * time.Millisecond)

	orders := []elevio.ButtonEvent{
		{Floor: 0, Button: elevio.BT_HallDown},
		{Floor: 2, Button: elevio.BT_HallUp},
		{Floor: 3, Button: elevio.BT_HallUp},
		{Floor: 1, Button: elevio.BT_Cab},
	}

	for _, order := range orders {
		fmt.Printf("[NEW ORDER SIM] floor=%d button=%v\n", order.Floor, order.Button)
		newOrderCh <- order
		time.Sleep(100 * time.Millisecond)
	}
}

func simulatePeerWorldviews(peerWvCh chan<- def.Worldview, peerID def.NodeID) {
	time.Sleep(1 * time.Second)
	fmt.Println("\n[PEER SIM] Worldview 1 — peer has no knowledge yet")
	peerWvCh <- def.Worldview{
		Nodes: []def.Node{{
			ID:          peerID,
			CabRequests: [def.NumFloors]def.OrderState{},
			Elevator:    def.Elevator{CurrentFloor: 0},
		}},
		HallRequests: [def.NumFloors][2]def.OrderState{},
	}

	time.Sleep(2 * time.Second)
	fmt.Println("[PEER SIM] Worldview 2 — peer confirms Exist (expect Acknowledged + orderHandler trigger)")
	peerWvCh <- def.Worldview{
		Nodes: []def.Node{{
			ID:          peerID,
			CabRequests: [def.NumFloors]def.OrderState{},
			Elevator:    def.Elevator{CurrentFloor: 0},
		}},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.Exist}, // floor 0: DirDown = index 1
			{def.NoCall, def.NoCall},
			{def.Exist,  def.NoCall}, // floor 2: DirUp = index 0
			{def.Exist,  def.NoCall}, // floor 3: DirUp = index 0
		},
	}

	time.Sleep(2 * time.Second)
	fmt.Println("[PEER SIM] Worldview 3 — peer reports Complete (expect NoCall reset)")
	peerWvCh <- def.Worldview{
		Nodes: []def.Node{{
			ID:          peerID,
			CabRequests: [def.NumFloors]def.OrderState{},
			Elevator:    def.Elevator{CurrentFloor: 2},
		}},
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.Complete},
			{def.NoCall, def.NoCall},
			{def.Complete, def.NoCall},
			{def.Complete, def.NoCall},
		},
	}
}

func simulateSMCompletions(orderCompleteCh chan<- def.OrderMessage) {
	time.Sleep(4 * time.Second)

	completions := []def.OrderMessage{
		{Floor: 0, Dir: def.DirDown},
		{Floor: 1, Dir: def.DirUp},  // cab — Dir ignored by OM
		{Floor: 2, Dir: def.DirUp},
		{Floor: 3, Dir: def.DirUp},
	}

	for _, msg := range completions {
		fmt.Printf("\n[SM SIM] Completion at floor %d dir %v\n", msg.Floor, msg.Dir)
		orderCompleteCh <- msg
		time.Sleep(500 * time.Millisecond)
	}
}

func printFinalWorldview() {
	time.Sleep(12 * time.Second)
	fmt.Println("\n=== Final Local Worldview ===")
	printWorldview(om.GetLocalWv())
}