package main

import(
	"network-go/network/bcast"
	"fmt"
	"log"
)

type NetWorkHandler struct {
	 myId int
	 sendChan chan NetworkMessage
	 receiveChan chan NetworkMessage
}

func (nh *NetWorkHandler) Send(msg NetworkMessage) {
	nh.sendChan <- msg
}

func (nh *NetWorkHandler) Receive() NetworkMessage {
	return <- nh.receiveChan
}

func (nh *NetWorkHandler) GetId() int