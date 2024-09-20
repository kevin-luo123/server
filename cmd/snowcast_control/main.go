package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) != 4 {
		log.Printf("Usage:  %s <server IP> <server port> <listener port>", os.Args[0])
		return
	}
	serverIP := os.Args[1]
	serverPort := os.Args[2]
	listenerPort, err := strconv.Atoi(os.Args[3])
	if err != nil || listenerPort < 1024 || listenerPort > 65535 {
		log.Printf("listener port invalid")
		return
	}
	addr := fmt.Sprintf("%s:%s", serverIP, serverPort)
	conn, err := net.Dial("tcp4", addr)
	if err != nil {
		log.Printf("failed to connect to server")
		return
	}
	defer conn.Close()

	hello := make([]byte, 3)
	hello[0] = 0                                                //commandType
	binary.BigEndian.PutUint16(hello[1:], uint16(listenerPort)) //udpPort
	conn.Write(hello)                                           //saying hello

	welcome := make([]byte, 3)
	n, err := conn.Read(welcome) //welcome from server
	if err != nil || n != 3 || welcome[0] != 2 {
		log.Printf("welcome message failed")
		return
	}
	num_stations := binary.BigEndian.Uint16(welcome[1:])

	fmt.Println("Welcome to Snowcast! The server has " + strconv.FormatUint(uint64(num_stations), 10) + " stations")

	var response string
	fmt.Scanln(&response)
	station, err := strconv.Atoi(response)
	if err != nil {
		log.Printf("quitting")
		return
	}

	set_station := make([]byte, 3)
	set_station[0] = 1
	binary.BigEndian.PutUint16(set_station[1:], uint16(station))
	conn.Write(set_station)

	announce := make([]byte, 2)
	n, err = conn.Read(announce)
	if err != nil || n != 2 || announce[0] != 3 {
		log.Printf("announcement failed")
		return
	}
	song_name_size := announce[1]
	song_name := make([]byte, song_name_size)
	n, err = conn.Read(song_name)
	if err != nil {
		log.Printf("announcement failed")
		return
	}
}
