package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

/* A Simple function to verify error */
func CheckError(err error) {
	if err != nil {
		fmt.Println("Error: ", err)
		os.Exit(0)
	}
}

func MessageReceiver(conn *net.UDPConn) {
	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		fmt.Println("Received: ", string(buf[0:n]), " from ", addr)

		if err != nil {
			fmt.Println("Error: ", err)
		}
	}
}
func MessageSender(src *net.UDPAddr, dst *net.UDPAddr) {

	Conn, err := net.DialUDP("udp", src, dst)
	CheckError(err)

	defer Conn.Close()

	i := 0
	for {
		msg := fmt.Sprintf("Hi - %s", strconv.Itoa(i))
		i++
		buf := []byte(msg)
		_, err := Conn.Write(buf)
		if err != nil {
			fmt.Println(msg, err)
		}
		time.Sleep(time.Second * 2)
	}
}

func main() {
	/* Lets prepare a address at any address at port 10001*/
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	CheckError(err)

	/* Now listen at selected port */
	serverConn, err := net.ListenUDP("udp", serverAddr)
	CheckError(err)
	defer serverConn.Close()

	go MessageReceiver(serverConn)

	//dstAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:10001")
	//CheckError(err)

	srcAddr, err := net.ResolveUDPAddr("udp", "127.0.0.2:10002")
	CheckError(err)

	go MessageSender(srcAddr, serverAddr)

	for {
		time.Sleep(time.Millisecond * 1)
	}

}
