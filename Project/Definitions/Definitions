//Definitions, constants, which are used by all modules.

package definitions

import (
	"Project/Hardware/elevio/elevator_io"
	"os"
)

// Default values, elevator:
var Addr = "localhost"
var Port = 15657

const num_floors = 4
const num_buttons = 3     //the 3 are: 0:hallup, 1:halldown, 2:cab. Defined in elevio module.
const between_floors = -1 //a floor-value used under init.
const ground_floor = 0

// consts for timers
const door_open_timeout = 3000 // ms. (door should be open for 3 seconds, then close).
const watchdog_timeout = 5000  // ms  (should not use more than 5 seconds to move between 2 floors)

// states for instantiated Elevator:
type possible_states int

const (
	moving possible_states = iota
	idle
	door_open
)

// struct to hold elevator values
type Elevator struct {
	state         possible_states
	dir     	  elevio.MotorDirection
	current_floor int
	ID            int
	is_obstructed bool //default value = false
}

// struct so the the modules can message-pass about:
//
//	"where is the elevator currently",
//	"which order has been cleared?",
//	"which floor should I go to next" etc.
type floor_and_dir struct {
	floor     int
	dir elevio.MotorDirection
}
