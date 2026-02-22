package tester

import (
	"Network-go/network/peers"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	networkhandler "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
)

func RunNwTest() {
	fmt.Println("=== Network Test ===")
	id := networkhandler.CheckIP()
	fmt.Println("Local ID:", id)
	networkhandler.NetworkInit()

	localWorldviewCh := make(chan def.Worldview)
	peerWorldviewCh := make(chan def.Worldview)
	peerUpdateCh := make(chan peers.PeerUpdate)

	go networkhandler.NetworkRun(localWorldviewCh, peerWorldviewCh, peerUpdateCh)
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
		networkhandler.UpdateAliveList(update)
		printAliveList(networkhandler.GetAliveList())
	}
}

func printAliveList(aliveList *def.AliveList) {
	fmt.Println("[ALIVE] Current peer statuses:")
	for ip, alive := range aliveList.Peers {
		status := "alive"
		if !alive {
			status = "dead"
		}
		fmt.Printf("        %s -> %s\n", ip, status)
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
						def.Exist, def.NoCall, def.Acknowledged, def.NoCall, // Fixed: Available->Exist, Taken->Acknowledged
					},
					ElevState: def.ElevState{
						Floor:         floor,
						Behavior:      def.Moving,
						Malfunctioned: false,
					},
				},
			},
			HallRequests: [def.NumFloors][2]def.OrderState{
				{def.Exist, def.NoCall}, // Fixed: Available->Exist
				{def.NoCall, def.NoCall},
				{def.Acknowledged, def.NoCall}, // Fixed: Taken->Acknowledged
				{def.NoCall, def.Exist},        // Fixed: Available->Exist
			},
		}

		localWorldviewCh <- wv
		fmt.Printf("[SEND] Floor: %d | CabCalls: %v | HallCalls: %v\n",
			wv.Nodes[0].ElevState.Floor,
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
		fmt.Printf("[RECV] From: %s | Floor: %d | Behavior: %d | Malfunctioned: %v\n",
			wv.Nodes[0].ID,
			wv.Nodes[0].ElevState.Floor,
			wv.Nodes[0].ElevState.Behavior,
			wv.Nodes[0].ElevState.Malfunctioned,
		)
		fmt.Printf("       CabCalls: %v\n", wv.Nodes[0].CabRequests)
		fmt.Printf("       HallCalls: %v\n", wv.HallRequests)
	}
}

func testDisconnectReconnect() {
	fmt.Println("\n--- Test: Disconnect/Reconnect ---")
	time.Sleep(10 * time.Second)
	fmt.Println("[DISC] Disabling transmit - peers should detect loss after 500ms")
	networkhandler.DisableTransmit()
	time.Sleep(5 * time.Second)
	fmt.Println("[DISC] Re-enabling transmit - peers should detect reconnect")
	networkhandler.EnableTransmit()
}
