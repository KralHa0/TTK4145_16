// code for state machine.
// Daniel writes this part.
package Statemachine

import (
	"Driver-go/elevio"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
)

// function is done
// copilot says its okay
func InitStateMachine(
	malfunctionStatusCH chan bool,
	clearOrderCH chan def.FsmClearOrderMessage,
	buttonEventCH chan elevio.ButtonEvent,
	costFunctionOutputCH <-chan def.AssignedOrders,
	worldviewCH chan def.Worldview,
) {
	// Initialize elevio/hardware
	elevatorAddr := fmt.Sprintf("%s:%d", def.Addr, def.Port)
	//elevio init`s using this function:`: func Init(addr string, numFloors int)
	elevio.Init(elevatorAddr, def.NumFloors)

	// Turn off all button lights:
	//this iterates for all floors, times all buttons (hallup, halldown, cab),
	for floor := 0; floor < def.NumFloors; floor++ {
		for button := elevio.ButtonType(0); button < def.NumButtons; button++ {
			elevio.SetButtonLamp(button, floor, false)
		}
	}

	//make channels associated with timers:
	doorTimerResetCH := make(chan bool, 10)
	doorTimerTimeoutCH := make(chan bool, 10)
	watchdogResetCH := make(chan bool, 10)
	watchdogTimeoutCH := make(chan bool, 10)
	floorTimerResetCH := make(chan bool, 10)
	floorTimerTimeoutCH := make(chan bool, 10)
	floorTimerStopCH := make(chan bool, 10)

	//create channels for obstruction/stop-notification from state-machine to malfunction-function
	obstructionMalfunctionCH := make(chan bool, 10)
	stopMalfunctionCH := make(chan bool, 10)

	//channel so produceNextElevatorDestination can know where the elevator is:
	currentElevatorPositionCH := make(chan def.OrderMessage, 10)

	//make channels so the elevio/hardware can send state machine module, messages about events:
	drvButtonsCH := make(chan elevio.ButtonEvent, 10)
	drvFloorsCH := make(chan int, 10)
	//on startup sends floor int if on sensor.
	//then only sends floor int once when hit a new floor.
	drvObstructionCH := make(chan bool, 10)
	//only sends bool on change in obstruction-state
	drvStopCH := make(chan bool, 10)

	//channels for getting latest destination from producenextdestination, into state-machine.
	latestDestinationCH := make(chan def.OrderMessage, 10)
	requestLatestDestinationCH := make(chan def.OrderMessage)
	receiveLatestDestinationCH := make(chan def.OrderMessage)

	//for holding door open a certain length of time:
	go DoorTimer(doorTimerResetCH, doorTimerTimeoutCH)
	//to raise alarm to the other elevators, if this elevator control state-machine-code gets stuck
	go WatchdogTimer(watchdogResetCH, watchdogTimeoutCH)
	//to raise alarm if elevator uses too long moving between two adjacent floors (because of hardware failure).
	go FloorTimer(floorTimerResetCH, floorTimerTimeoutCH, floorTimerStopCH)

	//main elevator control state-machine function
	go controlElevatorStateMachine(
		currentElevatorPositionCH,
		buttonEventCH,
		drvObstructionCH,
		drvFloorsCH,
		drvButtonsCH,
		drvStopCH,
		obstructionMalfunctionCH,
		stopMalfunctionCH,
		doorTimerResetCH,
		doorTimerTimeoutCH,
		watchdogResetCH,
		watchdogTimeoutCH,
		floorTimerResetCH,
		floorTimerStopCH,
		floorTimerTimeoutCH,
		clearOrderCH,
		requestLatestDestinationCH,
		receiveLatestDestinationCH,
	)

	//inform scheduler about our malfunction-status
	//(malfunction = obstructed OR stop-button OR watchdogtimeout OR floorTimertimeout).
	go sendMalfunctionStatus(
		obstructionMalfunctionCH,
		stopMalfunctionCH,
		watchdogTimeoutCH,
		floorTimerTimeoutCH,
		malfunctionStatusCH)

	//tells elevator where to go next:
	go produceNextElevatorDestination(costFunctionOutputCH, currentElevatorPositionCH, latestDestinationCH)

	//keeps track of the latest destination and gives it to the elevator state-machine, when requested:
	go provideLatestDestination(
		latestDestinationCH,
		requestLatestDestinationCH,
		receiveLatestDestinationCH)

	//pull events (button, floor, obstruction-switch) from hardware, and message the elevator:
	go pullHardwareNotifyStateMachine(drvButtonsCH, drvFloorsCH, drvObstructionCH, drvStopCH)

	//All lights controller
	go SetAllLights(worldviewCH)
}

// function is done
// copilot says its okay
func pullHardwareNotifyStateMachine(
	drvButtonsCH chan<- elevio.ButtonEvent, //send-only channels
	drvFloorsCH chan<- int,
	drvObstructionCH chan<- bool,
	drvStopCH chan<- bool,
) {
	//make threads which poll for events, and send them to controlElevatorStateMachine().
	go elevio.PollButtons(drvButtonsCH)
	go elevio.PollFloorSensor(drvFloorsCH)
	go elevio.PollObstructionSwitch(drvObstructionCH)
	go elevio.PollStopButton(drvStopCH)
}

// function is not done, but might work.
// copilot says I should have a common event-state. to avoid race conditions.
func controlElevatorStateMachine(
	currentElevatorPositionCH chan def.OrderMessage,
	buttonEventCH chan elevio.ButtonEvent,
	drvObstructionCH chan bool,
	drvFloorsCH chan int,
	drvButtonsCH chan elevio.ButtonEvent,
	drvStopCH chan bool,
	obstructionMalfunctionCH chan bool,
	stopMalfunctionCH chan bool,
	doorTimerResetCH chan bool,
	doorTimerTimeoutCH chan bool,
	watchdogResetCH chan bool,
	watchdogTimeoutCH chan bool,
	floorTimerResetCH chan bool,
	floorTimerStopCH chan bool,
	floorTimerTimeoutCH chan bool,
	clearOrderCH chan def.FsmClearOrderMessage,
	requestLatestDestinationCH chan def.OrderMessage,
	receiveLatestDestinationCH chan def.OrderMessage,
) {
	var Elevator def.Elevator
	//Elevator is moves downward, until hits a floorsensor.
	Elevator.ElevState = def.Moving
	fmt.Println("ElevState = Moving")
	Elevator.Direction = elevio.MD_Down
	elevio.SetMotorDirection(Elevator.Direction) //-------will this cause a jitter, if its on a floorsensor?
	Elevator.CurrentFloor = def.BetweenFloors
	//default destination: Ground floor, stop, go "Idle".
	var currentDestination def.OrderMessage
	currentDestination.Floor = def.GroundFloor
	currentDestination.Direction = elevio.MD_Stop
	ResetWatchdogTimer(watchdogResetCH, watchdogTimeoutCH)
	ResetFloorTimer(floorTimerResetCH, floorTimerTimeoutCH)
	var isObstructedFlag bool = false
	var isStoppedFlag bool = false
	//main loop:
	for {
		currentDestination = getLatestDestination(currentDestination, requestLatestDestinationCH, receiveLatestDestinationCH)
		switch Elevator.ElevState {
		case def.Moving:
			select {
			case isObstructedFlag = <-drvObstructionCH:
				obstructionMalfunctionCH <- isObstructedFlag
				elevio.SetMotorDirection(stopOrResumeMovinginueMoving(isObstructedFlag, isStoppedFlag, Elevator.Direction))
			case isStoppedFlag = <-drvStopCH:
				stopMalfunctionCH <- isStoppedFlag
				elevio.SetMotorDirection(stopOrResumeMovinginueMoving(isObstructedFlag, isStoppedFlag, Elevator.Direction))
			case buttonEvent := <-drvButtonsCH:
				buttonEventCH <- buttonEvent
			case Elevator.CurrentFloor = <-drvFloorsCH:
				elevio.SetFloorIndicator(Elevator.CurrentFloor)
				if currentDestination.Direction == elevio.MD_Stop {
					//there are no orders to clear
					Elevator.Direction = currentDestination.Direction
					Elevator.ElevState = def.Idle
					fmt.Println("ElevState = Idle")
					StopFloorTimer(floorTimerStopCH, floorTimerTimeoutCH)
				} else {
					//there is an order to clear
					if Elevator.CurrentFloor == currentDestination.Floor {
						elevio.SetMotorDirection(elevio.MD_Stop)
						Elevator.Direction = currentDestination.Direction
						Elevator.ElevState = def.DoorOpen
						fmt.Println("ElevState = DoorOpen")
						elevio.SetDoorOpenLamp(true)
						ResetDoorTimer(doorTimerResetCH)
						StopFloorTimer(floorTimerStopCH, floorTimerTimeoutCH)
						clearOrder(Elevator.CurrentFloor, Elevator.Direction, clearOrderCH)
					} else {
						//keep moving in current direciton
						ResetFloorTimer(floorTimerResetCH, floorTimerTimeoutCH)
					}
				}
				giveLocationToDestinationFunction(Elevator.CurrentFloor, Elevator.Direction, currentElevatorPositionCH)
			case <-time.After(5 * time.Millisecond):
				ResetWatchdogTimer(watchdogResetCH, watchdogTimeoutCH)
			}
		case def.DoorOpen:
			select {
			case isObstructedFlag = <-drvObstructionCH:
				obstructionMalfunctionCH <- isObstructedFlag
				if isObstructedFlag == false {
					ResetDoorTimer(doorTimerResetCH)
				}
			case isStoppedFlag = <-drvStopCH:
				stopMalfunctionCH <- isStoppedFlag
				if isStoppedFlag == false {
					ResetDoorTimer(doorTimerResetCH)
				}
			case buttonEvent := <-drvButtonsCH:
				//Drop order if elevator is already at right floor and direction
				if Elevator.CurrentFloor != buttonEvent.Floor {
					buttonEventCH <- buttonEvent
				} else {
					//floor matches, check direction.
					if (Elevator.Direction == elevio.MD_Down) && (buttonEvent.Button == elevio.BT_HallUp) {
						buttonEventCH <- buttonEvent
					} else if (Elevator.Direction == elevio.MD_Up) && (buttonEvent.Button == elevio.BT_HallDown) {
						buttonEventCH <- buttonEvent
					} else if buttonEvent.Button == elevio.BT_Cab {
						//cab call to current floor. Allow them time to step out:
						ResetDoorTimer(doorTimerResetCH)
					} else {
						//Hall-call in elevator`s direction`
						//Allow them time to step inside:
						ResetDoorTimer(doorTimerResetCH)
					}
				}
			case <-doorTimerTimeoutCH:
				if (isObstructedFlag == false) && (isStoppedFlag == false) {
					//close door.
					elevio.SetDoorOpenLamp(false)
					if Elevator.CurrentFloor == currentDestination.Floor {
						if currentDestination.Direction == elevio.MD_Stop {
							Elevator.Direction = currentDestination.Direction
							Elevator.ElevState = def.Idle
							fmt.Println("ElevState = Idle")
						} else {
							//open door and clear order:
							Elevator.Direction = currentDestination.Direction
							elevio.SetDoorOpenLamp(true)
							ResetDoorTimer(doorTimerResetCH)
							clearOrder(Elevator.CurrentFloor, Elevator.Direction, clearOrderCH)
						}
					} else {
						//destination is on another Floor, head there.
						Elevator.ElevState = def.Moving
						fmt.Println("ElevState = Moving")
						Elevator.Direction = getElevatorDirection(Elevator.CurrentFloor, currentDestination.Floor)
						elevio.SetMotorDirection(Elevator.Direction)
						ResetFloorTimer(floorTimerResetCH, floorTimerTimeoutCH)
					}
					giveLocationToDestinationFunction(Elevator.CurrentFloor, Elevator.Direction, currentElevatorPositionCH)
				}
			case <-time.After(5 * time.Millisecond):
				ResetWatchdogTimer(watchdogResetCH, watchdogTimeoutCH)
			}
		case def.Idle: //"Idle" = Elevator is stopped, doorclosed, checking for something to do.
			select {
			case isObstructedFlag = <-drvObstructionCH:
				obstructionMalfunctionCH <- isObstructedFlag
			case isStoppedFlag = <-drvStopCH:
				stopMalfunctionCH <- isStoppedFlag
			case buttonEvent := <-drvButtonsCH:
				buttonEventCH <- buttonEvent
			case <-time.After(5 * time.Millisecond):
				if (isObstructedFlag == false) && (isStoppedFlag == false) {
					//check if there is something to do:
					if Elevator.CurrentFloor == currentDestination.Floor {
						if currentDestination.Direction != elevio.MD_Stop {
							//open doors, clear order
							Elevator.ElevState = def.DoorOpen
							fmt.Println("ElevState = DoorOpen")
							elevio.SetDoorOpenLamp(true)
							ResetDoorTimer(doorTimerResetCH)
							Elevator.Direction = currentDestination.Direction
							clearOrder(Elevator.CurrentFloor, Elevator.Direction, clearOrderCH)
						}
					} else {
						//head towards destination:
						Elevator.ElevState = def.Moving
						fmt.Println("ElevState = Moving")
						Elevator.Direction = getElevatorDirection(Elevator.CurrentFloor, currentDestination.Floor)
						elevio.SetMotorDirection(Elevator.Direction)
						ResetFloorTimer(floorTimerResetCH, floorTimerTimeoutCH)
					}
					giveLocationToDestinationFunction(Elevator.CurrentFloor, Elevator.Direction, currentElevatorPositionCH)
				}
				ResetWatchdogTimer(watchdogResetCH, watchdogTimeoutCH)
			}
		}
	}
}

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
				if elevatorMovement.Direction == elevio.MD_Stop {
					//elevator is Idle.
					newDestination = findClosestOrderInAnyDirection(elevatorMovement, costfunctionOutput)
				} else {
					//elevator is moving in some direction
					newDestination = findClosestDestinationGivenCurrentDirectionDirection(elevatorMovement, costfunctionOutput)
				}
			}
			//Send destination to elevator:
			latestDestinationCH <- newDestination
		}
	}
}

// function is done.
// copilot says its okay
func findClosestOrderInAnyDirection(
	elevatorMovement def.OrderMessage,
	costfunctionOutput def.AssignedOrders,
) def.OrderMessage {
	var elevatorOrder def.OrderMessage
	var floorToCheck int

	//elevator is idle
	//there exists at least one order in costfunctionOutput
	//Check outward from the current floor, until an order is found
	//first check the current floor for orders
	//then check 1 floor above. "1" is then the floorOffset.
	//then check 1 floor below
	//then check 2 floors above etc.
	//only check floors that exist.
	for floorOffset := 0; floorOffset < def.NumFloors; floorOffset++ {
		//check floor above or at current location
		floorToCheck = elevatorMovement.Floor + floorOffset
		if (floorToCheck >= def.GroundFloor) && (floorToCheck < def.NumFloors) {
			//check if there is a DOWN-order on it:
			if costfunctionOutput[floorToCheck][def.DirDown] == true {
				elevatorOrder.Floor = floorToCheck
				elevatorOrder.Direction = elevio.MD_Down
				break
				//check if there is a UP-order on it:
			} else if costfunctionOutput[floorToCheck][def.DirUp] == true {
				elevatorOrder.Floor = floorToCheck
				elevatorOrder.Direction = elevio.MD_Up
				break
			}
		}
		//check floor below or at current location
		floorToCheck = elevatorMovement.Floor - floorOffset
		if (floorToCheck >= def.GroundFloor) && (floorToCheck < def.NumFloors) {
			//check if there is a DOWN-order on it:
			if costfunctionOutput[floorToCheck][def.DirDown] == true {
				elevatorOrder.Floor = floorToCheck
				elevatorOrder.Direction = elevio.MD_Down
				break
				//check if there is a UP-order on it:
			} else if costfunctionOutput[floorToCheck][def.DirUp] == true {
				elevatorOrder.Floor = floorToCheck
				elevatorOrder.Direction = elevio.MD_Up
				break
			}
		}
	}
	return elevatorOrder
}

// function is done
// copilot says its okay
func findClosestDestinationGivenCurrentDirectionDirection(
	elevatorMovement def.OrderMessage,
	costfunctionOutput def.AssignedOrders,
) def.OrderMessage {
	var closestDestination def.OrderMessage

	//elevator is moving, when this function is called.
	//check if its moving downwards.
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
				//tell it to stop and go idle there
				closestDestination.Direction = elevio.MD_Up
				closestDestination.Floor = i
				return closestDestination
			}
		}
		//If moving down, and no orders at all below it, tell it to stop.
		closestDestination.Direction = elevio.MD_Stop
		closestDestination.Floor = elevatorMovement.Floor - 1
		return closestDestination
	} else {
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
				//tell it to stop and go idle there
				closestDestination.Direction = elevio.MD_Down
				closestDestination.Floor = i
				return closestDestination
			}
		}
		//If no orders at all above it, tell it to stop.
		closestDestination.Direction = elevio.MD_Stop
		closestDestination.Floor = elevatorMovement.Floor + 1
		return closestDestination
	}
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

// function is done
// copilot says its okay
func stopOrResumeMovinginueMoving(
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

// go-function is done.
// copilot says its okay.
func sendMalfunctionStatus(
	obstructionMalfunctionCH <-chan bool, //recieve only
	stopMalfunctionCH <-chan bool, //recieve only
	watchdogTimeoutCH <-chan bool, //recieve only
	floorTimerTimeoutCH <-chan bool, //recieve only
	malfunctionStatusCH chan<- bool, //send only
) {
	//malfunctioned = obstructed OR stopped OR watchdogtimeout OR floorTimerTimeout
	var currentMalfunctionStatus bool = false
	var newMalfunctionStatus bool = false
	var isObstructed bool = false
	var isStoppedFlag bool = false
	var isWatchdogtimeout bool = false
	var isFloorTimerTimeout bool = false
	var FirstTransmission bool = true
	for {
		//waits for update:
		select {
		case isObstructed = <-obstructionMalfunctionCH:
			fmt.Printf("Obstruction-status: %v\n", isObstructed)
		case isStoppedFlag = <-stopMalfunctionCH:
			fmt.Printf("Stopped-status: %v\n", isStoppedFlag)
		case isWatchdogtimeout = <-watchdogTimeoutCH:
			fmt.Printf("Watchdogtimeout-status: %v\n", isWatchdogtimeout)
		case isFloorTimerTimeout = <-floorTimerTimeoutCH:
			fmt.Printf("FloorTimerTimeout-status: %v\n", isFloorTimerTimeout)
		}
		newMalfunctionStatus = (isObstructed || isStoppedFlag || isWatchdogtimeout || isFloorTimerTimeout)
		//only send an update on channel, if there has been a change, or on startup.
		if (newMalfunctionStatus != currentMalfunctionStatus) || FirstTransmission {
			currentMalfunctionStatus = newMalfunctionStatus
			malfunctionStatusCH <- currentMalfunctionStatus
			FirstTransmission = false
		}
	}
}

// function is done
func SetAllLights(wvCh <-chan def.Worldview) {
	ID := nw.GetIp()
	for wv := range wvCh {
		for _, node := range wv.Nodes {
			if node.ID == ID {
				checkAllLights(node, wv)
			}
		}

	}
}

// function is done
func checkAllLights(node def.Node, wv def.Worldview) {
	for floor := 0; floor < def.NumFloors; floor++ {
		// cabrequests
		if node.CabRequests[floor] == def.Acknowledged {
			elevio.SetButtonLamp(elevio.BT_Cab, floor, true)
		} else {
			elevio.SetButtonLamp(elevio.BT_Cab, floor, false)
		}

		// hallrequests UP
		if wv.HallRequests[floor][def.DirUp] == def.Acknowledged {
			elevio.SetButtonLamp(elevio.BT_HallUp, floor, true)
		} else {
			elevio.SetButtonLamp(elevio.BT_HallUp, floor, false)
		}

		// hallrequests Down
		if wv.HallRequests[floor][def.DirDown] == def.Acknowledged {
			elevio.SetButtonLamp(elevio.BT_HallDown, floor, true)
		} else {
			elevio.SetButtonLamp(elevio.BT_HallDown, floor, false)
		}
	}
}
