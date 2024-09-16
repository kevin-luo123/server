package main

import ("net"
		"log"
		"encoding/binary"
		"strconv"
		"os"
		"fmt")

func main(){
	if len(os.Args) != 4 {
		log.Fatalf("Usage:  %s <server IP> <server port> <listener port>", os.Args[0])
	}
	serverIP := os.Args[1]
	serverPort := os.Args[2]
	listenerPort, err0 := strconv.Atoi(os.Args[3])
	if err0 != nil{
		log.Fatal(err0)
	}
	addr := fmt.Sprintf("%s:%s", serverIP, serverPort)
	conn, err1 := net.Dial("tcp4", addr)
	if err1 != nil{
		log.Fatal(err1) //change later
	}
	defer conn.Close()

	message := make([]byte, 3) //hello message
	message[0] = 0 //commandType
	binary.BigEndian.PutUint16(message[1:], uint16(listenerPort)) //udpPort
	conn.Write(message)
	
	buffer := make([]byte, 3)
	_, err2 := conn.Read(buffer)
	if err2 != nil{
		log.Fatal(err2) //change later
	}
	numStations := binary.BigEndian.Uint16(buffer[1:])

	fmt.Println("Welcome to Snowcast! The server has " + strconv.FormatUint(uint64(numStations), 10) + " stations")
}
	