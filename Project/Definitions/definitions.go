package definitions

import (
	"Driver-go/elevio"
	"time"
)

const (
	NumFloors     = 4
	NumButtons    = 3
	BetweenFloors = -1 //a floor-value used under init.
	GroundFloor   = 0
	MsgFrq        = 100 * time.Millisecond
	Addr          = "localhost"
	Port          = 15657

	DoorOpenTimeout = 3000 * time.Millisecond // door should be open for 3 seconds, then close
	WatchdogTimeout = 5000 * time.Millisecond
)

type PossibleStates int

const (
	Moving PossibleStates = iota
	Idle
	DoorOpen
)

type OrderMessage struct {
	Floor     int
	Direction elevio.MotorDirection
}

type OrderState uint8

const (
	NoCall       OrderState = 0
	Exist        OrderState = 1
	Acknowledged OrderState = 2
	Complete     OrderState = 3
)

type Elevator struct {
	CurrentFloor  int
	Direction     elevio.MotorDirection
	ElevState     PossibleStates
	Malfunctioned bool
}

type Node struct {
	ID          string
	CabRequests [NumFloors]OrderState
	Elevator    Elevator
}

type Worldview struct {
	Nodes        []Node
	HallRequests [NumFloors][2]OrderState
}

type AliveList struct {
	Peers map[string]bool //Map of peer IDs to their alive status
}
