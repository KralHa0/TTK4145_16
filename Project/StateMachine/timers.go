package Statemachine

import (
	"time"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)


// Doortimer is to know when to close the door, after it has been opened.
func DoorTimer(doorTimerResetCH chan bool, doorTimerTimeoutCH chan bool) {
	timer := time.NewTimer(def.DoorOpenTimeout)
	timer.Stop()
	for {
		select {
		case <-doorTimerResetCH:
			//drain timer.C before reset.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(def.DoorOpenTimeout)
		case <-timer.C:
			signalDoortimeout(doorTimerTimeoutCH)
		}
	}
}

func ResetDoorTimer(doorTimerResetCH chan bool) {
	doorTimerResetCH <- true
}

func signalDoortimeout(doorTimerTimeoutCH chan bool) {
	doorTimerTimeoutCH <- true
}


// Watchdogtimer is to let the other elevators know, if the state-machine code is stuck/deadlock on this elevator.
func WatchdogTimer(watchdogResetCH chan bool, watchdogTimeoutCH chan bool) {
	timer := time.NewTimer(def.WatchdogTimeout)
	timer.Stop()
	for {
		select {
		case <-watchdogResetCH:
			//drain timer.C before reset.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(def.WatchdogTimeout)
		case <-timer.C:
			signalWatchdogTimeout(watchdogTimeoutCH)
		}
	}
}

func ResetWatchdogTimer(watchdogResetCH chan bool, watchdogTimeoutCH chan bool) {
	watchdogResetCH <- true
	//Signal that we are not currently in a state of watchdogTimeout:
	watchdogTimeoutCH <- false
}

func signalWatchdogTimeout(watchdogTimeoutCH chan bool) {
	watchdogTimeoutCH <- true
}


// Floortimer is to make sure the time moving between floors is reasonable.
func FloorTimer(
	floorTimerResetCH chan bool,
	floorTimerTimeoutCH chan bool,
	floorTimerStopCH chan bool,
) {
	timer := time.NewTimer(def.FloorTimerTimeout)
	timer.Stop()
	for {
		select {
		case <-floorTimerResetCH:
			//drain timer.C before reset:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(def.FloorTimerTimeout)
		case <-timer.C:
			signalFloorTimerTimeout(floorTimerTimeoutCH)
		case <-floorTimerStopCH:
			//drain timer.C and stop the timer, so that it doesnt sound the alarm.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
		}
	}
}

func ResetFloorTimer(floorTimerResetCH chan bool, floorTimerTimeoutCH chan bool) {
	floorTimerResetCH <- true
	//Signal that we do not currently have a FloorTimerTimeout:
	floorTimerTimeoutCH <- false
}

func signalFloorTimerTimeout(floorTimerTimeoutCH chan bool) {
	floorTimerTimeoutCH <- true
}

func StopFloorTimer(floorTimerStopCH chan bool, floorTimerTimeoutCH chan bool) {
	floorTimerStopCH <- true
	//Signal that we do not currently have a FloorTimerTimeout:
	floorTimerTimeoutCH <- false
}
