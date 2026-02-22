package networkhandler

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"fmt"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

const (
	bcastPort = 30000 // Worldview port
	peersPort = 30001 // Peer discovery port
)

var (
	stateTx        = make(chan def.Worldview)
	stateRx        = make(chan def.Worldview)
	peerUpdateRx   = make(chan peers.PeerUpdate)
	transmitEnable = make(chan bool)
	aliveList      = def.AliveList{
		Peers: make(map[string]bool),
	}
)

func NetworkInit() {
	id := CheckIP()

	go bcast.Transmitter(bcastPort, stateTx)
	go bcast.Receiver(bcastPort, stateRx)
	go peers.Transmitter(peersPort, id, transmitEnable)
	go peers.Receiver(peersPort, peerUpdateRx)

	go func() { transmitEnable <- true }()
}

func CheckIP() string {
	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println(err)
		localIP = "DISCONNECTED"
	}
	return fmt.Sprintf("%s", localIP)
}

func SendWorldview(wv def.Worldview) {
	stateTx <- wv
}

func ReceiveWorldview() def.Worldview {
	return <-stateRx
}

func GetPeerUpdate() peers.PeerUpdate {
	return <-peerUpdateRx
}

func DisableTransmit() {
	go func() { transmitEnable <- false }()
}

func EnableTransmit() {
	go func() { transmitEnable <- true }()
}

func UpdateAliveList(update peers.PeerUpdate) {
	if update.New != "" {
		aliveList.Peers[update.New] = true
	}
	for _, id := range update.Lost {
		aliveList.Peers[id] = false
	}
}

func GetAliveList() *def.AliveList {
	return &aliveList
}

func NetworkRun(
	localWvCh <-chan def.Worldview,
	peerWvCh chan<- def.Worldview,
	peerUpdateCh chan<- peers.PeerUpdate,
) {
	for {
		select {
		case wv := <-localWvCh:
			SendWorldview(wv)
		case wv := <-stateRx:
			peerWvCh <- wv
		case update := <-peerUpdateRx:
			peerUpdateCh <- update
		}
	}
}
