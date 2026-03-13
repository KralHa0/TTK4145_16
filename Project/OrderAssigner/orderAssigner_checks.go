package orderassigner

import (
	"fmt"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

// checkORAInput verifies that the input struct passed to the executable
// has the correct dimensions before sending it.
//
// Invariants:
//   I1: len(HallRequests) == NumFloors  (4×2 layout)
//   I2: each elevator's CabRequests slice has length NumFloors  (4×1 layout)
func checkORAInput(input ORAInput) bool{
	// I1: hall requests row count
	if len(input.HallRequests) != def.NumFloors {
		fmt.Printf("[ACCEPTANCE FAIL] I1: HallRequests has %d rows, expected %d\n",
			len(input.HallRequests), def.NumFloors)
		return false
	}

	// I2: cab requests length per elevator
	for id, state := range input.States {
		if len(state.CabRequests) != def.NumFloors {
			fmt.Printf("[ACCEPTANCE FAIL] I2: elevator %q has %d cab request entries, expected %d\n",
				id, len(state.CabRequests), def.NumFloors)
				return false
		}
	}
	return true
}

// checkORAOutputMap verifies the full output map returned by the executable
// before the own node's slice is extracted.
//
// Invariants:
//   O1: each elevator's order slice has length NumFloors  (4×2 layout)
//   O2: every Acknowledged hall request is assigned to at least one elevator
func checkORAOutputMap(outputMap map[def.NodeID][][2]bool, wv def.Worldview) {
	// O1: output slice length per elevator
	for id, orders := range outputMap {
		if len(orders) != def.NumFloors {
			fmt.Printf("[ACCEPTANCE FAIL] O1: elevator %q output has %d rows, expected %d\n",
				id, len(orders), def.NumFloors)
		}
	}

	// O2: every Acknowledged hall call must appear in at least one elevator's output
	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {
			if wv.HallRequests[floor][dir] != def.Acknowledged {
				continue
			}
			assigned := false
			for _, orders := range outputMap {
				if len(orders) > floor && orders[floor][dir] {
					assigned = true
					break
				}
			}
			if !assigned {
				fmt.Printf("[ACCEPTANCE FAIL] O2: Acknowledged hall request floor=%d dir=%d not assigned to any elevator\n",
					floor, dir)
			}
		}
	}
}

// checkOwnOutput verifies the own node's final AssignedOrders slice after
// cab calls have been inserted.
//
// Invariant:
//   A1: for every floor where own node has an Acknowledged cab request,
//       output[floor] must be {true, true}
func checkOwnOutput(ownOutput [][2]bool, wv def.Worldview, ownID def.NodeID) {
	var ownNode *def.Node
	for i := range wv.Nodes {
		if wv.Nodes[i].ID == ownID {
			ownNode = &wv.Nodes[i]
			break
		}
	}
	if ownNode == nil {
		fmt.Printf("[ACCEPTANCE FAIL] A1: own node %q not found in worldview\n", ownID)
		return
	}

	for floor := 0; floor < def.NumFloors; floor++ {
		if ownNode.CabRequests[floor] != def.Acknowledged {
			continue
		}
		if len(ownOutput) <= floor {
			fmt.Printf("[ACCEPTANCE FAIL] A1: own output has no entry for floor %d (len=%d)\n",
				floor, len(ownOutput))
			continue
		}
		if ownOutput[floor] != [2]bool{true, true} {
			fmt.Printf("[ACCEPTANCE FAIL] A1: Acknowledged cab request at floor %d not in output (got %v)\n",
				floor, ownOutput[floor])
		}
	}
}