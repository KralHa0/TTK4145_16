package tester

import (
	"Driver-go/elevio"
	"Network-go/network/peers"
	"bufio"
	"fmt"
	"os"
	"strings"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
	om "github.com/KralHa0/TTK4145_16/Project/OrderManager"
)

func RunTest() {
	fmt.Println("=== Combined Network + Updater Test ===")

	nw.NetworkInit()
	localID := nw.GetIp() // def.NodeID
	fmt.Println("Local ID:", localID)

	om.OrderManagerInit(localID, [def.NumFloors]def.OrderState{})

	peerWvCh := make(chan def.Worldview, 10)
	orderCompleteCh := make(chan def.FsmClearOrderMessage, 10)
	newOrderCh := make(chan elevio.ButtonEvent, 10)
	networkWvCh := make(chan def.Worldview, 10)
	orderHandlerWvCh := make(chan def.Worldview, 10)
	omToFsmWvCh := make(chan def.Worldview, 10)
	peerUpdateCh := make(chan peers.PeerUpdate, 10)
	localWvCh := make(chan def.Worldview, 10)
	malfunctionCh := make(chan bool, 10)

	go nw.NetworkRun(localWvCh, peerWvCh, peerUpdateCh)
	
	go func() {
		for update := range peerUpdateCh {
			nw.UpdateAliveList(update)
			printAliveList(nw.GetAliveList())
		}
	}()

	fsmElevStateCh := make(chan def.Elevator)
	go om.UpdaterRun(
		peerWvCh,
		orderCompleteCh,
		newOrderCh,
		networkWvCh,
		orderHandlerWvCh,
		omToFsmWvCh,
		nw.GetAliveList,
		malfunctionCh,
		fsmElevStateCh,
	)
	go drainNetwork(omToFsmWvCh)

	go func() {
		for wv := range networkWvCh {
			localWvCh <- wv
		}
	}()

	go listenOrderHandler(orderHandlerWvCh)

	printControls()
	handleKeyboard(newOrderCh, orderCompleteCh)
}

func printControls() {
	fmt.Println("\n--- Keyboard Controls ---")
	fmt.Println("  h <floor> <dir>  — new hall call (dir: 0=up, 1=down)")
	fmt.Println("  c <floor>        — new cab call")
	fmt.Println("  s <floor> <dir>  — signal completion (dir: 0=up, 1=down)")
	fmt.Println("  p                — print local worldview")
	fmt.Println("  q                — quit")
	fmt.Println("-------------------------")
}

func handleKeyboard(
	newOrderCh chan<- elevio.ButtonEvent,
	orderCompleteCh chan<- def.FsmClearOrderMessage,
) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {

		case "h":
			if len(parts) < 3 {
				fmt.Println("Usage: h <floor> <dir>")
				continue
			}
			floor := atoi(parts[1])
			dirInt := atoi(parts[2])
			if !validFloor(floor) || (dirInt != 0 && dirInt != 1) {
				fmt.Println("Invalid floor or direction")
				continue
			}
			newOrderCh <- elevio.ButtonEvent{
				Floor:  floor,
				Button: elevio.ButtonType(dirInt), // 0=BT_HallUp, 1=BT_HallDown
			}
			fmt.Printf("[KEY] New hall call: floor %d dir %d\n", floor, dirInt)

		case "c":
			if len(parts) < 2 {
				fmt.Println("Usage: c <floor>")
				continue
			}
			floor := atoi(parts[1])
			if !validFloor(floor) {
				fmt.Println("Invalid floor")
				continue
			}
			newOrderCh <- elevio.ButtonEvent{
				Floor:  floor,
				Button: elevio.BT_Cab,
			}
			fmt.Printf("[KEY] New cab call: floor %d\n", floor)

		case "s":
			if len(parts) < 3 {
				fmt.Println("Usage: s <floor> <dir>")
				continue
			}
			floor := atoi(parts[1])
			dirInt := atoi(parts[2])
			if !validFloor(floor) || (dirInt != 0 && dirInt != 1) {
				fmt.Println("Invalid floor or direction")
				continue
			}
			orderCompleteCh <- def.FsmClearOrderMessage{
				Floor: floor,
				Dir:   def.DirectionUpDown(dirInt),
			}
			fmt.Printf("[KEY] Completion signaled: floor %d dir %d\n", floor, dirInt)

		case "p":
			fmt.Println("--- Local Worldview ---")
			printWorldview(om.GetLocalWv())

		case "q":
			fmt.Println("Quitting.")
			os.Exit(0)

		default:
			fmt.Println("Unknown command. Use h, c, s, p, or q.")
		}
	}
}
