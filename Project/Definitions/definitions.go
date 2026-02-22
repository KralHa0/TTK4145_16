package definitions

import (
	"Driver-go/elevio"
	"time"
)

const (
	NumFloors = 4
	MsgFrq    = 100 * time.Millisecond
)

type Behavior int

const (
	Idle Behavior = iota
	Moving
	DoorOpen
)

type OrderState uint8

const (
	NoCall       OrderState = 0
	Exist        OrderState = 1
	Acknowledged OrderState = 2
	Complete     OrderState = 3
)

type ElevState struct {
	Floor         int
	Direction     elevio.MotorDirection
	Behavior      Behavior
	Malfunctioned bool
}

type Node struct {
	ID          string
	CabRequests [NumFloors]OrderState
	ElevState   ElevState
}

type Worldview struct {
	Nodes        []Node
	HallRequests [NumFloors][2]OrderState
}

type AliveList struct {
	Peers map[string]bool //Map of peer IDs to their alive status
}
