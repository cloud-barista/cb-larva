package app

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	cbnet "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network"
)

/* A Simple function to verify error */
func checkError(err error) {
	if err != nil {
		log.Println("Error: ", err)
		os.Exit(0)
	}
}

// MessageCatcher represents a function to receive messages continuously.
func MessageCatcher(conn *net.UDPConn) {
	buf := make([]byte, 1024)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		fmt.Printf("Received message: %s (from %s)", string(buf[0:n]), addr)

		if err != nil {
			log.Println("Error: ", err)
		}
	}
}

// PitcherAndCatcher represents a function to send messages continuously.
func PitcherAndCatcher(wg *sync.WaitGroup, CBNet *cbnet.CBNetwork, channel chan bool) {
	defer wg.Done()
	fmt.Println("Block pitching anc catching till networking rule setup")
	<-channel

	fmt.Println("Start PitcherAndCatcher after 3 seconds")
	time.Sleep(time.Second * 3)

	rule := CBNet.NetworkingRule
	index := rule.GetIndexOfPublicIP(CBNet.HostPublicIP)
	myCBNetIP := rule.HostIPAddress[index]
	// Catcher
	// Prepare a server address at any address at port 10001
	serverAddr, err := net.ResolveUDPAddr("udp", ":10001")
	checkError(err)

	// Listen at selected port
	serverConn, err := net.ListenUDP("udp", serverAddr)
	checkError(err)

	// Perform error handling
	defer func() {
		errClose := serverConn.Close()
		if errClose != nil {
			log.Fatal("can't close the file", errClose)
		}
	}()

	// Run Catcher
	go MessageCatcher(serverConn)

	// Pitcher
	// Pitch massage every 0.5 second
	for {
		// Read rule
		// Pitch to everybody (Broadcast) every 0.5 second
		time.Sleep(time.Second >> 2)
		for index := range rule.HostID {
			// Slow down
			time.Sleep(time.Millisecond * 10)

			// Get source(local) and destination(remote) in rules
			//src := rule.HostIPAddress[index]
			des := rule.PublicIPAddress[index]

			// Skip self pitching
			if des == CBNet.HostPublicIP {
				//log.Println("It's mine. Continue")
				continue
			}
			//log.Printf("Source: %s,	Destination: %s", src, des)

			//srcAddr, err := net.ResolveUDPAddr("udp", fmt.Sprint(src, ":10002"))
			//checkError(err)
			desAddr, err := net.ResolveUDPAddr("udp", fmt.Sprint(des, ":10001"))
			checkError(err)

			// Create connection
			Conn, err := net.DialUDP("udp", nil, desAddr)
			checkError(err)

			// Create message
			msg := fmt.Sprintf("Hi :D (sender: %s)", myCBNetIP)

			buf := []byte(msg)

			n, err := Conn.Write(buf)
			if err != nil {
				log.Printf("Error message: %s, (%s(%d))", err, msg, n)
			}
		}
	}
}
