package ordermanager

import (
	"time"
	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	_ "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
)

type SMUpdate struct {
	Floor     int
	Direction int // 0 = down, 1 = up
}

var localWv def.Worldview

func OrderManagerInit(localNodeID string, initialCabRequests [def.NumFloors]def.OrderState) {
	localWv = def.Worldview{
		Nodes: []def.Node{
			{
				ID:          localNodeID,
				CabRequests: initialCabRequests,
				ElevState:   def.ElevState{},
			},
		},
		HallRequests: [def.NumFloors][2]def.OrderState{},
	}
}

func UpdaterRun(
	peerWvCh <-chan def.Worldview,
	smUpdateCh <-chan SMUpdate,
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

		case sm := <-smUpdateCh:
			applyCompletion(sm.Floor)
		}
	}
}

func mergePeerWorldview(peerWv def.Worldview, aliveList *def.AliveList) bool {
	reachedAck := false

	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {
			local := localWv.HallRequests[floor][dir]
			newState := mergeHallState(local, peerWv.HallRequests[floor][dir], floor, dir, aliveList)
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
				localWv.Nodes[i].ElevState = peerNode.ElevState
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

func mergeHallState(local, peer def.OrderState, floor, dir int, aliveList *def.AliveList) def.OrderState {
	if local == def.NoCall && peer >= def.Exist {
		return def.Exist
	}
	if local == def.Exist {
		if allAliveHallAtOrAbove(floor, dir, def.Exist, aliveList) {
			return def.Acknowledged
		}
	}
	if local == def.Acknowledged && peer == def.Complete {
		return def.Complete
	}
	if local == def.Complete {
		if allAliveHallAtOrAbove(floor, dir, def.Complete, aliveList) || peer == def.NoCall {
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
func allAliveHallAtOrAbove(floor, dir int, threshold def.OrderState, aliveList *def.AliveList) bool {
	for _, alive := range aliveList.Peers {
		if !alive {
			continue
		}
		if localWv.HallRequests[floor][dir] < threshold {
			return false
		}
	}
	return true
}

//Helpers
func SetHallCall(floor, dir int, state def.OrderState) {
	localWv.HallRequests[floor][dir] = state
}

func SetCabCall(floor int, state def.OrderState) {
	localWv.Nodes[0].CabRequests[floor] = state
}