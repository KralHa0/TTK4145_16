package definitions

import (
	"Driver-go/elevio"
	"time"
)

const (
	NumFloors       = 4
	NumButtons      = 3
	BetweenFloors   = -1
	GroundFloor     = 0
	MsgFrq          = 100 * time.Millisecond
	Addr            = "localhost"
	Port            = 15657
	DoorOpenTimeout = 3000 * time.Millisecond
	WatchdogTimeout = 5000 * time.Millisecond
)

// --------------------------------------------------
// NodeID
// --------------------------------------------------

type NodeID string

const UnknownID NodeID = ""

func (id NodeID) IsValid() bool {
	return id != UnknownID && id != "DISCONNECTED"
}

// --------------------------------------------------
// Direction
// --------------------------------------------------

type Direction int

const (
	DirUp   Direction = 0
	DirDown Direction = 1
)

// DirFromMotor converts an elevio.MotorDirection to a Direction.
// Returns (dir, true) for MD_Up and MD_Down.
// Returns (0, false) for MD_Stop — not a valid hall direction.
func DirFromMotor(d elevio.MotorDirection) (Direction, bool) {
	switch d {
	case elevio.MD_Up:
		return DirUp, true
	case elevio.MD_Down:
		return DirDown, true
	default:
		return 0, false
	}
}

// ToMotorDir converts back to elevio.MotorDirection for hardware calls.
func (d Direction) ToMotorDir() elevio.MotorDirection {
	if d == DirUp {
		return elevio.MD_Up
	}
	return elevio.MD_Down
}

type AssignedOrders [NumFloors][2]bool

// --------------------------------------------------
// CallType
// --------------------------------------------------

type CallType int

const (
	Hallcall CallType = iota // Fix: was missing iota, both were 0
	Cabcall
)

// --------------------------------------------------
// OrderState
// --------------------------------------------------

type OrderState uint8

const (
	NoCall      OrderState = 0
	Exist       OrderState = 1
	Acknowledged OrderState = 2
	Complete    OrderState = 3
)

// --------------------------------------------------
// FSM messages
// --------------------------------------------------

type NewOrderMessage struct {
	Floor    int
	Dir      Direction         // uses typed Direction, not raw MotorDirection
	CallType CallType
}

type OrderMessage struct {
	Floor int
	Dir   Direction             // uses typed Direction, not raw MotorDirection
}

// --------------------------------------------------
// FSM states
// --------------------------------------------------

type PossibleStates int

const (
	Moving PossibleStates = iota
	Idle
	DoorOpen
)

type Elevator struct {
	CurrentFloor  int
	Direction     elevio.MotorDirection
	ElevState     PossibleStates
	Malfunctioned bool
}

// --------------------------------------------------
// Node / Worldview
// --------------------------------------------------

type Node struct {
	ID          NodeID
	CabRequests [NumFloors]OrderState
	Elevator    Elevator
}

type Worldview struct {
	Nodes        []Node
	HallRequests [NumFloors][2]OrderState
}

// --------------------------------------------------
// AliveList
// --------------------------------------------------

type AliveList struct {
	Peers map[NodeID]bool // keyed by NodeID, not raw string
}