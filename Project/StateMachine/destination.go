package Statemachine

import (
	"Driver-go/elevio"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

// go-function is done.
// copilot says its okay
func produceNextElevatorDestination(
	costFunctionOutputCH <-chan def.AssignedOrders, //receive only.
	currentElevatorPositionCH <-chan def.OrderMessage, //receive only.
	latestDestinationCH chan<- def.OrderMessage, //send only. it sends where elevator should go next.
) {
	var heardFromElevator bool = false
	var heardFromCostfunction bool = false
	var elevatorMovement def.OrderMessage
	var costfunctionOutput def.AssignedOrders
	var newDestination def.OrderMessage
	//main loop:
	for {
		//wait for news:
		select {
		case elevatorNews := <-currentElevatorPositionCH:
			elevatorMovement = elevatorNews
			heardFromElevator = true
			//the elevator reports to us, sleeps for 5 mS, then checks for our response.
		case costFunctionNews := <-costFunctionOutputCH:
			costfunctionOutput = costFunctionNews
			heardFromCostfunction = true
		}
		//only after hearing from both elevator and costfunction, do we start issuing orders
		if heardFromElevator && heardFromCostfunction {
			if orderCount(costfunctionOutput) == 0 {
				//no orders to take. Stop elevator. Let it go Idle.
				newDestination.Floor = elevatorMovement.Floor
				newDestination.Direction = elevio.MD_Stop
			} else {
				//there are orders for the elevator to take.
				newDestination = findClosestDestination(elevatorMovement, costfunctionOutput)
			}
			//Send destination to elevator:
			latestDestinationCH <- newDestination
		}
	}
}

// go-function is done
// copilot says its okay.
func provideLatestDestination(
	latestDestinationCH <-chan def.OrderMessage, //recieve only
	requestLatestDestinationCH <-chan def.OrderMessage, //recieve only
	receiveLatestDestinationCH chan<- def.OrderMessage, //send only
) {
	var gottenFirstUpdate bool = false
	var latestDestination def.OrderMessage
	for {
		//wait for either: an update to destination, or a request for the latest destination
		//(These events both happen rarely, avoiding full buffers)
		select {
		case destinationUpdate := <-latestDestinationCH:
			//there was an update to our destination
			latestDestination = destinationUpdate
			gottenFirstUpdate = true
		case currentDestination := <-requestLatestDestinationCH:
			if gottenFirstUpdate == true {
				//send the updated Destination
				receiveLatestDestinationCH <- latestDestination
			} else {
				//Send the caller's current destination back
				receiveLatestDestinationCH <- currentDestination
			}
		}
	}
}

// function is done
// copilot says its okay
func findClosestDestination(
	elevatorMovement def.OrderMessage,
	costfunctionOutput def.AssignedOrders,
) def.OrderMessage {
	var closestDestination def.OrderMessage

	//there is at least one order to take
	//If elevator is idle (direction = stop), then just give it a direction:
	if elevatorMovement.Direction == elevio.MD_Stop {
		elevatorMovement.Direction = elevio.MD_Down
	}

	//we now walk an entire circle (up and down) the orderfunction, and look for nearest "True"= order.
	if elevatorMovement.Direction == elevio.MD_Down {
		//Look for DOWN orders on a lower floor
		for i := elevatorMovement.Floor - 1; i >= def.GroundFloor; i-- {
			if costfunctionOutput[i][def.DirDown] == true {
				closestDestination.Direction = elevio.MD_Down
				closestDestination.Floor = i
				return closestDestination
			}
		}
		//Look for UP orders on a lower floor
		for i := def.GroundFloor; i < elevatorMovement.Floor; i++ {
			if costfunctionOutput[i][def.DirUp] == true {
				closestDestination.Direction = elevio.MD_Up
				closestDestination.Floor = i
				return closestDestination
			}
		}
		//If direction = down, and no orders at all below it, look on current floor and above
		for i := elevatorMovement.Floor; i < def.NumFloors; i++ {
			if costfunctionOutput[i][def.DirUp] == true {
				closestDestination.Direction = elevio.MD_Up
				closestDestination.Floor = i
				return closestDestination
			}
		}

		//If direction = down, and no orders at all below it, or going up, check orders above it going down
		for i := def.NumFloors - 1; i >= elevatorMovement.Floor; i-- {
			if costfunctionOutput[i][def.DirDown] == true {
				closestDestination.Direction = elevio.MD_Down
				closestDestination.Floor = i
				return closestDestination
			}
		}
	} else {
		//we now walk an entire circle in the oposite direction (up and down) the orderfunction, and look for nearest "True"= order.
		//elevator is moving upwards
		//Look for UP orders on a floor above
		for i := elevatorMovement.Floor + 1; i < def.NumFloors; i++ {
			if costfunctionOutput[i][def.DirUp] == true {
				closestDestination.Direction = elevio.MD_Up
				closestDestination.Floor = i
				return closestDestination
			}
		}
		//Look for DOWN orders on a floor above
		for i := def.NumFloors - 1; i > elevatorMovement.Floor; i-- {
			if costfunctionOutput[i][def.DirDown] == true {
				closestDestination.Direction = elevio.MD_Down
				closestDestination.Floor = i
				return closestDestination
			}
		}
		//There are no orders above the elevator. Now we check on currentfloor and below.
		for i := elevatorMovement.Floor; i >= def.GroundFloor; i-- {
			if costfunctionOutput[i][def.DirDown] == true {
				closestDestination.Direction = elevio.MD_Down
				closestDestination.Floor = i
				return closestDestination
			}
		}
		//check from ground floor going UP to and including currentfloor
		for i := def.GroundFloor; i <= elevatorMovement.Floor; i++ {
			if costfunctionOutput[i][def.DirUp] == true {
				closestDestination.Direction = elevio.MD_Up
				closestDestination.Floor = i
				return closestDestination
			}
		}
	}
	return closestDestination
}

// function is done
// copilot says its okay
func orderCount(
	costfunctionOutput def.AssignedOrders,
) int {
	var nrOfOrders int = 0
	var nrOfRows = len(costfunctionOutput)
	var nrOfColumns = len(costfunctionOutput[0])

	//counts and returns how many "true" there are in costfunctionOutput
	for row := 0; row < nrOfRows; row++ {
		for col := 0; col < nrOfColumns; col++ {
			if costfunctionOutput[row][col] == true {
				nrOfOrders += 1
			}
		}
	}
	return nrOfOrders
}
