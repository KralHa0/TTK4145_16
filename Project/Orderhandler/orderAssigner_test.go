package Orderhandler

import (
	"fmt"
	"testing"

	"Driver-go/elevio"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

/* to run, call:
go test -v ./Orderhandler/...
from project
Assigner tar ikke cab requests */

func TestRunHRA(t *testing.T) {
	wvCh := make(chan def.Worldview, 1)
	hraOutputCh := make(chan map[string][][2]bool, 1)

	wv := def.Worldview{
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.NoCall},
			{def.Acknowledged, def.NoCall},
			{def.NoCall, def.Exist},
			{def.NoCall, def.Acknowledged},
		},
		Nodes: []def.Node{
			{
				ID: "one",
				Elevator: def.Elevator{
					ElevState: def.Moving,
					Direction: elevio.MD_Up, CurrentFloor: 2,
				},
				CabRequests: [def.NumFloors]def.OrderState{
					def.NoCall, def.NoCall, def.NoCall, def.Acknowledged,
				},
			},
			{
				ID: "two",
				Elevator: def.Elevator{
					ElevState:    def.Idle,
					CurrentFloor: 0,
					Direction:    elevio.MD_Stop,
				},
				CabRequests: [def.NumFloors]def.OrderState{
					def.Acknowledged, def.NoCall, def.NoCall, def.NoCall,
				},
			},
		},
	}

	go RunHRA(wvCh, hraOutputCh)

	wvCh <- wv

	output := <-hraOutputCh
	fmt.Println("HRA output:")
	for id, orders := range output {

		/*make acceptance test. cross reference hall/cabcalls to output*/
		fmt.Printf("  %s: %v\n", id, orders)
	}

	// --- Acceptance tests ---

	// 1. All elevators in worldview should have an entry in output
	for _, node := range wv.Nodes {
		if _, exists := output[node.ID]; !exists {
			t.Errorf("Expected output to contain elevator ID %s, but it was missing", node.ID)
		}
	}

	// 2. Output should not contain IDs not in worldview
	nodeIDs := make(map[string]bool)
	for _, node := range wv.Nodes {
		nodeIDs[node.ID] = true
	}
	for id := range output {
		if !nodeIDs[id] {
			t.Errorf("Output contains unknown elevator ID: %s", id)
		}
	}

	// 3. Hall requests with Acknowledged/Exist should be assigned to at least one elevator
	for floor := 0; floor < def.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {
			state := wv.HallRequests[floor][btn]
			if state == def.Acknowledged {
				assigned := false
				for _, orders := range output {
					if orders[floor][btn] {
						assigned = true
						break
					}
				}
				if !assigned {
					t.Errorf("Hall request at floor %d btn %d (state %v) was not assigned to any elevator", floor, btn, state)
				}
			}
		}
	}

	// 4. Cab requests with Acknowledged should be assigned to the correct elevator
	for _, node := range wv.Nodes {
		for floor := 0; floor < def.NumFloors; floor++ {
			if node.CabRequests[floor] == def.Acknowledged {
				orders, exists := output[node.ID]
				if !exists {
					t.Errorf("Elevator %s missing from output", node.ID)
					continue
				}
				if !orders[floor][0] && !orders[floor][1] {
					t.Errorf("Cab request for elevator %s at floor %d was not reflected in output", node.ID, floor)
				}
			}
		}
	}
	/*
		// 5. Hall requests with NoCall should not be assigned
		for floor := 0; floor < def.NumFloors; floor++ {
			for btn := 0; btn < 2; btn++ {
				if wv.HallRequests[floor][btn] == def.NoCall {
					for id, orders := range output {
						if orders[floor][btn] {
							t.Errorf("Hall request at floor %d btn %d is NoCall but was assigned to elevator %s", floor, btn, id)
						}
					}
				}
			}
		}*/
}
