package main
import (
	"os/exec"
	"fmt"
	"encoding/json"
	"runtime"
)

// Struct members must be public in order to be accessible by json.Marshal/.Unmarshal
// This means they must start with a capital letter, so we need to use field renaming struct tags to make them camelCase

type HRAElevState struct {
    Behavior    string      `json:"behaviour"`
    Floor       int         `json:"floor"` 
    Direction   string      `json:"direction"`
    CabRequests []bool      `json:"cabRequests"`
}

type HRAInput struct {
    HallRequests    [][2]bool                   `json:"hallRequests"`
    States          map[string]HRAElevState     `json:"states"`
}



func main(){

    hraExecutable := ""
    switch runtime.GOOS {
        case "linux":   hraExecutable  = "hall_request_assigner"
        case "windows": hraExecutable  = "hall_request_assigner.exe"
        default:        panic("OS not supported")
    }

    input := HRAInput{
        HallRequests: [][2]bool{{false, false}, {true, false}, {false, false}, {false, true}},
        States: map[string]HRAElevState{
            "one": HRAElevState{
                Behavior:       "moving",
                Floor:          2,
                Direction:      "up",
                CabRequests:    []bool{false, false, false, true},
            },
            "two": HRAElevState{
                Behavior:       "idle",
                Floor:          0,
                Direction:      "stop",
                CabRequests:    []bool{false, false, false, false},
            },
        },
    }

    jsonBytes, err := json.Marshal(input)
    if err != nil {
        fmt.Println("json.Marshal error: ", err)
        return
    }
    //_, filename, _, _ := runtime.Caller(0)
	//dir := filepath.Dir(filename)
    ret, err := exec.Command("../Orderhandler/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
    if err != nil {
        fmt.Println("exec.Command error: ", err)
        fmt.Println(string(ret))
        return
    }
    
    output := new(map[string][][2]bool)
    err = json.Unmarshal(ret, &output)
    if err != nil {
        fmt.Println("json.Unmarshal error: ", err)
        return
    }
        
    fmt.Printf("output: \n")
    for k, v := range *output {
        fmt.Printf("%6v :  %+v\n", k, v)
    }
}


//type HallRequest struct {
//	Floor     int
//	Direction string
//}
//
//func main() {
//	scanner := bufio.NewScanner(os.Stdin)
//	
//	fmt.Println("Hall Request Assigner")
//	fmt.Println("Enter requests (format: floor direction)")
//	fmt.Println("Example: 3 up")
//	fmt.Println("Type 'quit' to exit")
//	fmt.Println()
//
//	for {
//		fmt.Print("> ")
//		if !scanner.Scan() {
//			break
//		}
//
//		input := strings.TrimSpace(scanner.Text())
//
//		if input == "quit" {
//			fmt.Println("Exiting...")
//			break
//		}
//
//		if input == "" {
//			continue
//		}
//
//		parts := strings.Fields(input)
//		if len(parts) != 2 {
//			fmt.Println("Invalid format. Use: floor direction")
//			continue
//		}
//
//		var floor int
//		_, err := fmt.Sscanf(parts[0], "%d", &floor)
//		if err != nil {
//			fmt.Println("Floor must be a number")
//			continue
//		}
//
//		direction := strings.ToLower(parts[1])
//		if direction != "up" && direction != "down" {
//			fmt.Println("Direction must be 'up' or 'down'")
//			continue
//		}
//
//		request := HallRequest{Floor: floor, Direction: direction}
//		fmt.Printf("Assigned: Floor %d, Direction %s\n", request.Floor, request.Direction)
//	}
//}


