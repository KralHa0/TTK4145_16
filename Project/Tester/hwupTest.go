package tester

import (
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

	localID := nw.CheckIP()
	fmt.Println("Local ID:", localID)

	nw.NetworkInit()
	om.OrderManagerInit(localID, [def.NumFloors]def.OrderState{})

	peerWvCh := make(chan def.Worldview, 10)
	smUpdateCh := make(chan om.SMUpdate)
	networkWvCh := make(chan def.Worldview, 10)
	orderHandlerWvCh := make(chan def.Worldview, 10)
	peerUpdateCh := make(chan peers.PeerUpdate, 10)
	localWvCh := make(chan def.Worldview, 10)

	go nw.NetworkRun(localWvCh, peerWvCh, peerUpdateCh)

	go func() {
		for update := range peerUpdateCh {
			nw.UpdateAliveList(update)
			printAliveList(nw.GetAliveList())
		}
	}()

	go om.UpdaterRun(peerWvCh, smUpdateCh, networkWvCh, orderHandlerWvCh, nw.GetAliveList())

	go func() {
		for wv := range networkWvCh {
			localWvCh <- wv
		}
	}()

	go listenOrderHandler(orderHandlerWvCh)

	printControls()
	handleKeyboard(smUpdateCh)
}

func printControls() {
	fmt.Println("\n--- Keyboard Controls ---")
	fmt.Println("  h <floor> <dir>  — set hall call to Exist (dir: 0=down, 1=up)")
	fmt.Println("  c <floor>        — set cab call to Exist")
	fmt.Println("  s <floor> <dir>  — signal SM completion")
	fmt.Println("  p                — print current local worldview")
	fmt.Println("  q                — quit")
	fmt.Println("-------------------------")
}

func handleKeyboard(smUpdateCh chan<- om.SMUpdate) {
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
			floor, dir := atoi(parts[1]), atoi(parts[2])
			if !validFloor(floor) || (dir != 0 && dir != 1) {
				fmt.Println("Invalid floor or direction")
				continue
			}
			om.SetHallCall(floor, dir, def.Exist)
			fmt.Printf("[KEY] Hall call set to Exist: floor %d dir %d\n", floor, dir)

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
			om.SetCabCall(floor, def.Exist)
			fmt.Printf("[KEY] Cab call set to Exist: floor %d\n", floor)

		case "s":
			if len(parts) < 3 {
				fmt.Println("Usage: s <floor> <dir>")
				continue
			}
			floor, dir := atoi(parts[1]), atoi(parts[2])
			if !validFloor(floor) || (dir != 0 && dir != 1) {
				fmt.Println("Invalid floor or direction")
				continue
			}
			smUpdateCh <- om.SMUpdate{Floor: floor, Direction: dir}
			fmt.Printf("[KEY] SM completion signaled: floor %d dir %d\n", floor, dir)

		case "p":
			printWorldview(om.GetLocalWv())

		case "q":
			fmt.Println("Quitting.")
			os.Exit(0)

		default:
			fmt.Println("Unknown command. Use h, c, s, p, or q.")
		}
	}
}
