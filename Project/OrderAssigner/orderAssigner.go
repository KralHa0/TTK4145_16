package orderassigner

import (
	"Driver-go/elevio"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

// Interface:
type IOrderAssigner interface {
	Run()
}

// Type Struct:
type OrderAssigner struct {
	wvCh       <-chan def.Worldview
	outputCh   chan<- def.AssignedOrders //TODO: make separate struct to cohesefy this/ abstraction
	executable string
	ownID      def.NodeID //implement stufff yippi
}

/*TODO:
Evaluate what vars should be public or not
Make separate struct for ORAOutpurchan
*/

type ORAElevState struct {
	ElevState   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type ORAInput struct {
	HallRequests [][2]bool               `json:"hallRequests"`
	States       map[string]ORAElevState `json:"states"`
}

// Constructor:
func NewOrderAssigner(
	ownID NodeID,
	wvCh <-chan def.Worldview,
	outputCh chan<- map[string][][2]bool,
) *OrderAssigner {
	exe := ""
	switch runtime.GOOS {
	case "linux":
		exe = "hall_request_assigner"
	case "windows":
		exe = "hall_request_assigner.exe"
	default:
		panic("OS not supported")
	}
	return &OrderAssigner{
		wvCh:       wvCh,
		outputCh:   outputCh,
		executable: exe,
		ownID:      ownID,
	}
}

/*
Public met: Run the cost function, is called once to initilize

	TODO:
	- add timeout
	- panic recovery

	- buffered channels for begge, slik at den ikke blokkerer hvis den får flere worldviews før den er ferdig med å kjøre kostfunksjonen.
	- pass på at du leser og tømmer siste sending i wvCh buffer. Hvis du får flere worldviews før du er ferdig med å kjøre kostfunksjonen, vil du bare tømme bufferet og bruke den siste worldviewen som input til kostfunksjonen.

	for wv := range o.wvCh {
    // drain to get latest
    for len(o.wvCh) > 0 {
        wv = <-o.wvCh
    }
    // ... process wv
}

	- add input validation 
		- check that wv is not empty, and that make ORAstateMap is not empty

	*/
func (o *OrderAssigner) Run() {
	for wv := range o.wvCh {
		fmt.Println("Received new worldview, running cost function...")

		input := o.worldviewToORAInput(wv)

		jsonBytes, err := makeExecutableInput(input)
		if err != nil {
			fmt.Println("Error marshaling input: ", err)
			continue
		}

		fmt.Println("JSON being sent to executable:")
		fmt.Println(string(jsonBytes))

		costFuncResult, err := o.runORAExecutable(jsonBytes)
		if err != nil {
			fmt.Println("Error running ORA executable: ", err)
			continue
		}

		output, err := makeResult(costFuncResult)
		if err != nil {
			fmt.Println("Error unmarshaling output: ", err)
			continue
		}

		insertCabCallsIntoOutput(output[string(o.ownID)], wv)

		fmt.Println("No errors during execution")
		//fmt.Println(output)
		o.outputCh <- output //[o.ownID] // fix type stuff
	}
}

///////Called directly from Run/////////

// TODO: rename func to fit
func (o *OrderAssigner) worldviewToORAInput(w def.Worldview) ORAInput {
	input := ORAInput{
		HallRequests: hallrequestToBool(w.HallRequests),
		States:       makeORAStateMap(w.Nodes),
	}
	return input
}

// rename to makeExecutableInput
func makeExecutableInput(input ORAInput) ([]byte, error) {
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal error: %w", err)
	}
	return jsonBytes, nil
}

func (o *OrderAssigner) runORAExecutable(jsonBytes []byte) ([]byte, error) {
	// use o.executable instead of re-detecting OS each call
	ret, err := exec.Command("../OrderAssigner/"+o.executable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec.Command error: %w, output: %s", err, string(ret))
	}
	return ret, nil
}

// rename to better name
func makeResult(ret []byte) (map[string][][2]bool, error) {
	outputMap := new(map[string][][2]bool)
	err := json.Unmarshal(ret, outputMap)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal error: %w", err)
	}
	return *outputMap, nil
}

// make subfunc of unmarshal thing
func insertCabCallsIntoOutput(myOutput def.AssignedOrders, wv def.Worldview){

	
}


/*
func makeResult(ret []byte) (map[string][][2]bool, error) {
	output := new(map[string][][2]bool)
	err := json.Unmarshal(ret, output)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal error: %w", err)
	}
	return *output, nil
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
}*/

/////called from subfuncs of Run /////////

// convert hallreq Type to executable bool list
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

// convert node type to executable Elevstate
func makeORAStateMap(nodes []def.Node) map[string]ORAElevState {
	States := make(map[string]ORAElevState)
	for _, node := range nodes {
		cabRequestBools := make([]bool, def.NumFloors)
		for floor := 0; floor < def.NumFloors; floor++ {
			if node.CabRequests[floor] == def.Acknowledged {
				cabRequestBools[floor] = true
			} else {
				cabRequestBools[floor] = false
			}
		}

		States[string(node.ID)] = ORAElevState{
			ElevState:   stateToString(node.Elevator.ElevState),
			Floor:       node.Elevator.CurrentFloor,
			Direction:   directionToString(node.Elevator.Direction),
			CabRequests: cabRequestBools,
		}
	}

	return States
} /*TODO: make acc test */

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

////////Called from subfunctions//////////////

/* TODO:
		Restructure output
			send only own outputlist
		Make init funtion
			take in own ID
		restructure insertIntoOutput
			iterate only over own Id key
		Make code pretty
			Restructure as ADT with interface and the like

		Fault tolerance
			acc test everywhere
			Implement buffered channels


TODO: Generalize toBool functionality for both hall and cab calls*/

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

/* TODO: make a general func to assign true or false for different depth array

func assignTrueOrders(inputList //generic array of numfloors x 1 or 2 cols )  {
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

/* ønsket nytt output:
[[up-0, down-0],
  [up-1, down-1],
  [...],
  [up-N, down-N]]

dvs: send bare valuelista tilhørende own ID
*/
