package main

import (
	"fmt"
)

func main() {

	server_IP, err := discover_server()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Server IP Address: %s\n", server_IP)

	conn, err := connect_TCP_server(server_IP)
	if err != nil {
		panic(err)
	}
	fmt.Println("TCP communication with server completed successfully")

	TCP_send_message(conn, "Hei pa deg")
	response, err := TCP_receive_message(conn)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Received from server: %s\n", response)
}
