package tester

import (
	hw "Driver-go/elevio"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw  "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
	om  "github.com/KralHa0/TTK4145_16/Project/OrderManager"
)

func RunOMTest() {
	fmt.Println("=== OrderManager Test ===")

	nw.NetworkInit()
	localID := nw.GetIp()
	fmt.Println("Local ID:", localID)

	om.OrderManagerInit(localID, [def.NumFloors]def.OrderState{})

	peerWvCh         := make(chan def.Worldview, 10)
	orderCompleteCh  := make(chan def.OrderMessage, 10)
	newOrderCh       := make(chan def.NewOrderMessage, 10)
	networkWvCh      := make(chan def.Worldview, 10)
	orderHandlerWvCh := make(chan def.Worldview, 10)

	peerID := "192.168.1.100"

	aliveList := def.AliveList{
		Peers: map[string]bool{
			localID: true,
			peerID:  true,
		},
	}
	getAliveList := func() def.AliveList {
		copyMap := make(map[string]bool)
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
		getAliveList,
	)

	go listenOrderHandler(orderHandlerWvCh)
	go drainNetwork(networkWvCh)
	go simulateNewOrders(newOrderCh)
	go simulatePeerWorldviews(peerWvCh, peerID)
	go simulateSMCompletions(orderCompleteCh)
	go printFinalWorldview()

	select {}
}

func simulateNewOrders(newOrderCh chan<- def.NewOrderMessage) {
	time.Sleep(500 * time.Millisecond)

	orders := []def.NewOrderMessage{
		{Floor: 0, Direction: hw.MD_Down, CallType: def.Hallcall},
		{Floor: 2, Direction: hw.MD_Up,   CallType: def.Hallcall},
		{Floor: 3, Direction: hw.MD_Up,   CallType: def.Hallcall},
		{Floor: 1, Direction: hw.MD_Stop, CallType: def.Cabcall},
	}

	for _, order := range orders {
		fmt.Printf("[NEW ORDER SIM] floor=%d dir=%v type=%v\n",
			order.Floor, order.Direction, order.CallType)
		newOrderCh <- order
		time.Sleep(100 * time.Millisecond)
	}
}

func simulatePeerWorldviews(peerWvCh chan<- def.Worldview, peerID string) {
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
			{def.NoCall, def.Exist}, // floor 0: index 1 = down
			{def.NoCall, def.NoCall},
			{def.Exist,  def.NoCall}, // floor 2: index 0 = up
			{def.Exist,  def.NoCall}, // floor 3: index 0 = up
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
		{Floor: 0, Direction: hw.MD_Down},
		{Floor: 1, Direction: hw.MD_Up},  // cab — direction ignored by OM
		{Floor: 2, Direction: hw.MD_Up},
		{Floor: 3, Direction: hw.MD_Up},
	}

	for _, msg := range completions {
		fmt.Printf("\n[SM SIM] Completion at floor %d dir %v\n", msg.Floor, msg.Direction)
		orderCompleteCh <- msg
		time.Sleep(500 * time.Millisecond)
	}
}

func printFinalWorldview() {
	time.Sleep(12 * time.Second)
	fmt.Println("\n=== Final Local Worldview ===")
	printWorldview(om.GetLocalWv())
}