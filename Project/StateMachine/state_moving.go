package Statemachine

import (
	"Driver-go/elevio"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

func handleMoving(
	elev *def.Elevator,
	currentDestination *def.OrderMessage,
	isObstructedFlag *bool,
	isStoppedFlag *bool,
	drvObstructionCH <-chan bool,
	drvStopCH <-chan bool,
	drvButtonsCH <-chan elevio.ButtonEvent,
	drvFloorsCH <-chan int,
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
		SetMotorDirection(stopOrResumeMoving(*isObstructedFlag, *isStoppedFlag, elev.Direction))
	case *isStoppedFlag = <-drvStopCH:
		stopMalfCH <- *isStoppedFlag
		elevio.SetStopLamp(true)
		if *isStoppedFlag == false {
			elevio.SetStopLamp(false)
		}
		SetMotorDirection(stopOrResumeMoving(*isObstructedFlag, *isStoppedFlag, elev.Direction))
	case buttonEvent := <-drvButtonsCH:
		buttonEventCH <- buttonEvent
	case elev.CurrentFloor = <-drvFloorsCH:
		SetFloorIndicator(elev.CurrentFloor)
		if currentDestination.Direction == elevio.MD_Stop {
			//there are no orders to clear
			elev.Direction = currentDestination.Direction
			elev.ElevState = def.Idle
			fmt.Println("ElevState = Idle")
			SetMotorDirection(elev.Direction)
			StopFloorTimer(floorTimerStopCH, floorTimerTimeoutCH)
		} else {
			//there is an order to clear
			if elev.CurrentFloor == currentDestination.Floor {
				SetMotorDirection(elevio.MD_Stop)
				elev.Direction = currentDestination.Direction
				elev.ElevState = def.DoorOpen
				fmt.Println("ElevState = DoorOpen")
				SetDoorOpenLamp(true)
				ResetDoorTimer(doorTimerResetCH)
				StopFloorTimer(floorTimerStopCH, floorTimerTimeoutCH)
				clearOrder(elev.CurrentFloor, elev.Direction, clearOrderCH)
			} else {
				// Recalculate direction in case destination changed since last loop iteration
				newDir := getElevatorDirection(elev.CurrentFloor, currentDestination.Floor)
				if newDir != elev.Direction {
					elev.Direction = newDir
					SetMotorDirection(elev.Direction)
				}
				ResetFloorTimer(floorTimerResetCH, floorTimerTimeoutCH)
			}
		}
		giveLocationToDestinationFunction(elev.CurrentFloor, elev.Direction, currentPosCH)
	case <-time.After(5 * time.Millisecond):
		ResetWatchdogTimer(watchdogResetCH, watchdogTimeoutCH)
	}
}
