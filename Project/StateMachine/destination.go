package Statemachine

import (
	"Driver-go/elevio"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)


func produceNextElevatorDestination(
	costFunctionOutputCH <-chan def.AssignedOrders, //receive only.
	currentElevatorPositionCH <-chan def.OrderMessage, //receive only.
	latestDestinationCH chan<- def.OrderMessage, //send only.
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
			//the elevator control-state-machine reports to us, sleeps, then checks for our response.
		case costFunctionNews := <-costFunctionOutputCH:
			costfunctionOutput = costFunctionNews
			heardFromCostfunction = true
		}
		//only after hearing from both elevator and costfunction, do we start issuing orders
		if heardFromElevator && heardFromCostfunction{
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

	//we now check in a circle on the costfunction`s output`, and look for nearest "True"/order to take.
	//if we move clockwise or counter-clockwise, depends on if elevator is moving up or down.
	if elevatorMovement.Direction == elevio.MD_Down {
		//Look for DOWN orders on a lower floor
		closestDestination = checkForTrueInInterval(elevatorMovement.Floor - 1, def.GroundFloor, def.DirDown, elevatorMovement) 
		if(cosestDestination.Direction != elevio.MD_Stop){
			return closestDestination
		}

		//Look for UP orders on a lower floor
		closestDestination = checkForTrueInInterval(def.GroundFloor, elevatorMovement.Floor - 1, def.DirUp, elevatorMovement) 
		if(cosestDestination.Direction != elevio.MD_Stop){
			return closestDestination
		}

		//direction = down, and no orders at all below it, look on current floor and above
		closestDestination = checkForTrueInInterval(elevatorMovement.Floor, def.NumFloors - 1, def.DirUp, elevatorMovement) 
		if(cosestDestination.Direction != elevio.MD_Stop){
			return closestDestination
		}

		//direction = down, and no orders at all below it, check for DOWN orders above it
		closestDestination = checkForTrueInInterval(def.NumFloors - 1, elevatorMovement.Floor, def.DirDown, elevatorMovement) 
		if(cosestDestination.Direction != elevio.MD_Stop){
			return closestDestination
		}
	} else {
		//elevator is moving upwards
		//Look for UP orders on a floor above
		closestDestination = checkForTrueInInterval(elevatorMovement.Floor + 1, def.NumFloors - 1, def.DirUp, elevatorMovement) 
		if(cosestDestination.Direction != elevio.MD_Stop){
			return closestDestination
		}

		//Look for DOWN orders on a floor above
		closestDestination = checkForTrueInInterval(def.NumFloors - 1, elevatorMovement.Floor + 1, def.DirDown, elevatorMovement) 
		if(cosestDestination.Direction != elevio.MD_Stop){
			return closestDestination
		}

		//There are no orders above the elevator. Now we check on currentfloor and below.
		closestDestination = checkForTrueInInterval(elevatorMovement.Floor, def.GroundFloor, def.DirDown, elevatorMovement) 
		if(cosestDestination.Direction != elevio.MD_Stop){
			return closestDestination
		}

		//check from ground floor going UP to and including currentfloor
		closestDestination = checkForTrueInInterval(def.GroundFloor, elevatorMovement.Floor, def.DirUp, elevatorMovement) 
		if(cosestDestination.Direction != elevio.MD_Stop){
			return closestDestination
		}
	}
	return closestDestination
}



func checkForTrueInInterval(
	var startIndex int
	var finalIndex int
	var direction def.DirectionUpDown
	var elevatorMovement def.OrderMessage
) def.OrderMessage{
	var closestDestination def.OrderMessage
	closestDestination.Direction = elevio.MD_Stop

	if(direction == def.DirUp){
		for i := startIndex; i <= finalIndex; i++ {
			if costfunctionOutput[i][def.DirUp] == true {
				closestDestination.Direction = elevio.MD_Up
				closestDestination.Floor = i
				return closestDestination
			}
		}
	}else{
		for i := startIndex; i >= finalIndex; i-- {
			if costfunctionOutput[i][def.DirDown] == true {
				closestDestination.Direction = elevio.MD_Down
				closestDestination.Floor = i
				return closestDestination
			}
		}
	}
	//if no "true" is found, then the return has direction = MD_Stop.
	return closestDestination
}


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
