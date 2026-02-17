package main

import "Driver-go/elevio"

const NumFloors = 4

type State int
const (
    Idle     State = iota
    Moving
    DoorOpen
)

type CabRequests struct {
    Requests [NumFloors]uint8
}

type HallRequests struct {
    Requests [NumFloors][2]uint8
}

type ElevState struct {
    Floor     int
    Direction elevio.MotorDirection
    State     State
	Malfunctioned bool
}

type Node struct {
    ID          string
    CabRequests CabRequests
    ElevState   ElevState
}

type Worldview struct {
    Nodes        []Node
    HallRequests HallRequests
}