package Orderhandler

import (
	"Driver-go/elevio"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

/*

TODO: Generalize toBool functionality for both hall and cab calls*/

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

type HRAElevState struct {
	ElevState   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]HRAElevState `json:"states"`
}

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

/*func assignTrueOrders(inputList //generic array of numfloors x 1 or 2 cols )  {
	for rows := 0; rows < len(inputList); rows++ {
		for cols := 0; cols < len(inputList[rows]); cols++ {
			if inputList[rows][cols] == def.Acknowledged {
				inputList[rows][cols] = true
			} else {
				inputList[rows][cols] = false
			}
		}
	}
}*/

/*TODO: make a general func to assign true or false for a */
func hallrequestToBool(hallRequests [def.NumFloors][2]def.OrderState) [][2]bool {
	boolRequests := make([][2]bool, def.NumFloors)
	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {
			if hallRequests[floor][dir] == def.Acknowledged {
				boolRequests[floor][dir] = true
			} else {
				boolRequests[floor][dir] = false
			}
		}
	}
	return boolRequests
} /*TODO: make acc test */

// TODO: change what is considered true/false (ONLY 2 is true all else is false)
func makeHRAStateMap(nodes []def.Node) map[string]HRAElevState {
	States := make(map[string]HRAElevState)
	for _, node := range nodes {
		cabRequestBools := make([]bool, def.NumFloors)
		for floor := 0; floor < def.NumFloors; floor++ {
			if node.CabRequests[floor] == def.Acknowledged {
				cabRequestBools[floor] = true
			} else {
				cabRequestBools[floor] = false
			}
		}

		States[node.ID] = HRAElevState{
			ElevState:   stateToString(node.Elevator.ElevState),
			Floor:       node.Elevator.CurrentFloor,
			Direction:   directionToString(node.Elevator.Direction),
			CabRequests: cabRequestBools,
		}
	}

	return States
} /*TODO: make acc test */

func worldviewToHRAInput(w def.Worldview) HRAInput {
	HRAInput := HRAInput{
		HallRequests: hallrequestToBool(w.HallRequests),
		States:       makeHRAStateMap(w.Nodes),
	}

	return HRAInput
} /*TODO: make acc test */

/*TODO: run costexe functionality*/

func marshalInput(input HRAInput) ([]byte, error) {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal error: %w", err)
	}
	return jsonBytes, nil
}

func runHRAExecutable(jsonBytes []byte) ([]byte, error) {
	hraExecutable := ""
	switch runtime.GOOS {
	case "linux":
		hraExecutable = "hall_request_assigner"
	case "windows":
		hraExecutable = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}

	ret, err := exec.Command("../Orderhandler/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec.Command error: %w, output: %s", err, string(ret))
	}
	return ret, nil
}

func insertCabCallsIntoOutput(output map[string][][2]bool, wv def.Worldview) {

	for outputId, orderList := range output { //iterate over all keys
		for _, inputNode := range wv.Nodes {
			inputNodeId := inputNode.ID
			if outputId == inputNodeId {
				for floor := range len(orderList) {
					if inputNode.CabRequests[floor] == def.Acknowledged {
						output[outputId][floor] = [2]bool{true, true}
					}
				}
			}
		}
	}
}

func unmarshalOutput(ret []byte) (map[string][][2]bool, error) {
	output := new(map[string][][2]bool)
	err := json.Unmarshal(ret, output)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal error: %w", err)
	}
	return *output, nil
}

/*make init func som bygger wvCh og hraOutputCh, slik som i test func.
send chanelnavn til ordermanager, og start runHRA i en goroutine.
- mota wvCh og hraOutputCh fra main
- go runHRA(wvCh, hraOutputCh) i init func
	-da vil runHRA kjøre i bakgrunnen og vente på nye worldviews, og sende output til ordermanager hver gang den kjører kostfunksjonen

- buffered channels for begge, slik at den ikke blokkerer hvis den får flere worldviews før den er ferdig med å kjøre kostfunksjonen.
	- pass på at du leser og tømmer siste sending i wvCh buffer. Hvis du får flere worldviews før du er ferdig med å kjøre kostfunksjonen, vil du bare tømme bufferet og bruke den siste worldviewen som input til kostfunksjonen.
*/

func runHRA(
	wvCh <-chan def.Worldview,
	hraOutputCh chan<- map[string][][2]bool,
) {
	for wv := range wvCh {
		fmt.Println("Received new worldview, running cost function...")
		input := worldviewToHRAInput(wv)
		jsonBytes, err := marshalInput(input)
		if err != nil {
			fmt.Println("Error marshaling input: ", err)
			continue
		}

		fmt.Println("JSON being sent to executable:")
		fmt.Println(string(jsonBytes))

		ret, err := runHRAExecutable(jsonBytes)
		if err != nil {
			fmt.Println("Error running HRA executable: ", err)
			continue
		}

		output, err := unmarshalOutput(ret)
		if err != nil {
			fmt.Println("Error unmarshaling output: ", err)
			continue
		}

		insertCabCallsIntoOutput(output, wv)

		fmt.Println("No errors during execution")
		//fmt.Println(output)
		hraOutputCh <- output
	}
}

/* Output format: map of key= id, value = list of orders

Ex:
id1 : [[up-0, down-0],
	  [up-1, down-1],
	  [...],
	  [up-N, down-N]]

id2 : [[up-0, down-0],
	  [up-1, down-1],
	  [...],
	  [up-N, down-N]]

id3 : [[up-0, down-0],
	  [up-1, down-1],
	  [...],
	  [up-N, down-N]]
*/
