package tester

import (
	"fmt"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

func listenOrderHandler(orderHandlerWvCh <-chan def.Worldview) {
	for wv := range orderHandlerWvCh {
		fmt.Println("\n[ORDER HANDLER] Triggered — requests reached Acknowledged:")
		printWorldview(wv)
	}
}

func printWorldview(wv def.Worldview) {
	fmt.Println("  HallRequests:")
	for floor := 0; floor < def.NumFloors; floor++ {
		// Fix: index 0 = up (MD_Up), index 1 = down (MD_Down)
		fmt.Printf("    Floor %d: up=%d down=%d\n", floor,
			wv.HallRequests[floor][0],
			wv.HallRequests[floor][1],
		)
	}
	for _, node := range wv.Nodes {
		fmt.Printf("  Node %s:\n", node.ID)
		fmt.Printf("    CabRequests: %v\n", node.CabRequests)
		// Fix: added %v for Direction so all 4 args match 4 format verbs
		fmt.Printf("    ElevState: Floor=%d Dir=%v Behavior=%d Malfunctioned=%v\n",
			node.Elevator.CurrentFloor,
			node.Elevator.Direction,
			node.Elevator.ElevState,
			node.Elevator.Malfunctioned,
		)
	}
}

func printAliveList(aliveList def.AliveList) {
	fmt.Println("[ALIVE] Current peer statuses:")
	for ip, alive := range aliveList.Peers {
		status := "alive"
		if !alive {
			status = "dead"
		}
		fmt.Printf("        %s -> %s\n", ip, status)
	}
}

func drainNetwork(networkWvCh <-chan def.Worldview) {
	for range networkWvCh {}
}

func atoi(s string) int {
	n := 0
	fmt.Sscanf(s, "%d", &n)
	return n
}

func validFloor(floor int) bool {
	return floor >= 0 && floor < def.NumFloors
}