package tester

import (
	"Network-go/network/peers"
	hw "Driver-go/elevio"
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
	localID := nw.GetIp()
	fmt.Println("Local ID:", localID)

	om.OrderManagerInit(localID, [def.NumFloors]def.OrderState{})

	// --- Channels ---
	peerWvCh         := make(chan def.Worldview, 10)
	orderCompleteCh  := make(chan def.OrderMessage, 10)    // FSM -> OM: completed orders
	newOrderCh       := make(chan def.NewOrderMessage, 10) // FSM -> OM: new button presses
	networkWvCh      := make(chan def.Worldview, 10)
	orderHandlerWvCh := make(chan def.Worldview, 10)
	peerUpdateCh     := make(chan peers.PeerUpdate, 10)
	localWvCh        := make(chan def.Worldview, 10)

	// --- Network ---
	go nw.NetworkRun(localWvCh, peerWvCh, peerUpdateCh)

	// --- Alive list updates ---
	go func() {
		for update := range peerUpdateCh {
			nw.UpdateAliveList(update)
			printAliveList(nw.GetAliveList())
		}
	}()

	// --- OrderManager: single owner of localWv ---
	go om.UpdaterRun(
		peerWvCh,
		orderCompleteCh,
		newOrderCh,
		networkWvCh,
		orderHandlerWvCh,
		nw.GetAliveList,
	)

	// --- Forward outbound worldview to network ---
	go func() {
		for wv := range networkWvCh {
			localWvCh <- wv
		}
	}()

	go listenOrderHandler(orderHandlerWvCh)

	printControls()
	handleKeyboard(newOrderCh, orderCompleteCh)
}

// --------------------------------------------------
// Keyboard handler
// --------------------------------------------------

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

		// New hall call
		case "h":
			if len(parts) < 3 {
				fmt.Println("Usage: h <floor> <dir>")
				continue
			}
			floor := atoi(parts[1])
			dir   := atoi(parts[2])
			if !validFloor(floor) || (dir != 0 && dir != 1) {
				fmt.Println("Invalid floor or direction")
				continue
			}
			newOrderCh <- def.NewOrderMessage{
				Floor:     floor,
				Direction: indexToMotorDir(dir),
				CallType:  def.Hallcall,
			}
			fmt.Printf("[KEY] New hall call: floor %d dir %d\n", floor, dir)

		// New cab call
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
				Floor:     floor,
				Direction: hw.MD_Stop, // direction unused for cab calls inside OM
				CallType:  def.Cabcall,
			}
			fmt.Printf("[KEY] New cab call: floor %d\n", floor)

		// Signal order completion
		case "s":
			if len(parts) < 3 {
				fmt.Println("Usage: s <floor> <dir>")
				continue
			}
			floor := atoi(parts[1])
			dir   := atoi(parts[2])
			if !validFloor(floor) || (dir != 0 && dir != 1) {
				fmt.Println("Invalid floor or direction")
				continue
			}
			orderCompleteCh <- def.OrderMessage{
				Floor:     floor,
				Direction: indexToMotorDir(dir),
			}
			fmt.Printf("[KEY] Completion signaled: floor %d dir %d\n", floor, dir)

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

// --------------------------------------------------
// Utility
// --------------------------------------------------

// indexToMotorDir converts a 0/1 index to elevio.MotorDirection.
// Mirrors motorDirToIndex inside ordermanager — kept local to avoid
// exposing an internal helper through the OM package API.
func indexToMotorDir(i int) hw.MotorDirection {
	if i == 0 {
		return hw.MD_Up
	}
	return hw.MD_Down
}