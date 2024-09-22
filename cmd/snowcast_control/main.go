package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

var conn net.TCPConn
var station_set bool
var hello_sent bool
var welcome_received bool
var num_stations uint16

// handle accepting the annoucne whwen a song restarts
// error checking for getting welcome before hello
// error checking for getting annoucne before set station
// handle timeouts
func main() {
	if len(os.Args) != 4 {
		log.Printf("Usage:  %s <server IP> <server port> <listener port>", os.Args[0])
		return
	}

	//setting up connection
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

	//set up states
	station_set = false
	hello_sent = false
	welcome_received = false

	//wait for server messages
	go wait_for_server()

	//start handshake, send hello to server
	hello := make([]byte, 3)
	hello[0] = 0
	binary.BigEndian.PutUint16(hello[1:], uint16(listenerPort))
	conn.Write(hello)
	hello_sent = true

	//wait for user input
	go wait_for_input()
}

func wait_for_server() {
	for {
		message_type := make([]byte, 1)
		_, err := conn.Read(message_type)
		if err != nil {
			log.Fatal(err)
		} else if message_type[0] == 2 && hello_sent && !welcome_received { //received welcome
			welcome := make([]byte, 2)
			n, err := conn.Read(welcome)
			if err != nil || n != 2 {
				end_connection()
				log.Fatal("corrupted welcome")
			}
			num_stations = binary.BigEndian.Uint16(welcome)
			//print prompt for user
			fmt.Println("Welcome to Snowcast! The server has " + strconv.FormatUint(uint64(num_stations), 10) + " stations")
			welcome_received = true
		} else if message_type[0] == 3 && station_set { //received announce
			//reading server's announcement
			server_response := make([]byte, 1)
			n, err := conn.Read(server_response)
			if err != nil || n != 1 {
				end_connection()
				log.Fatal("corrupted announcement (song name length)")
			}
			//responding to valid announcement
			song_name_size := server_response[0]
			song_name := make([]byte, song_name_size)
			n, err = conn.Read(song_name)
			if n != int(song_name_size) || err != nil {
				end_connection()
				log.Fatal("corrupted announcement (song name)")
			}
			log.Printf("New song announced: " + string(song_name))
		} else { //received invalid or unknown message, disconnect in both case
			end_connection()
			log.Fatal("invalid use of protocol")
		}
	}
}

func wait_for_input() {
	for {
		var input string
		fmt.Scanln(&input) //user input read

		if input == "q" { //user quits
			end_connection()
			log.Fatal("closing connection")
		}
		station, err := strconv.Atoi(input)
		if err != nil || station < 0 || station >= int(num_stations) { //user entered a non-number of an invalid number
			log.Printf("To quit, enter q. To set station, enter a number from 0 - " + strconv.Itoa(int(num_stations)-1))
		} else { //user entered a valid station
			//sending set station message to server
			set_station := make([]byte, 3)
			set_station[0] = 1
			binary.BigEndian.PutUint16(set_station[1:], uint16(station))
			conn.Write(set_station)
			station_set = true
		}
	}
}

func end_connection() {
	quit := make([]byte, 1)
	quit[0] = 5
	conn.Write(quit)
}
