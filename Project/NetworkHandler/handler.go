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
	stateTx        = make(chan def.Worldview, 10)
	stateRx        = make(chan def.Worldview)
	peerUpdateRx   = make(chan peers.PeerUpdate)
	transmitEnable = make(chan bool)

	aliveList = def.AliveList{
		Peers: make(map[def.NodeID]bool), // keyed by NodeID
	}
	aliveListMu sync.RWMutex

	ip def.NodeID // typed as NodeID — invalid state is detectable via IsValid()
)

func NetworkInit() {
	ip = def.NodeID(CheckIP()) // assign to package-level ip, no shadowing
	go bcast.Transmitter(bcastPort, stateTx)
	go bcast.Receiver(bcastPort, stateRx)
	go peers.Transmitter(peersPort, string(ip), transmitEnable)
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

func GetIp() def.NodeID {
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
		aliveList.Peers[def.NodeID(update.New)] = true
	}
	for _, id := range update.Lost {
		aliveList.Peers[def.NodeID(id)] = false
	}
}

func GetAliveList() def.AliveList {
	aliveListMu.RLock()
	defer aliveListMu.RUnlock()

	copyMap := make(map[def.NodeID]bool)
	for k, v := range aliveList.Peers {
		copyMap[k] = v
	}
	return def.AliveList{Peers: copyMap}
}

func NetworkRun(
	localWvCh    <-chan def.Worldview,
	peerWvCh     chan<- def.Worldview,
	peerUpdateCh chan<- peers.PeerUpdate,
) {
	for {
		select {
		case wv := <-localWvCh:
			SendWorldview(wv)
		case wv := <-stateRx:
			select {
			case peerWvCh <- wv:
			default:
			}
		case update := <-peerUpdateRx:
			select {
			case peerUpdateCh <- update:
			default:
			}
		}
	}
}