package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"sync"
	"time"
)

var num_to_client map[int]ClientInfo
var station_to_nums map[uint16]map[int]struct{}
var next_client_num int
var station_count uint16
var stations []string

var num_to_client_mutex sync.Mutex
var station_to_nums_mutex sync.Mutex
var client_count_mutex sync.Mutex

const chunk_size = 160 //160 bytes, sent 100 times a second, 16 kB/s

type ClientInfo struct {
	connection     *net.TCPConn
	udp_connection *net.UDPConn
	udp_port       uint16
	station        uint16
}

// handle timeouts
func main() {
	if len(os.Args) < 3 {
		log.Printf("Usage: %s <tcp port> <file0> [file 1] [file 2] ...", os.Args[0])
		return
	}
	//process arguments, set up connection
	stations = os.Args[2:]
	station_count = uint16(len(stations))
	port := fmt.Sprintf(":%s", os.Args[1])
	addr, err := net.ResolveTCPAddr("tcp4", port)
	if err != nil {
		log.Printf("could nto resolve tcp address")
		return
	}
	list, err := net.ListenTCP("tcp4", addr)
	if err != nil {
		log.Printf("listening failed")
		return
	}
	defer list.Close()

	next_client_num = 0
	go wait_for_connections(list)

	//start streaming music
	for idx := range stations {
		file, err := os.Open(stations[idx])
		if err != nil {
			log.Printf("invalid file: " + stations[idx])
			return
		}
		go stream(file, uint16(idx))
		defer file.Close()
	}

	//respond to user input
	for {
		var input string
		fmt.Scanln(&input)
		re := regexp.MustCompile(`^p\s+(.+)$`)
		matches := re.FindStringSubmatch(input)
		if input == "q" { //quit
			for _, client_info := range num_to_client {
				message := make([]byte, 1)
				message[0] = 9 //random number, make client quit
				client_info.connection.Write(message)
				client_info.connection.Close()
			}
			return
		} else if input == "p" {
			station_to_nums_mutex.Lock()
			for id := 0; id < len(stations); id++ {
				list := ""
				for num, _ := range station_to_nums[uint16(id)] {
					list += ",127.0.0.1:" + string(num_to_client[num].udp_port)
				}
				fmt.Println(string(id) + "," + stations[id] + list)
			}
			station_to_nums_mutex.Unlock()
		} else if len(matches) > 1 { //write station info to file
			file, err := os.Create(matches[1])
			if err != nil {
				log.Printf("invalid file")
			} else {
				defer file.Close()
				content := ""
				station_to_nums_mutex.Lock()
				for id := 0; id < len(stations); id++ {
					list := ""
					for num, _ := range station_to_nums[uint16(id)] {
						list += ",127.0.0.1:" + string(num_to_client[num].udp_port)
					}
					content += (string(id) + "," + stations[id] + list + "\n")
				}
				station_to_nums_mutex.Unlock()
				_, err = file.WriteString(content)
				if err != nil {
					log.Printf("could not copy to file")
				}
			}
		} else {
			log.Printf("invalid command")
		}
	}
}

func stream(file *os.File, song_idx uint16) {
	reader := bufio.NewReader(file)
	buffer := make([]byte, chunk_size)
	for {
		//load song data into buffer
		start := time.Now()
		n, err := reader.Read(buffer)
		if err == io.EOF {
			_, err = file.Seek(0, io.SeekStart)
			if err != nil {
				log.Println("Station failed to loop: " + stations[song_idx])
				return
			}
			reader = bufio.NewReader(file)
			_, err = reader.Read(buffer[n:])
			if err != nil {
				log.Println("Station failed to restart: " + stations[song_idx])
			}
		} else if err != nil {
			log.Println("Station failed to play: " + stations[song_idx])
		}

		//play to all clients listening to station
		station_to_nums_mutex.Lock()
		if clients, exists := station_to_nums[song_idx]; exists {
			for client_num, _ := range clients {
				//no mutex needed because we are reading something that is never changed
				_, err = num_to_client[client_num].udp_connection.Write(buffer)
				if err != nil {
					log.Println("Failed to send data to client number " + string(client_num))
					return
				}
			}
		}
		station_to_nums_mutex.Unlock()
		stop := time.Now()
		time.Sleep(10*time.Millisecond - stop.Sub(start))
	}
}

func wait_for_connections(list *net.TCPListener) {
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

	//client struct instantiated
	client := ClientInfo{
		connection:     conn,
		udp_port:       0,
		station:        station_count,
		udp_connection: nil,
	}

	//setting up client num and mapping from num to struct
	client_count_mutex.Lock()
	client_num := next_client_num
	next_client_num += 1
	client_count_mutex.Unlock()
	num_to_client_mutex.Lock()
	num_to_client[client_num] = client
	num_to_client_mutex.Unlock()

	for {
		message := make([]byte, 3)
		_, err := conn.Read(message)
		if err != nil || message[0] == 5 { //read failed or client tells server it's quitting
			log.Printf("client connection closed")
			clean(client_num)
			return
		}
		if message[0] == 0 { //hello
			if client.udp_port != 0 { //multiple hellos sent
				invalid_command(1, conn)
				clean(client_num)
				return
			} else { //respond with welcome message
				client.udp_port = binary.BigEndian.Uint16(message[1:])

				//set up udp connection
				udp_addr := fmt.Sprintf("127.0.0.1:%s", client.udp_port)
				addr, err := net.ResolveUDPAddr("udp", udp_addr)
				if err != nil {
					invalid_command(4, conn)
					clean(client_num)
					return
				}
				udp_conn, err := net.DialUDP("udp", nil, addr)
				if err != nil {
					invalid_command(4, conn)
					clean(client_num)
					return
				}
				defer udp_conn.Close()
				client.udp_connection = udp_conn

				//send welcome message
				welcome := make([]byte, 3)
				welcome[0] = 2
				binary.BigEndian.PutUint16(welcome[1:], station_count)
				conn.Write(welcome)
			}
		} else if message[0] == 1 { //set station
			station_number := binary.BigEndian.Uint16(message[1:])
			if client.udp_port == 0 {
				invalid_command(2, conn)
				clean(client_num)
				return
			} else if station_number > station_count {
				invalid_command(3, conn)
				clean(client_num)
				return
			} else { //announce the station
				if station_number != client.station { //setting new station
					station_to_nums_mutex.Lock()
					if client.station != station_number { //need to get off old station
						delete(station_to_nums[client.station], client_num)
					}
					station_to_nums[station_number][client_num] = struct{}{}
					station_to_nums_mutex.Unlock()
				}
				announce := make([]byte, 2+len(stations[station_number]))
				announce[0] = 3
				announce[1] = uint8(len(stations[station_number]))
				copy(announce[2:], stations[station_number])
				client.station = station_number
				conn.Write(announce)
			}
		} else { //unknown message
			invalid_command(4, conn)
			clean(client_num)
			return
		}
	}
}

func invalid_command(x int, conn *net.TCPConn) {
	if x == 1 { //multiple hellos
		invalid := make([]byte, 20)
		invalid[0] = 4
		invalid[1] = 18
		copy(invalid[2:], "hello already sent")
		conn.Write(invalid)
	} else if x == 2 { //set station command before sending hello
		invalid := make([]byte, 20)
		invalid[0] = 4
		invalid[1] = 18
		copy(invalid[2:], "hello not sent yet")
		conn.Write(invalid)
	} else if x == 3 { //invalid set station number
		invalid := make([]byte, 24)
		invalid[0] = 4
		invalid[1] = 22
		copy(invalid[2:], "invalid station number")
		conn.Write(invalid)
	} else if x == 4 {
		invalid := make([]byte, 12)
		invalid[0] = 4
		invalid[1] = 10
		copy(invalid[2:], "udp failed")
	} else { //unknown command sent
		invalid := make([]byte, 22)
		invalid[0] = 4
		invalid[1] = 20
		copy(invalid[2:], "unknown command sent")
		conn.Write(invalid)
	}
}

// remove client from both maps
func clean(client_num int) {
	station_to_nums_mutex.Lock()
	delete(station_to_nums[num_to_client[client_num].station], client_num)
	station_to_nums_mutex.Unlock()

	num_to_client_mutex.Lock()
	delete(num_to_client, client_num)
	num_to_client_mutex.Unlock()
}
