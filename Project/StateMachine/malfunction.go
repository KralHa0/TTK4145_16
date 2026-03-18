package Statemachine


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
		case isStoppedFlag = <-stopMalfunctionCH:
		case isWatchdogtimeout = <-watchdogTimeoutCH:
		case isFloorTimerTimeout = <-floorTimerTimeoutCH:
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
