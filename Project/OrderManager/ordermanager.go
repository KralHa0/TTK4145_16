package ordermanager

import (
	hw "Driver-go/elevio"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	_ "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
)

//Motordirectoion Up = 1, Down = -1, idle = 0

var localWv def.Worldview

func OrderManagerInit(localNodeID string, initialCabRequests [def.NumFloors]def.OrderState) {
	localWv = def.Worldview{
		Nodes: []def.Node{
			{
				ID:          localNodeID,
				CabRequests: initialCabRequests,
				Elevator:    def.Elevator{},
			},
		},
		HallRequests: [def.NumFloors][2]def.OrderState{},
	}
}

func UpdaterRun(
	peerWvCh <-chan def.Worldview,
	orderMsgCh <-chan def.OrderMessage,
	networkWvCh chan<- def.Worldview,
	orderHandlerWvCh chan<- def.Worldview,
	aliveList *def.AliveList,
) {
	ticker := time.NewTicker(def.MsgFrq)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			networkWvCh <- localWv

		case peerWv := <-peerWvCh:
			reachedAck := mergePeerWorldview(peerWv, aliveList)
			if reachedAck {
				orderHandlerWvCh <- localWv
			}

		case sm := <-orderMsgCh:
			applyCompletion(sm.Floor)
		}
	}
}

func mergePeerWorldview(peerWv def.Worldview, aliveList *def.AliveList) bool {
	reachedAck := false

	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {
			order := def.OrderMessage{Floor: floor, Direction: ToMotorDirection(dir)}
			local := localWv.HallRequests[floor][dir]
			newState := mergeHallState(local, peerWv.HallRequests[floor][dir], order, aliveList)
			if newState != local {
				localWv.HallRequests[floor][dir] = newState
				if newState == def.Acknowledged {
					reachedAck = true
				}
			}
		}
	}

	localID := localWv.Nodes[0].ID
	for _, peerNode := range peerWv.Nodes {
		if peerNode.ID == localID {
			continue
		}
		found := false
		for i, node := range localWv.Nodes {
			if node.ID == peerNode.ID {
				for floor := 0; floor < def.NumFloors; floor++ {
					local := localWv.Nodes[i].CabRequests[floor]
					newState := mergeCabState(local, peerNode.CabRequests[floor])
					if newState != local {
						localWv.Nodes[i].CabRequests[floor] = newState
						if newState == def.Acknowledged {
							reachedAck = true
						}
					}
				}
				localWv.Nodes[i].Elevator = peerNode.Elevator
				found = true
				break
			}
		}
		if !found {
			localWv.Nodes = append(localWv.Nodes, peerNode)
		}
	}

	return reachedAck
}

func GetLocalWv() def.Worldview {
	return localWv
}

func mergeHallState(local, peer def.OrderState, ordermsg def.OrderMessage, aliveList *def.AliveList) def.OrderState {
	if local == def.NoCall && peer >= def.Exist {
		return def.Exist
	}
	if local == def.Exist {
		if allAliveHallAtOrAbove(ordermsg, def.Exist, aliveList) {
			return def.Acknowledged
		}
	}
	if local == def.Acknowledged && peer == def.Complete {
		return def.Complete
	}
	if local == def.Complete {
		if allAliveHallAtOrAbove(ordermsg, def.Complete, aliveList) || peer == def.NoCall {
			return def.NoCall
		}
	}
	return local
}

func mergeCabState(local, peer def.OrderState) def.OrderState {
	if local == def.NoCall && peer >= def.Exist {
		return def.Exist
	}
	if local == def.Acknowledged && peer == def.Complete {
		return def.Complete
	}
	if local == def.Complete && peer == def.NoCall {
		return def.NoCall
	}
	return local
}

func applyCompletion(floor int) {
	for dir := 0; dir < 2; dir++ {
		if localWv.HallRequests[floor][dir] == def.Acknowledged {
			localWv.HallRequests[floor][dir] = def.Complete
		}
	}
	if localWv.Nodes[0].CabRequests[floor] == def.Acknowledged {
		localWv.Nodes[0].CabRequests[floor] = def.Complete
	}
}

// allAliveHallAtOrAbove checks if all alive nodes have reported the hall cell at or above threshold.
// Since hall requests are global, we use localWv.HallRequests which reflects the merged state.
func allAliveHallAtOrAbove(ordermsg def.OrderMessage, threshold def.OrderState, aliveList *def.AliveList) bool {
	for _, alive := range aliveList.Peers {
		if !alive {
			continue
		}
		if localWv.HallRequests[ordermsg.Floor][ToIntHelper(ordermsg.Direction)] < threshold {
			return false
		}
	}
	return true
}

// Helpers
func SetHallCall(ordermsg def.OrderMessage, state def.OrderState) {
	localWv.HallRequests[ordermsg.Floor][ToIntHelper(ordermsg.Direction)] = state
}

func SetCabCall(ordermsg def.OrderMessage, state def.OrderState) {
	localWv.Nodes[0].CabRequests[ordermsg.Floor] = state
}

func ToMotorDirection(i int) hw.MotorDirection {
	if i == 0 {
		return hw.MD_Up
	} else {
		return hw.MD_Down
	}
}

func ToIntHelper(md hw.MotorDirection) int {
	if md == hw.MD_Up {
		return 0
	} else {
		return 1
	}
}
