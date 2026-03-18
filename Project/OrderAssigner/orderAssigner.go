package orderassigner

import (
	"context"
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

				////fmt.Println("Received new worldview, running cost function...")

				
				input, err := o.worldviewToORAInput(wv)
				if err != nil {
					fmt.Println("Error building costfunction input: ", err)
					continue
				}

				if !checkORAInput(input) {
					continue
				}
				
				if len(input.States) == 0 {
					fmt.Println("OrderAssigner: skipping worldview with no nodes")
					continue
				}

				jsonBytes, err := makeExecutableInput(input)
				if err != nil {
					fmt.Println("Error marshaling input: ", err)
					continue
				}

				//fmt.Println("JSON being sent to executable:")
				//fmt.Println(string(jsonBytes))

				

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

				assigned[0][def.DirDown] = false
				assigned[def.NumFloors-1][def.DirUp] = false

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

