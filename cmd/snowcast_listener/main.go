package main

import (
	"log"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <udp_port>", os.Args[0])
	}
	udp_port, err := strconv.Atoi(os.Args[0])
	if err != nil || udp_port < 1024 || udp_port > 65535 {
		log.Fatalf("invalid port number")
	}
}
