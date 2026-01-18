package main

import (
	"fmt"
	"net"
	"time"
)

func discover_server() (string, error) {
	//Bind to UDP port 20001 to listen for messages from server
	localAddr, err := net.ResolveUDPAddr("udp", ":20001")
	if err != nil {
		panic(err)
	}

	//Create UDP connection
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	//Server address
	serverAddr, err := net.ResolveUDPAddr("udp", "localhost:20000")
	if err != nil {
		panic(err)
	}

	//Send discovery message to server
	_, err = conn.WriteToUDP([]byte("Discover_server"), serverAddr)
	if err != nil {
		panic(err)
	}
	fmt.Println("Sent discovery message to server")

	//Wait for response from server
	buf := make([]byte, 1025)
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	n, from, err := conn.ReadFromUDP(buf)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Received %d bytes from %s\n", n, from.String())
	fmt.Printf("Message: %s\n", string(buf[:n]))

	//Return server IP address
	return from.IP.String(), nil
}
