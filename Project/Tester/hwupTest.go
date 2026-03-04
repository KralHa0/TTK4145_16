package tester

import (
	"Network-go/network/peers"
	"bufio"
	"fmt"
	"os"
	"strings"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw  "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
	om  "github.com/KralHa0/TTK4145_16/Project/OrderManager"
)

func RunTest() {
	fmt.Println("=== Combined Network + Updater Test ===")

	nw.NetworkInit()
	localID := nw.GetIp() // def.NodeID
	fmt.Println("Local ID:", localID)

	om.OrderManagerInit(localID, [def.NumFloors]def.OrderState{})

	peerWvCh         := make(chan def.Worldview, 10)
	orderCompleteCh  := make(chan def.OrderMessage, 10)
	newOrderCh       := make(chan def.NewOrderMessage, 10)
	networkWvCh      := make(chan def.Worldview, 10)
	orderHandlerWvCh := make(chan def.Worldview, 10)
	peerUpdateCh     := make(chan peers.PeerUpdate, 10)
	localWvCh        := make(chan def.Worldview, 10)
	fsmElevStateCh   := make(chan def.Elevator, 10)
    malfunctionCh	 := make(chan bool, 10)

	go nw.NetworkRun(localWvCh, peerWvCh, peerUpdateCh)

	go func() {
		for update := range peerUpdateCh {
			nw.UpdateAliveList(update)
			printAliveList(nw.GetAliveList())
		}
	}()

	go om.UpdaterRun(
		peerWvCh,
		orderCompleteCh,
		newOrderCh,
		networkWvCh,
		orderHandlerWvCh,
		nw.GetAliveList,
		fsmElevStateCh,
		malfunctionCh,
	)

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
	newOrderCh      chan<- def.NewOrderMessage,
	orderCompleteCh chan<- def.OrderMessage,
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
			dir := def.Direction(dirInt)
			newOrderCh <- def.NewOrderMessage{
				Floor:    floor,
				Dir:      dir,
				CallType: def.Hallcall,
			}
			fmt.Printf("[KEY] New hall call: floor %d dir %v\n", floor, dir)

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
			newOrderCh <- def.NewOrderMessage{
				Floor:    floor,
				Dir:      def.DirUp, // ignored for cab calls inside OM
				CallType: def.Cabcall,
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
			dir := def.Direction(dirInt)
			orderCompleteCh <- def.OrderMessage{
				Floor: floor,
				Dir:   dir,
			}
			fmt.Printf("[KEY] Completion signaled: floor %d dir %v\n", floor, dir)

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