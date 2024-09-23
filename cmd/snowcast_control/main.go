package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

var station_set bool
var hello_sent bool
var welcome_received bool
var num_stations uint16
var quitting = false

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
		log.Println("listener port invalid")
		return
	}
	addr := fmt.Sprintf("%s:%s", serverIP, serverPort)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println("failed to connect to server")
		return
	}
	defer conn.Close()

	//set up states
	station_set = false
	hello_sent = false
	welcome_received = false

	//start handshake, send hello to server
	hello := make([]byte, 3)
	hello[0] = 0
	binary.BigEndian.PutUint16(hello[1:], uint16(listenerPort))
	conn.Write(hello)
	set_deadline(conn)
	hello_sent = true
	for {
		message_type := make([]byte, 1)
		_, err := conn.Read(message_type)
		if err != nil || quitting {
			end_connection(conn)
			return
		}
		set_deadline(conn)
		if message_type[0] == 2 && hello_sent && !welcome_received { //received welcome
			welcome := make([]byte, 2)
			bytes_read := 0
			for bytes_read != 2 {
				n, err := conn.Read(welcome[bytes_read:])
				if err != nil {
					end_connection(conn)
					log.Println("corrupted welcome")
					return
				}
				set_deadline(conn)
				bytes_read += n
			}
			remove_deadline(conn)
			num_stations = binary.BigEndian.Uint16(welcome)

			//print prompt for user, wait for input
			fmt.Println("Welcome to Snowcast! The server has " + fmt.Sprintf("%d", num_stations) + " stations")
			welcome_received = true
			go wait_for_input(conn)
		} else if message_type[0] == 3 && station_set { //received announce
			//reading song name length
			song_name_size := make([]byte, 1)
			n, err := conn.Read(song_name_size)
			if err != nil || n != 1 {
				end_connection(conn)
				log.Println("corrupted announcement (song name length)")
				return
			}
			set_deadline(conn)

			//reading song name
			song_name := make([]byte, song_name_size[0])
			bytes_read := 0
			for bytes_read != int(song_name_size[0]) {
				n, err := conn.Read(song_name[bytes_read:])
				if err != nil {
					end_connection(conn)
					log.Println("corrupted announce")
					return
				}
				set_deadline(conn)
				bytes_read += n
			}
			remove_deadline(conn)
			fmt.Println("New song announced: " + string(song_name))
		} else { //received invalid or unknown message, disconnect in both case
			end_connection(conn)
			log.Println("invalid use of protocol")
			return
		}
	}
}

func wait_for_input(conn net.Conn) {
	for {
		var input string
		fmt.Scanln(&input) //user input read
		log.Println("got a user input")
		if input == "q" { //user quits
			end_connection(conn)
			quitting = true
			return
		}
		station, err := strconv.Atoi(input)
		if err != nil || station < 0 || station >= int(num_stations) { //user entered a non-number of an invalid number
			log.Println("To quit, enter q. To set station, enter a number from 0 - " + fmt.Sprintf("%d", int(num_stations)-1))
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

func set_deadline(conn net.Conn) {
	deadline := time.Now().Add(100 * time.Millisecond)
	err := conn.SetReadDeadline(deadline)
	if err != nil || quitting {
		end_connection(conn)
		quitting = true
		return
	}
}

func remove_deadline(conn net.Conn) {
	err := conn.SetDeadline(time.Time{})
	if err != nil {
		end_connection(conn)
		quitting = true
		return
	}
}

func end_connection(conn net.Conn) {
	quit := make([]byte, 1)
	quit[0] = 5
	conn.Write(quit)
}
