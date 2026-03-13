// code for state machine.
// Daniel writes this part.
package Statemachine

import (
	"Driver-go/elevio"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

// function is done
// copilot says its okay
func InitStateMachine(
	malfunctionStatusCH chan bool,
	clearOrderCH chan def.FsmClearOrderMessage,
	buttonEventCH chan elevio.ButtonEvent,
	costFunctionOutputCH <-chan def.AssignedOrders,
	worldviewCH chan def.Worldview,
	fsmToOMElevStateCh chan def.Elevator,
) {
	// Initialize elevio/hardware
	elevatorAddr := fmt.Sprintf("%s:%d", def.Addr, def.Port)
	//elevio init`s using this function:`: func Init(addr string, numFloors int)
	elevio.Init(elevatorAddr, def.NumFloors)

	// Turn off all button lights:
	//this iterates for all floors, times all buttons (hallup, halldown, cab),
	for floor := 0; floor < def.NumFloors; floor++ {
		for button := elevio.ButtonType(0); button < def.NumButtons; button++ {
			SetButtonLamp(button, floor, false)
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
	drvStopCH := make(chan bool)

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

	//pull events (button, floor, obstruction-switch) from hardware, and message the elevator:
	go pullHardwareNotifyStateMachine(drvButtonsCH, drvFloorsCH, drvObstructionCH, drvStopCH)

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
		fsmToOMElevStateCh,
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

	//All lights controller
	go SetAllLights(worldviewCH)
}

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
	fsmToOMElevStateCh chan def.Elevator,
) {
	time.Sleep(1 * time.Second)
	var elev def.Elevator

	elev.CurrentFloor = def.BetweenFloors

	select {
	case elev.CurrentFloor = <-drvFloorsCH:
		//Elevator is already on a floorsensor:
		elev.ElevState = def.Idle
		fmt.Println("Init ElevState = Idle")
		elev.Direction = elevio.MD_Stop
	default:
		//move downward, until hits a floorsensor.
		elev.ElevState = def.Moving
		fmt.Println("Init ElevState = Moving")
		elev.Direction = elevio.MD_Down
		ResetFloorTimer(floorTimerResetCH, floorTimerTimeoutCH)
	}
	SetMotorDirection(elev.Direction)
	//default destination: stop, go "Idle".
	var currentDestination def.OrderMessage
	currentDestination.Floor = def.GroundFloor
	currentDestination.Direction = elevio.MD_Stop
	ResetWatchdogTimer(watchdogResetCH, watchdogTimeoutCH)
	var isObstructedFlag bool = false
	var isStoppedFlag bool = false
	//main loop:
	for {
		time.Sleep(200 * time.Millisecond)
		currentDestination = getLatestDestination(currentDestination, requestLatestDestinationCH, receiveLatestDestinationCH)
		if elev.CurrentFloor != def.BetweenFloors {
			select {
			case fsmToOMElevStateCh <- elev:
			default:
			}
		}

		switch elev.ElevState {
		case def.Moving:
			handleMoving(
				&elev, &currentDestination, &isObstructedFlag, &isStoppedFlag,
				drvObstructionCH, drvStopCH, drvButtonsCH, drvFloorsCH,
				buttonEventCH, obstructionMalfunctionCH, stopMalfunctionCH,
				clearOrderCH, currentElevatorPositionCH,
				watchdogResetCH, watchdogTimeoutCH,
				floorTimerResetCH, floorTimerTimeoutCH, floorTimerStopCH,
				doorTimerResetCH,
			)
		case def.DoorOpen:
			handleDoorOpen(
				&elev, &currentDestination, &isObstructedFlag, &isStoppedFlag,
				drvObstructionCH, drvStopCH, drvButtonsCH, doorTimerTimeoutCH,
				buttonEventCH, obstructionMalfunctionCH, stopMalfunctionCH,
				clearOrderCH, currentElevatorPositionCH,
				watchdogResetCH, watchdogTimeoutCH,
				floorTimerResetCH, floorTimerTimeoutCH, floorTimerStopCH,
				doorTimerResetCH,
			)
		case def.Idle:
			handleIdle(
				&elev, &currentDestination, &isObstructedFlag, &isStoppedFlag,
				drvObstructionCH, drvStopCH, drvButtonsCH,
				buttonEventCH, obstructionMalfunctionCH, stopMalfunctionCH,
				clearOrderCH, currentElevatorPositionCH,
				watchdogResetCH, watchdogTimeoutCH,
				floorTimerResetCH, floorTimerTimeoutCH,
				doorTimerResetCH,
			)
		}
	}
}
