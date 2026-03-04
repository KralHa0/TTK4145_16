package tester

import (
	"Driver-go/elevio"
	"Network-go/network/peers"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
)

func RunNwTest() {
	fmt.Println("=== Network Test ===")
	id := nw.GetIp()
	fmt.Println("Local ID:", id)
	nw.NetworkInit()
	localWorldviewCh := make(chan def.Worldview)
	peerWorldviewCh := make(chan def.Worldview)
	peerUpdateCh := make(chan peers.PeerUpdate)
	go nw.NetworkRun(localWorldviewCh, peerWorldviewCh, peerUpdateCh)
	go testPeerDiscovery(peerUpdateCh)
	go testSendWorldview(localWorldviewCh, id)
	go testReceiveWorldview(peerWorldviewCh, id)
	go testDisconnectReconnect()
	select {}
}

func testPeerDiscovery(peerUpdateCh <-chan peers.PeerUpdate) {
	fmt.Println("\n--- Test: Peer Discovery ---")
	for update := range peerUpdateCh {
		if update.New != "" {
			fmt.Println("[PEER] New peer detected:", update.New)
		}
		for _, lost := range update.Lost {
			fmt.Println("[PEER] Lost peer:", lost)
		}
		nw.UpdateAliveList(update)
		printAliveList(nw.GetAliveList())
	}
}

func testSendWorldview(localWorldviewCh chan<- def.Worldview, id string) {
	fmt.Println("\n--- Test: Sending Worldview ---")
	floor := 0
	for {
		wv := def.Worldview{
			Nodes: []def.Node{
				{
					ID: id,
					CabRequests: [def.NumFloors]def.OrderState{
						def.Exist, def.NoCall, def.Acknowledged, def.NoCall,
					},
					Elevator: def.Elevator{
						CurrentFloor:  floor,
						Direction:     elevio.MD_Up,
						ElevState:     def.Moving,
						Malfunctioned: false,
					},
				},
			},
			HallRequests: [def.NumFloors][2]def.OrderState{
				{def.Exist, def.NoCall},
				{def.NoCall, def.NoCall},
				{def.Acknowledged, def.NoCall},
				{def.NoCall, def.Exist},
			},
		}
		localWorldviewCh <- wv
		fmt.Printf("[SEND] Floor: %d | CabCalls: %v | HallCalls: %v\n",
			wv.Nodes[0].Elevator.CurrentFloor,
			wv.Nodes[0].CabRequests,
			wv.HallRequests,
		)
		floor = (floor + 1) % def.NumFloors
		time.Sleep(1 * time.Second)
	}
}

func testReceiveWorldview(peerWorldviewCh <-chan def.Worldview, id string) {
	fmt.Println("\n--- Test: Receiving Worldview ---")
	for wv := range peerWorldviewCh {
		if len(wv.Nodes) == 0 {
			continue
		}
		if wv.Nodes[0].ID == id {
			continue
		}
		fmt.Printf("[RECV] From: %s | Floor: %d | Direction: %d | Behavior: %d | Malfunctioned: %v\n",
			wv.Nodes[0].ID,
			wv.Nodes[0].Elevator.CurrentFloor,
			wv.Nodes[0].Elevator.Direction,
			wv.Nodes[0].Elevator.ElevState,
			wv.Nodes[0].Elevator.Malfunctioned,
		)
		fmt.Printf("       CabCalls: %v\n", wv.Nodes[0].CabRequests)
		fmt.Printf("       HallCalls: %v\n", wv.HallRequests)
	}
}

func testDisconnectReconnect() {
	fmt.Println("\n--- Test: Disconnect/Reconnect ---")
	time.Sleep(10 * time.Second)
	fmt.Println("[DISC] Disabling transmit")
	nw.DisableTransmit()
	time.Sleep(5 * time.Second)
	fmt.Println("[DISC] Re-enabling transmit")
	nw.EnableTransmit()
}
