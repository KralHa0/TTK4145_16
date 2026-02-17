package networkhandler

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"fmt"
	"os"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

const (
	bcastPort = 30000 //Worldview port
	peersPort = 30001 //Peer discovery port
)

func NetworkInit() (
	stateTx chan def.Worldview,
	stateRx chan def.Worldview,
	peerUpdateRx chan peers.PeerUpdate,
	transmitEnable chan bool,
) {
	id := CheckIP()
	stateTx = make(chan def.Worldview)
	stateRx = make(chan def.Worldview)
	peerUpdateRx = make(chan peers.PeerUpdate)
	transmitEnable = make(chan bool)

	go bcast.Transmitter(bcastPort, stateTx)
	go bcast.Receiver(bcastPort, stateRx)
	go peers.Transmitter(peersPort, id, transmitEnable)
	go peers.Receiver(peersPort, peerUpdateRx)

	go func() { transmitEnable <- true }()

	return
}

func CheckIP() string {
	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
	}
	return fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
}

func SendWorldview(wv def.Worldview) {

}
func ReceiveWorldview() def.Worldview {
	return
}
func GetPeerUpdate() peers.PeerUpdate {

}

func DisableTransmit() {

}

func EnableTransmit() {

}
