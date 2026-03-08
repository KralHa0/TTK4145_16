package orderassigner

import (
	"fmt"
	"testing"
	"time"

	"Driver-go/elevio"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

/* to run, call:
go test -v ./Orderhandler/...
from project
Assigner tar ikke cab requests */

func TestRunORA(t *testing.T) {
	wvCh := make(chan def.Worldview, 1)
	hraOutputCh := make(chan def.AssignedOrders, 1)

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

	a := NewOrderAssigner("one", wvCh, hraOutputCh)
	go a.Run()

	wvCh <- wv

	output := <-hraOutputCh

	fmt.Println("Worldview hall requests:")
	for floor, dirs := range wv.HallRequests {
		fmt.Printf("  floor %d: up=%v down=%v\n", floor, dirs[0], dirs[1])
	}

	fmt.Println("HRA output (assigned to own node):")
	for floor, dirs := range output {
		fmt.Printf("  floor %d: up=%v down=%v\n", floor, dirs[0], dirs[1])
	}

	// --- Acceptance tests ---

	// needed tests: all cabreqs placed (correct floor is {true, true}), i want to test that all hallreqs are assigned, but i dont have acces to the full

	// --- Old Tests ---
	/*
			// 1. All elevators in worldview should have an entry in output
			for _, node := range wv.Nodes {
				if _, exists := output[string(node.ID)]; !exists {
					t.Errorf("Expected output to contain elevator ID %s, but it was missing", node.ID)
				}
			}

			// 2. Output should not contain IDs not in worldview
			nodeIDs := make(map[string]bool)
			for _, node := range wv.Nodes {
				nodeIDs[string(node.ID)] = true
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
						orders, exists := output[string(node.ID)]
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
				}
		}
			for floor := 0; floor < def.NumFloors; floor++ {
				for dir := 0; dir < 2; dir++ {
					state := wv.HallRequests[floor][dir]
					if state == def.Acknowledged {
						if !output[floor][dir] {
							t.Errorf("Hall request at floor %d btn %d was not assigned to own elevator", floor, dir)
						}
					}
				}
			}

			// 4. Cab requests with Acknowledged should be assigned to the correct elevator
			for floor := 0; floor < def.NumFloors; floor++ {
				if wv.Nodes[0].CabRequests[floor] == def.Acknowledged {
					if !output[floor][0] && !output[floor][1] {
						t.Errorf("Cab request for own elevator at floor %d not in output", floor)
					}
				}
			}

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

func TestRunHRA_PanicRecovery(t *testing.T) {
	wvCh := make(chan def.Worldview, 1)
	hraOutputCh := make(chan def.AssignedOrders, 1)

	a := NewOrderAssigner("one", wvCh, hraOutputCh)

	// closing outputCh causes a panic when Run tries to send to it
	close(hraOutputCh)

	done := make(chan struct{})
	go func() {
		a.Run()
		close(done)
	}()

	wv := def.Worldview{ /* minimal valid worldview */ }
	wvCh <- wv

	// close wvCh to let Run exit after recovering
	close(wvCh)

	select {
	case <-done:
		// passed: Run returned without crashing
	case <-time.After(5 * time.Second):
		t.Error("Run did not return after panic recovery")
	}
}
