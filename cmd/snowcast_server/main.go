package main

import ("net"
		"log"
		"os"
		"fmt"
		"encoding/binary")

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: %s <listen port> <file0> [file 1] [file 2] ...", os.Args[0])
	}
	port := fmt.Sprintf(":%s", os.Args[1])
	addr, err := net.ResolveTCPAddr("tcp4", port)
	if err != nil {
		log.Fatal(err)
	}

	list, err := net.ListenTCP("tcp4", addr)
	if err != nil{
		log.Fatal(err)
	}

	defer list.Close()

	for {
		tcp_conn, err := list.AcceptTCP()
		if err != nil{
			log.Println("connection failed to establish", err)
			continue
		} 
		go handle_Conn(tcp_conn)
	}
}

func handle_Conn(conn *net.TCPConn){
	defer conn.Close()
	buffer := make([]byte, 3)
	_, err := conn.Read(buffer)
	if err != nil{
		log.Fatal(err) //change later
	}

	message := make([]byte, 3) //welcome message
	message[0] = 2 //replyType
	binary.BigEndian.PutUint16(message[1:], uint16(len(os.Args[2:]))) //numStations
	conn.Write(message)
}