package Statemachine

import (
	"Driver-go/elevio"
	"reflect"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
	nw "github.com/KralHa0/TTK4145_16/Project/NetworkHandler"
)

// Hardware output wrappers — state handlers call these instead of elevio directly.

func SetMotorDirection(dir elevio.MotorDirection) {
	elevio.SetMotorDirection(dir)
}

func SetFloorIndicator(floor int) {
	elevio.SetFloorIndicator(floor)
}

func SetDoorOpenLamp(value bool) {
	elevio.SetDoorOpenLamp(value)
}

func SetButtonLamp(button elevio.ButtonType, floor int, value bool) {
	elevio.SetButtonLamp(button, floor, value)
}


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


func SetAllLights(wvCh <-chan def.Worldview) {
	ID := nw.GetIp()
	var oldWv def.Worldview
	for wv := range wvCh {
		if !reflect.DeepEqual(oldWv, wv) {
			for _, node := range wv.Nodes {
				if node.ID == ID {
					checkAllLights(node, wv)
				}
			}
			oldWv = wv
		}
	}
}


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
