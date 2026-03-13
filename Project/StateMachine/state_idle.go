package Statemachine

import (
	"Driver-go/elevio"
	"fmt"
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

func handleIdle(
	elev                *def.Elevator,
	currentDestination  *def.OrderMessage,
	isObstructedFlag    *bool,
	isStoppedFlag       *bool,
	drvObstructionCH    <-chan bool,
	drvStopCH           <-chan bool,
	drvButtonsCH        <-chan elevio.ButtonEvent,
	buttonEventCH       chan<- elevio.ButtonEvent,
	obstructionMalfCH   chan<- bool,
	stopMalfCH          chan<- bool,
	clearOrderCH        chan<- def.FsmClearOrderMessage,
	currentPosCH        chan<- def.OrderMessage,
	watchdogResetCH     chan bool,
	watchdogTimeoutCH   chan bool,
	floorTimerResetCH   chan bool,
	floorTimerTimeoutCH chan bool,
	doorTimerResetCH    chan bool,
) {
	select {
	case *isObstructedFlag = <-drvObstructionCH:
		obstructionMalfCH <- *isObstructedFlag
	case *isStoppedFlag = <-drvStopCH:
		stopMalfCH <- *isStoppedFlag
		elevio.SetStopLamp(true)
		if *isStoppedFlag == false {
			elevio.SetStopLamp(false)
		}
	case buttonEvent := <-drvButtonsCH:
		buttonEventCH <- buttonEvent
	case <-time.After(5 * time.Millisecond):
		if (*isObstructedFlag == false) && (*isStoppedFlag == false) {
			//check if there is something to do:
			if currentDestination.Direction != elevio.MD_Stop {
				if elev.CurrentFloor == currentDestination.Floor {
					//open doors, clear order
					elev.ElevState = def.DoorOpen
					fmt.Println("ElevState = DoorOpen")
					SetDoorOpenLamp(true)
					ResetDoorTimer(doorTimerResetCH)
					elev.Direction = currentDestination.Direction
					clearOrder(elev.CurrentFloor, elev.Direction, clearOrderCH)
				} else {
					//head towards destination:
					elev.ElevState = def.Moving
					fmt.Println("ElevState = Moving")
					elev.Direction = getElevatorDirection(elev.CurrentFloor, currentDestination.Floor)
					SetMotorDirection(elev.Direction)
					ResetFloorTimer(floorTimerResetCH, floorTimerTimeoutCH)
				}
			}
			giveLocationToDestinationFunction(elev.CurrentFloor, elev.Direction, currentPosCH)
		}
		ResetWatchdogTimer(watchdogResetCH, watchdogTimeoutCH)
	}
}
