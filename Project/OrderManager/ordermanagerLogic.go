package ordermanager

import def "github.com/KralHa0/TTK4145_16/Project/Definitions"

// --------------------------------------------------
// Merge logic
// --------------------------------------------------

func mergePeerWorldview(peerWv def.Worldview, aliveList def.AliveList) bool {
	reachedAck := false

	// Store latest worldview per peer for consensus checks
	if len(peerWv.Nodes) > 0 && peerWv.Nodes[0].ID.IsValid() {
		peerWorldviews[peerWv.Nodes[0].ID] = deepCopyWorldview(peerWv)
	}

	// Merge peer nodes into localWv so ORA sees all elevators
	for _, peerNode := range peerWv.Nodes {
		if !peerNode.ID.IsValid() {
			continue
		}
		if peerNode.ID == localNode().ID {
			// Recover our own cab calls from peer memory on reconnect.
			// Only apply if we have no local knowledge of that floor's state
			// (cabCallKnown is false only on fresh startup). This prevents
			// stale peer worldviews from re-activating calls we already served.
			for f := 0; f < def.NumFloors; f++ {
				if !cabCallKnown[f] && localNode().CabRequests[f] == def.NoCall {
					if peerNode.CabRequests[f] == def.Acknowledged {
						localNode().CabRequests[f] = def.Acknowledged
						cabCallKnown[f] = true
					} else if peerNode.CabRequests[f] == def.Exist {
						localNode().CabRequests[f] = def.Exist
						cabCallKnown[f] = true
					}
				}
			}
			continue
		}
		found := false
		for i := range localWv.Nodes {
			if localWv.Nodes[i].ID == peerNode.ID {
				localWv.Nodes[i].Elevator = peerNode.Elevator
				localWv.Nodes[i].CabRequests = peerNode.CabRequests
				found = true
				break
			}
		}
		if !found {
			localWv.Nodes = append(localWv.Nodes, peerNode)
		}
	}

	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {

			local := localWv.HallRequests[floor][dir]
			peer := peerWv.HallRequests[floor][dir]

			// NoCall -> Exist: propagate existence from any peer
			if local == def.NoCall && (peer == def.Exist) {
				if validTransition(local, def.Exist) {
					localWv.HallRequests[floor][dir] = def.Exist
				}
			}

			// Exist -> Acknowledged: all alive peers must confirm Exist
			if localWv.HallRequests[floor][dir] == def.Exist {
				if allAliveHallAtOrAbove(floor, dir, def.Exist, aliveList) {
					if validTransition(localWv.HallRequests[floor][dir], def.Acknowledged) {
						localWv.HallRequests[floor][dir] = def.Acknowledged
						reachedAck = true
					}
				}
			}

			// Acknowledged -> Complete: adopt completion signal from peer
			if localWv.HallRequests[floor][dir] == def.Acknowledged && peer == def.Complete {
				if validTransition(localWv.HallRequests[floor][dir], def.Complete) {
					localWv.HallRequests[floor][dir] = def.Complete
				}
			}

			// Complete -> NoCall: all alive peers confirm Complete
			if localWv.HallRequests[floor][dir] == def.Complete {
				if allAliveHallAtOrAbove(floor, dir, def.Complete, aliveList) {
					if validTransition(localWv.HallRequests[floor][dir], def.NoCall) {
						localWv.HallRequests[floor][dir] = def.NoCall
					}
				}
			}

			// Complete -> NoCall: adopt from peer that already cleared it.
			// When the first node to clear broadcasts NoCall, other nodes see
			// NoCall (0) < Complete (3) and allAliveHallAtOrAbove never passes —
			// this rule breaks that deadlock.
			if localWv.HallRequests[floor][dir] == def.Complete && peer == def.NoCall {
				localWv.HallRequests[floor][dir] = def.NoCall
			}
		}
	}

	// Cab: Exist→Acknowledged and Complete→NoCall consensus
	for f := 0; f < def.NumFloors; f++ {
		if localNode().CabRequests[f] == def.Exist {
			if allAliveCabAtOrAbove(f, def.Exist, aliveList) {
				if validTransition(localNode().CabRequests[f], def.Acknowledged) {
					localNode().CabRequests[f] = def.Acknowledged
					reachedAck = true
				}
			}
		}
		if localNode().CabRequests[f] == def.Complete {
			if allAliveCabAtOrAbove(f, def.Complete, aliveList) {
				if validTransition(localNode().CabRequests[f], def.NoCall) {
					localNode().CabRequests[f] = def.NoCall
					cabCallKnown[f] = true
				}
			}
		}
	}

	return reachedAck
}

// --------------------------------------------------
// Alive consensus helpers
// --------------------------------------------------

// allAliveCabAtOrAbove checks that every alive peer's stored worldview has the
// local node's cab request for the given floor at or above threshold.
func allAliveCabAtOrAbove(floor int, threshold def.OrderState, aliveList def.AliveList) bool {
	if len(aliveList.Peers) == 0 {
		return false
	}
	localID := localNode().ID
	for peerID, alive := range aliveList.Peers {
		if !alive {
			continue
		}
		if peerID == localID {
			if localNode().CabRequests[floor] < threshold {
				return false
			}
			continue
		}
		peerWv, known := peerWorldviews[peerID]
		if !known {
			return false
		}
		found := false
		for _, node := range peerWv.Nodes {
			if node.ID == localID {
				if node.CabRequests[floor] < threshold {
					return false
				}
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func allAliveHallAtOrAbove(
	floor int,
	dir int,
	threshold def.OrderState,
	aliveList def.AliveList,
) bool {
	if len(aliveList.Peers) == 0 {
		return false
	}
	for peerID, alive := range aliveList.Peers {
		if !alive {
			continue
		}

		if peerID == localNode().ID {
			if localWv.HallRequests[floor][dir] < threshold {
				return false
			}
			continue
		}

		peerWv, known := peerWorldviews[peerID]
		if !known {
			return false
		}
		if peerWv.HallRequests[floor][dir] < threshold {
			return false
		}
	}
	return true
}

// applyHallConsensus advances hall request states that can be resolved with
// the current alive list alone, without needing a peer worldview message.
// Handles Exist→Acknowledged and Complete→NoCall for isolated elevators.
// Caller must hold localWvMu write lock.
func applyHallConsensus(aliveList def.AliveList) bool {
	reachedAck := false
	for floor := 0; floor < def.NumFloors; floor++ {
		for dir := 0; dir < 2; dir++ {
			if localWv.HallRequests[floor][dir] == def.Exist {
				if allAliveHallAtOrAbove(floor, dir, def.Exist, aliveList) {
					if validTransition(localWv.HallRequests[floor][dir], def.Acknowledged) {
						localWv.HallRequests[floor][dir] = def.Acknowledged
						reachedAck = true
					}
				}
			}
			if localWv.HallRequests[floor][dir] == def.Complete {
				if allAliveHallAtOrAbove(floor, dir, def.Complete, aliveList) {
					if validTransition(localWv.HallRequests[floor][dir], def.NoCall) {
						localWv.HallRequests[floor][dir] = def.NoCall
					}
				}
			}
		}
	}
	// Cab: Exist→Acknowledged and Complete→NoCall consensus
	for f := 0; f < def.NumFloors; f++ {
		if localNode().CabRequests[f] == def.Exist {
			if allAliveCabAtOrAbove(f, def.Exist, aliveList) {
				if validTransition(localNode().CabRequests[f], def.Acknowledged) {
					localNode().CabRequests[f] = def.Acknowledged
					reachedAck = true
				}
			}
		}
		if localNode().CabRequests[f] == def.Complete {
			if allAliveCabAtOrAbove(f, def.Complete, aliveList) {
				if validTransition(localNode().CabRequests[f], def.NoCall) {
					localNode().CabRequests[f] = def.NoCall
					cabCallKnown[f] = true
				}
			}
		}
	}
	return reachedAck
}

// --------------------------------------------------
// Transition guard
// --------------------------------------------------

func validTransition(from, to def.OrderState) bool {
	return to == from+1 || (from == def.Complete && to == def.NoCall)
}

// --------------------------------------------------
// External safe read
// --------------------------------------------------

func GetLocalWv() def.Worldview {
	localWvMu.RLock()
	defer localWvMu.RUnlock()
	return deepCopyWorldview(localWv)
}
