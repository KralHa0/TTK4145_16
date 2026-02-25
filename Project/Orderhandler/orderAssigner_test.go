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
					ElevState:  def.Moving,
					Direction: elevio.MD_Up,CurrentFloor:     2,
				},
				CabRequests: [def.NumFloors]def.OrderState{
					def.NoCall, def.NoCall, def.NoCall, def.Acknowledged,
				},
			},
			{
				ID: "two",
				Elevator: def.Elevator{
					ElevState:  def.Idle,
					CurrentFloor:     0,
					Direction: elevio.MD_Stop,
				},
				CabRequests: [def.NumFloors]def.OrderState{
					def.Acknowledged, def.NoCall, def.NoCall, def.NoCall,
				},
			},
		},
	}

	go runHRA(wvCh, hraOutputCh)

	wvCh <- wv

	output := <-hraOutputCh
	fmt.Println("HRA output:")
	for id, orders := range output {
		/*make acceptance test. cross reference hall/cabcalls to output*/
		fmt.Printf("  %s: %v\n", id, orders)
	}
}
