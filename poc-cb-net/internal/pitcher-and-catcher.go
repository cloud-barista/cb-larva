package internal

import (
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net"
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

func MessageCatcher(conn *net.UDPConn) {
	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		fmt.Println("Received: ", string(buf[0:n]), " from ", addr)

		if err != nil {
			fmt.Println("Error: ", err)
		}
	}
}

func PitcherAndCatcher(CBNet *poc_cb_net.CBNetwork, channel chan bool) {

	fmt.Println("Blocked till Networking Rule setup")
	<-channel

	rule := CBNet.NetworkingRule
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
		// Pitch to everybody (Broadcast)
		for index, _ := range CBNet.NetworkingRule.ID {
			// Slow down
			time.Sleep(time.Millisecond * 5)

			// Get source(local) and destination(remote) in rules
			src := rule.CBNetIP[index]
			des := rule.PublicIP[index]
			
			// Skip self pitching
			if des == CBNet.MyPublicIP {
				continue
			}

			srcAddr, err := net.ResolveUDPAddr("udp", src)
			CheckError(err)
			desAddr, err := net.ResolveUDPAddr("udp", fmt.Sprint(des, ":10001"))
			CheckError(err)

			// Create connection
			Conn, err := net.DialUDP("udp", srcAddr, desAddr)
			CheckError(err)

			defer Conn.Close()

			// Create message
			msg := fmt.Sprintf("Hi (from %s)", src)

			buf := []byte(msg)

			n, err := Conn.Write(buf)
			if err != nil {
				fmt.Printf("Error message: %s, (%s(%d)) ", err, msg, n)
			}
		}
		time.Sleep(time.Second * 2)
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
