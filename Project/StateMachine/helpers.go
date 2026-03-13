package Statemachine

import (
	"Driver-go/elevio"
	"fmt"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

// function is done
// copilot says its okay
func getElevatorDirection(
	currentFloor int,
	destinationFloor int,
) elevio.MotorDirection {
	//currentFloor != destinationFloor, when this function is called.
	if currentFloor == destinationFloor {
		return elevio.MD_Stop //just in case function is called wrongly.
	}
	if currentFloor > destinationFloor {
		return elevio.MD_Down
	} else {
		return elevio.MD_Up
	}
}

// function is done
// copilot says its okay.
func clearOrder(
	clearedFloor int,
	clearedDirection elevio.MotorDirection,
	clearOrderCH chan<- def.FsmClearOrderMessage, //send only
) {
	//put floor and dir into one struct-instance
	var clearedMessage def.FsmClearOrderMessage
	clearedMessage.Floor = clearedFloor
	clearedOrderUpDown, complete := def.DirFromMotor(clearedDirection)
	if complete == false {
		fmt.Println("[ERROR] Direction into dirFromMotor is MD_stop")
	}
	clearedMessage.Dir = clearedOrderUpDown
	//send it to order manager
	clearOrderCH <- clearedMessage
}

// function is done
// copilot says its okay.
func giveLocationToDestinationFunction(
	currentFloor int,
	currentDirection elevio.MotorDirection,
	currentElevatorPositionCH chan<- def.OrderMessage, //send only
) {
	// Build the elevator's current position message
	var currentElevPosition def.OrderMessage
	currentElevPosition.Floor = currentFloor
	currentElevPosition.Direction = currentDirection
	// Send it to the destination function
	currentElevatorPositionCH <- currentElevPosition
}

// function is done
// copilot says its okay
func stopOrResumeMoving(
	isObstructedFlag bool,
	isStoppedFlag bool,
	currentElevatorDirection elevio.MotorDirection,
) elevio.MotorDirection {
	if (isObstructedFlag == true) || (isStoppedFlag == true) {
		return elevio.MD_Stop
	} else {
		//keep moving the way the elevator was going before
		return currentElevatorDirection
	}
}

// function is done
// copilot says its okay.
func getLatestDestination(
	currentDestination def.OrderMessage,
	requestLatestDestinationCH chan def.OrderMessage,
	receiveLatestDestinationCH chan def.OrderMessage,
) def.OrderMessage {
	//request the latest destination, by sending our current destination:
	requestLatestDestinationCH <- currentDestination

	//wait for reply: (Blocking wait is ok).
	updatedDestination := <-receiveLatestDestinationCH

	// Either unchanged (updatedDestination == currentDestination) or updated.
	return updatedDestination
}
