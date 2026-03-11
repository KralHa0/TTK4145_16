package orderassigner

import (
	"Driver-go/elevio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

// Interface:
type IOrderAssigner interface {
	Run()
}

// Constants
const oraTimeout = 2 * time.Second

// Type Struct:
type OrderAssigner struct {
	wvCh       <-chan def.Worldview
	outputCh   chan<- def.AssignedOrders
	executable string
	ownID      def.NodeID
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
	ownID def.NodeID,
	wvCh <-chan def.Worldview,
	outputCh chan<- def.AssignedOrders,
) *OrderAssigner {
	_, srcFile, _, _ := runtime.Caller(0)
	srcDir := filepath.Dir(srcFile)
	exe := ""
	switch runtime.GOOS {
	case "linux":
		exe = filepath.Join(srcDir, "hall_request_assigner")
	case "windows":
		exe = filepath.Join(srcDir, "hall_request_assigner.exe")
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

// start running the order assigner
func (o *OrderAssigner) Run() {
	for {
		closedNormaly := false
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("OrderAssigner: panic recovered: %v\n", r)
				}
			}()

			//main goroutine loop
			for wv := range o.wvCh {

				// Drain loop, handle only latest call
				for len(o.wvCh) > 0 {
					wv = <-o.wvCh
				}

				//fmt.Println("Received new worldview, running cost function...")

				if len(wv.Nodes) == 0 {
					fmt.Println("OrderAssigner: skipping worldview with no nodes")
					continue
				}

				input := o.worldviewToORAInput(wv)
				checkORAInput(input)

				jsonBytes, err := makeExecutableInput(input)
				if err != nil {
					fmt.Println("Error marshaling input: ", err)
					continue
				}

				//fmt.Println("JSON being sent to executable:")
				fmt.Println("\n", string(jsonBytes))

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

				checkORAOutputMap(output, wv)

				// traditional testing turn into acc test
				if _, ok := output[o.ownID]; !ok {
					fmt.Println("OrderAssigner: own ID not in output, skipping")
					continue
				}

				insertCabCallsIntoOutput(output[o.ownID], wv, o.ownID)
				checkOwnOutput(output[o.ownID], wv, o.ownID)

				//fmt.Println("No errors during execution")
				var assigned def.AssignedOrders
				copy(assigned[:], output[o.ownID])

				//send
				select {
				case o.outputCh <- assigned:
				default:
				}

			}
			closedNormaly = true // only reached if channel closed normally
		}()

		if closedNormaly {
			fmt.Println("OrderAssigner: input channel closed, stopping")
			return
		}

		fmt.Println("OrderAssigner: restarting after panic...")
	}
}

//----------
//Called directly from Run
//----------

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
	exePath := o.executable

	ctx, cancel := context.WithTimeout(context.Background(), oraTimeout)
	defer cancel()
	ret, err := exec.CommandContext(ctx, exePath, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("exec.Command error: %w, output: %s", err, string(ret))
	}
	return ret, nil
}

// rename to better name
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

/////called from subfuncs of Run /////////

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

// convert node type to executable Elevstate
func makeORAStateMap(nodes []def.Node) map[string]ORAElevState {
	States := make(map[string]ORAElevState)
	for _, node := range nodes {
		if !node.ID.IsValid() {
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
