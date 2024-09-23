package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

const chunk_size = 160

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <udp_port>", os.Args[0])
	}
	udp_port, err := strconv.Atoi(os.Args[1])
	if err != nil || udp_port < 1024 || udp_port > 65535 {
		log.Fatalf("invalid port number")
	}

	//setting up udp listen connection
	address := fmt.Sprintf("localhost:%d", udp_port)
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		log.Fatal("Error resolving address:", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal("Error creating UDP connection:", err)
	}
	defer conn.Close()

	buffer := make([]byte, chunk_size)
	//read data
	for {
		n, _, err := conn.ReadFromUDP(buffer)
		if err != nil || n != chunk_size {
			log.Fatal("Error reading from connection:", err)
		}
		fmt.Println(buffer)
	}
}
