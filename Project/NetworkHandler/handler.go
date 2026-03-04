package networkhandler

import (
	"Network-go/network/bcast"
	"Network-go/network/localip"
	"Network-go/network/peers"
	"fmt"
	"sync"

	def "github.com/KralHa0/TTK4145_16/Project/Definitions"
)

const (
	bcastPort = 30000
	peersPort = 30001
)

var (
	stateTx        = make(chan def.Worldview)
	stateRx        = make(chan def.Worldview)
	peerUpdateRx   = make(chan peers.PeerUpdate)
	transmitEnable = make(chan bool)

	aliveList = def.AliveList{
		Peers: make(map[string]bool),
	}
	aliveListMu sync.RWMutex

	// ip is the local node's IP — assigned in NetworkInit, never shadowed
	ip string
)

func NetworkInit() {
	// Fix: assignment not short declaration, so package-level ip is set
	ip = CheckIP()

	go bcast.Transmitter(bcastPort, stateTx)
	go bcast.Receiver(bcastPort, stateRx)
	go peers.Transmitter(peersPort, ip, transmitEnable)
	go peers.Receiver(peersPort, peerUpdateRx)
	go func() { transmitEnable <- true }()
}

func CheckIP() string {
	localIP, err := localip.LocalIP()
	if err != nil {
		fmt.Println("Warning: could not get local IP:", err)
		localIP = "DISCONNECTED"
	}
	return fmt.Sprintf("%s", localIP)
}

func GetIp() string {
	return ip
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
	aliveListMu.Lock()
	defer aliveListMu.Unlock()

	if update.New != "" {
		aliveList.Peers[update.New] = true
	}
	for _, id := range update.Lost {
		aliveList.Peers[id] = false
	}
}

func GetAliveList() def.AliveList {
	aliveListMu.RLock()
	defer aliveListMu.RUnlock()

	copyMap := make(map[string]bool)
	for k, v := range aliveList.Peers {
		copyMap[k] = v
	}
	return def.AliveList{Peers: copyMap}
}

func NetworkRun(
	localWvCh   <-chan def.Worldview,
	peerWvCh    chan<- def.Worldview,
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