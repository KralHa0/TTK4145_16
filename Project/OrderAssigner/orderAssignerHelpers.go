package orderassigner

import (
	"Driver-go/elevio"
	"encoding/json"
	"fmt"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
)

// convert from dirtype to executabe string
func directionToString(d elevio.MotorDirection) string {
	switch d {
	case elevio.MD_Up:
		return "up"
	case elevio.MD_Down:
		return "down"
	default:
		return "stop"
	}
}

// convert fron PossibeState type to executable string
func stateToString(s def.PossibleStates) string {
	switch s {
	case def.Moving:
		return "moving"
	case def.DoorOpen:
		return "doorOpen"
	default:
		return "idle"
	}
}

func (o *OrderAssigner) worldviewToORAInput(w def.Worldview) (ORAInput, error) {
	input := ORAInput{
		HallRequests: hallrequestToBool(w.HallRequests),
		States:       makeORAStateMap(w.Nodes, o.ownID),
	}
	return input, nil
}

func makeResult(ret []byte) (map[def.NodeID][][2]bool, error) {
	var outputMap map[def.NodeID][][2]bool
	err := json.Unmarshal(ret, &outputMap)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal error: %w", err)
	}
	return outputMap, nil
}

// make subfunc of unmarshal thing
func insertCabCallsIntoOutput(myOutPut [][2]bool, wv def.Worldview, ID def.NodeID) {
	for _, node := range wv.Nodes {
		if node.ID == ID {
			for floor := 0; floor < def.NumFloors; floor++ {
				if node.CabRequests[floor] == def.Acknowledged {
					myOutPut[floor] = [2]bool{true, true}
				}
			}
			return
		}
	}
}

// convert hallreq Type to executable bool list
func hallrequestToBool(hallRequests [def.NumFloors][2]def.OrderState) [][2]bool {
	boolRequests := make([][2]bool, def.NumFloors)
	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {
			boolRequests[floor][dir] = hallRequests[floor][dir] == def.Acknowledged
		}
	}
	return boolRequests
}

func isNodeAvailable(node def.Node) bool {
	if !nw.GetAliveList().Peers[node.ID] {
		return false
	}
	if node.Elevator.Malfunctioned {

		return false
	}
	return true
}

func makeExecutableInput(input ORAInput) ([]byte, error) {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal error: %w", err)
	}
	return jsonBytes, nil
}

// convert node type to executable Elevstate
func makeORAStateMap(nodes []def.Node, ownID def.NodeID) map[string]ORAElevState {
	States := make(map[string]ORAElevState)
	for _, node := range nodes {
		if !node.ID.IsValid() {
			continue
		}
		if node.ID == ownID {
			if node.Elevator.Malfunctioned {
				continue
			}
		} else if !isNodeAvailable(node) {
			continue
		}

		cabRequestBools := make([]bool, def.NumFloors)
		for floor := 0; floor < def.NumFloors; floor++ {
			cabRequestBools[floor] = node.CabRequests[floor] == def.Acknowledged
		}

		States[string(node.ID)] = ORAElevState{
			ElevState:   stateToString(node.Elevator.ElevState),
			Floor:       node.Elevator.CurrentFloor,
			Direction:   directionToString(node.Elevator.Direction),
			CabRequests: cabRequestBools,
		}
	}

	return States
}
