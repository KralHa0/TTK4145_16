package Orderhandler

import (
	"fmt"
	"testing"

	"Driver-go/elevio"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

func TestRunHRA(t *testing.T) {
	wvCh := make(chan def.Worldview, 1)
	hraOutputCh := make(chan map[string][][2]bool, 1)

	wv := def.Worldview{
		HallRequests: [def.NumFloors][2]def.OrderState{
			{def.NoCall, def.NoCall},
			{def.Acknowledged, def.NoCall},
			{def.NoCall, def.NoCall},
			{def.NoCall, def.Acknowledged},
		},
		Nodes: []def.Node{
			{
				ID: "one",
				ElevState: def.ElevState{
					Behavior:  def.Moving,
					Floor:     2,
					Direction: elevio.MD_Up,
				},
				CabRequests: [def.NumFloors]def.OrderState{
					def.NoCall, def.NoCall, def.NoCall, def.Acknowledged,
				},
			},
			{
				ID: "two",
				ElevState: def.ElevState{
					Behavior:  def.Idle,
					Floor:     0,
					Direction: elevio.MD_Stop,
				},
				CabRequests: [def.NumFloors]def.OrderState{
					def.NoCall, def.NoCall, def.NoCall, def.NoCall,
				},
			},
		},
	}

	go runHRA(wvCh, hraOutputCh)

	wvCh <- wv

	output := <-hraOutputCh
	fmt.Println("HRA output:")
	for id, orders := range output {
		fmt.Printf("  %s: %v\n", id, orders)
	}
}
