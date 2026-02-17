// system status;
	//incl: cabcalls for all nodes, 
	// hallcalls for all floors
	// kjøreliste for alle noder
//blå: read
//red write

// kjørelistse/ jobQue/ stopQue
	// hvor en node skal innom: 
	// 1. node 1: 2. node 2: 3. node 3:

// formating: [[up-0, down-0], [up-1, down-1], ...]
type Cabrequest struct {
	cabrequest [numFloors][2]bool
	}


type Hallrequests struct {
	hallrequests [numFloors][2]bool
	}

type node struct {
	id string
	cabrequest Cabrequest
	hallrequests Hallrequests
	elevstate ElevState
	}	


type Worldview struct {
	nodes []node
	}

type ElevState struct {
    Floor     int
    Direction Direction
    Behavior  Behavior
}


//visitqueue:

