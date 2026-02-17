type Behavior struct {
	Idle      int
	Moving    int
	DoorOpen  int
	Maintenance int
}

type Worldview struct {
	nodes []node
}

type ElevState struct {
    Floor     int
    Direction Direction
    Behavior  Behavior
}

type node struct {
	id string
	cabrequest Cabrequest
	hallrequests Hallrequests
	elevstate ElevState
}	

