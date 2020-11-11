package internal

import (
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

/* A Simple function to verify error */
func CheckError(err error) {
	if err != nil {
		log.Println("Error: ", err)
		os.Exit(0)
	}
}

func MessageCatcher(conn *net.UDPConn) {
	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		log.Printf("%d Received: %s from %s", n, string(buf[0:n]), addr)

		if err != nil {
			log.Println("Error: ", err)
		}
	}
}

func PitcherAndCatcher(CBNet *poc_cb_net.CBNetwork, channel chan bool) {

	fmt.Println("Blocked till Networking Rule setup")
	<-channel

	time.Sleep(time.Second * 3)
	fmt.Println("Start PitcherAndCatcher")

	rule := &CBNet.NetworkingRule
	// Catcher
	// Prepare a server address at any address at port 10001
	serverAddr, err := net.ResolveUDPAddr("udp", ":10001")
	CheckError(err)

	// Listen at selected port
	serverConn, err := net.ListenUDP("udp", serverAddr)
	CheckError(err)

	defer serverConn.Close()

	// Run Catcher
	go MessageCatcher(serverConn)

	// Pitcher
	// Pitch massage every 2second
	for {
		// Read rule
		// Pitch to everybody (Broadcast) every 2second
		time.Sleep(time.Second * 2)
		for index, _ := range rule.ID {
			// Slow down
			time.Sleep(time.Millisecond * 10)

			// Get source(local) and destination(remote) in rules
			src := rule.CBNetIP[index]
			des := rule.PublicIP[index]
			
			// Skip self pitching
			if des == CBNet.MyPublicIP {
				log.Println("It's mine. Continue")
				continue
			}
			log.Printf("Source: %s,	Destination: %s", src, des)

			//srcAddr, err := net.ResolveUDPAddr("udp", fmt.Sprint(src, ":10002"))
			//CheckError(err)
			desAddr, err := net.ResolveUDPAddr("udp", fmt.Sprint(des, ":10001"))
			CheckError(err)

			// Create connection
			Conn, err := net.DialUDP("udp", nil, desAddr)
			CheckError(err)

			defer Conn.Close()

			// Create message
			msg := fmt.Sprintf("Hi (from %s)", src)

			buf := []byte(msg)

			n, err := Conn.Write(buf)
			if err != nil {
				log.Printf("Error message: %s, (%s(%d))\n", err, msg, n)
			}
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
