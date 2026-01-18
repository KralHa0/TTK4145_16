package main

import (
	"bufio"
	"fmt"
	"net"
)

func connect_TCP_server(server_IP string) (net.Conn, error) {
	//Connect to TCP server
	address := server_IP + ":33546"

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	//defer conn.Close()

	fmt.Println("Connected to TCP server at", address)

	reader := bufio.NewReader(conn)

	//Read welcome message from server
	welcome, err := reader.ReadBytes('\x00')
	if err != nil {
		return nil, err
	}
	fmt.Printf("Received welcome message: %s\n", string(welcome[:len(welcome)-1]))

	//Send response message to server
	response := "Hello from TCP client\x00"
	_, err = conn.Write([]byte(response))
	if err != nil {
		return nil, err
	}
	fmt.Println("Sent response message to server")

	return conn, nil
}

func TCP_send_message(conn net.Conn, message string) error {
	_, err := conn.Write([]byte(message + "\x00"))
	if err != nil {
		return err
	}
	return nil
}

func TCP_receive_message(conn net.Conn) (string, error) {
	reader := bufio.NewReader(conn)
	message, err := reader.ReadBytes('\x00')
	if err != nil {
		return "", err
	}
	return string(message[:len(message)-1]), nil
}
