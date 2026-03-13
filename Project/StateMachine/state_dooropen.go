package Statemachine

import (
	"Driver-go/elevio"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

func handleDoorOpen(
	elev *def.Elevator,
	currentDestination *def.OrderMessage,
	isObstructedFlag *bool,
	isStoppedFlag *bool,
	drvObstructionCH <-chan bool,
	drvStopCH <-chan bool,
	drvButtonsCH <-chan elevio.ButtonEvent,
	doorTimerTimeoutCH <-chan bool,
	buttonEventCH chan<- elevio.ButtonEvent,
	obstructionMalfCH chan<- bool,
	stopMalfCH chan<- bool,
	clearOrderCH chan<- def.FsmClearOrderMessage,
	currentPosCH chan<- def.OrderMessage,
	watchdogResetCH chan bool,
	watchdogTimeoutCH chan bool,
	floorTimerResetCH chan bool,
	floorTimerTimeoutCH chan bool,
	floorTimerStopCH chan bool,
	doorTimerResetCH chan bool,
) {
	select {
	case *isObstructedFlag = <-drvObstructionCH:
		obstructionMalfCH <- *isObstructedFlag
		if *isObstructedFlag == false {
			ResetDoorTimer(doorTimerResetCH)
		}
	case *isStoppedFlag = <-drvStopCH:
		stopMalfCH <- *isStoppedFlag
		elevio.SetStopLamp(true)
		if *isStoppedFlag == false {
			ResetDoorTimer(doorTimerResetCH)
			elevio.SetStopLamp(false)
		}
	case buttonEvent := <-drvButtonsCH:
		//Drop order if elevator is already at right floor and direction
		if elev.CurrentFloor != buttonEvent.Floor {
			buttonEventCH <- buttonEvent
		} else {
			//floor matches, check direction.
			if (elev.Direction == elevio.MD_Down) && (buttonEvent.Button == elevio.BT_HallUp) {
				buttonEventCH <- buttonEvent
			} else if (elev.Direction == elevio.MD_Up) && (buttonEvent.Button == elevio.BT_HallDown) {
				buttonEventCH <- buttonEvent
			} else if buttonEvent.Button == elevio.BT_Cab {
				//cab call to current floor. Allow them time to step out:
				ResetDoorTimer(doorTimerResetCH)
			} else {
				//Hall-call in elevator's direction
				//Allow them time to step inside:
				ResetDoorTimer(doorTimerResetCH)
			}
		}
	case <-doorTimerTimeoutCH:
		if (*isObstructedFlag == false) && (*isStoppedFlag == false) {
			//close door.
			SetDoorOpenLamp(false)
			if currentDestination.Direction == elevio.MD_Stop {
				//no calls to clear
				elev.Direction = currentDestination.Direction
				elev.ElevState = def.Idle
				fmt.Println("ElevState = Idle")
			} else {
				//there is a call to clear
				if elev.CurrentFloor == currentDestination.Floor {
					//open door and clear order:
					elev.Direction = currentDestination.Direction
					SetDoorOpenLamp(true)
					ResetDoorTimer(doorTimerResetCH)
					clearOrder(elev.CurrentFloor, elev.Direction, clearOrderCH)
				} else {
					//destination is on another floor, head there.
					elev.ElevState = def.Moving
					fmt.Println("ElevState = Moving")
					elev.Direction = getElevatorDirection(elev.CurrentFloor, currentDestination.Floor)
					SetMotorDirection(elev.Direction)
					ResetFloorTimer(floorTimerResetCH, floorTimerTimeoutCH)
				}
			}
			giveLocationToDestinationFunction(elev.CurrentFloor, elev.Direction, currentPosCH)
		}
	case <-time.After(5 * time.Millisecond):
		ResetWatchdogTimer(watchdogResetCH, watchdogTimeoutCH)
	}
}
