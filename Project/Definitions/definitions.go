package definitions

import "Driver-go/elevio"

const NumFloors = 4

type State int

const (
	Idle State = iota
	Moving
	DoorOpen
)

type OrderState uint8

const (
	NoCall       OrderState = 0
	Available    OrderState = 1
	Taken        OrderState = 2
	Complete     OrderState = 3
	Acknowledged OrderState = 4
)

type ElevState struct {
	Floor         int
	Direction     elevio.MotorDirection
	State         State
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
