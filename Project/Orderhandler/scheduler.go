package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// kj


//get elev state and make input type for HRA
type HRAInput struct {
    HallRequests [][2]bool `json:"hallRequests"`
    States map[string]HRAElevState `json:"states"`
}
    
// check for correct os and set executable name
hraExecutable := ""
    switch runtime.GOOS {
        case "linux":   hraExecutable  = "hall_request_assigner"
        case "windows": hraExecutable  = "hall_request_assigner.exe"
        default:        panic("OS not supported")
	}

	// run cost function


	ret, err := exec.Command("../hall_request_assigner/"+hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
    if err != nil {
        fmt.Println("exec.Command error: ", err)
        fmt.Println(string(ret))
        return
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


