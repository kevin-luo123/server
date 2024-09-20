package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
)

var num_to_client map[int]ClientInfo
var station_to_nums map[uint16]map[int]struct{}
var next_client_num int
var client_count_mutex sync.Mutex
var station_mutex sync.Mutex
var station_count uint16
var stations []string

type ClientInfo struct {
	connection *net.TCPConn
	udp_port   uint16
	station    uint16
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <listen port> <file0> [file 1] [file 2] ...", os.Args[0])
	}
	//listen on port 16800, files are mp3
	stations = os.Args[2:]
	station_count = uint16(len(stations))
	port := fmt.Sprintf(":%s", os.Args[1])
	addr, err := net.ResolveTCPAddr("tcp4", port)
	if err != nil {
		log.Fatal(err)
	}

	list, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Fatal(err)
	}

	defer list.Close()
	next_client_num = 0
	for {
		tcp_conn, err := list.AcceptTCP()
		if err != nil {
			log.Println("connection failed to establish", err)
			continue
		}
		go handle_Conn(tcp_conn)
	}
}

func handle_Conn(conn *net.TCPConn) {
	defer conn.Close()

	client := ClientInfo{
		connection: conn,
		udp_port:   0,
		station:    station_count,
	}
	client_count_mutex.Lock()
	client_num := next_client_num
	num_to_client[client_num] = client
	next_client_num += 1
	client_count_mutex.Unlock()

	for {
		message := make([]byte, 3)
		_, err := conn.Read(message)
		if err != nil || message[0] == 5 {
			log.Printf("client connection closed")
			return
		}
		if message[0] == 0 { //responding to hello
			if client.udp_port != 0 {
				invalid := make([]byte, 20)
				invalid[0] = 4
				invalid[1] = 18
				copy(invalid[2:], "hello already sent")
				conn.Write(invalid)
			} else {
				client.udp_port = binary.BigEndian.Uint16(message[1:])
				welcome := make([]byte, 3)
				welcome[0] = 2
				binary.BigEndian.PutUint16(welcome[1:], station_count) //numStations
				conn.Write(welcome)
			}
		} else if message[0] == 1 { //setting station
			station_number := binary.BigEndian.Uint16(message[1:])
			if client.udp_port == 0 {
				invalid := make([]byte, 20)
				invalid[0] = 4
				invalid[1] = 18
				copy(invalid[2:], "hello not sent yet")
				conn.Write(invalid)
			} else if station_number < 0 || station_number >= station_count {
				invalid := make([]byte, 24)
				invalid[0] = 4
				invalid[1] = 22
				copy(invalid[2:], "invalid station number")
				conn.Write(invalid)
			} else {
				station_number := binary.BigEndian.Uint16(message[1:])
				if client.station == station_count { //first time
					station_mutex.Lock()
					station_to_nums[station_number][client_num] = struct{}{}
					station_mutex.Unlock()
				} else if client.station != station_number { //change station
					station_mutex.Lock()
					station_to_nums[station_number][client_num] = struct{}{}
					delete(station_to_nums[client.station], client_num)
					station_mutex.Unlock()
				}
				announce := make([]byte, 2+len(stations[station_number]))
				announce[0] = 3
				announce[1] = uint8(len(stations[station_number]))
				copy(announce[2:], stations[station_number])
				client.station = station_number
				conn.Write(announce)
			}
		} else {
			invalid := make([]byte, 22)
			invalid[0] = 4
			invalid[1] = 20
			copy(invalid[2:], "unknown command sent")
			conn.Write(invalid)
		}
	}
}
