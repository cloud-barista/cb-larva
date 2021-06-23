package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"os"
	"os/exec"
)

const (
	// BUFFERSIZE represents a size of read buffer.
	BUFFERSIZE = 1500
	// MTU represents a maximum transmission unit.
	MTU        = "1300"
)

func runIP(args ...string) {
	cmd := exec.Command("/sbin/ip", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if nil != err {
		log.Fatalln("Error running /sbin/ip:", err)
	}
}

// Tunneling represents thg information for tunneling.
type Tunneling struct {
	RemoteIP string `json:"RemoteIP"`
	Port     int    `json:"Port"`
}

// LoadConfig represents a function to read tunneling information from json file.
func LoadConfig() (Tunneling, error) {
	var config Tunneling
	file, err := os.Open("tunneling.json")
	// Perform error handling
	defer func() {
		errClose := file.Close()
		if errClose != nil {
			log.Fatal("can't close the file", errClose)
		}
	}()

	if err != nil {
		log.Fatal(err)
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}
	return config, err
}

func main() {

	//var CBNetAgent = poc_cb_net.NewCBNetworkAgent()
	//temp := CBNetAgent.GetNetworkInterface()

	//var localIP, remoteIP *string
	//var port *int

	//for _, networkInterface := range temp {
	//	if networkInterface.Name == "eth0" || networkInterface.Name == "ens4" {
	//		fmt.Println(networkInterface)
	//		for _, IP := range networkInterface.IPs {
	//			if IP.Version == "IPv4" {
	//				pieces := strings.Split(IP.CLADNetID, "/")
	//				prefix := pieces[1]
	//				IPAddressWithPrefix := IP.IPAddress + "/" + prefix
	//				fmt.Println(IPAddressWithPrefix)
	//				localIP = flag.String("local", IPAddressWithPrefix, "Local tun interface IP/MASK like 192.168.3.3⁄24")
	//				break
	//			}
	//		}
	//	}
	//}

	//config, err := LoadConfig()
	var (
		localIP  = flag.String("local", "192.168.7.1/24", "Local tun interface IP/MASK like 192.168.3.3⁄24")
		remoteIP = flag.String("remote", "3.128.34.227", "Remote server (external) IP like 8.8.8.8")
		port     = flag.Int("port", 20000, "UDP port for communication")
	)
	flag.Parse()

	// check if we have anything
	if "" == *localIP {
		flag.Usage()
		log.Fatalln("\nlocal ip is not specified")
	}
	if "" == *remoteIP {
		flag.Usage()
		log.Fatalln("\nremote server is not specified")
	}

	// create TUN interface
	iface, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if nil != err {
		log.Fatalln("Unable to allocate TUN interface:", err)
	}
	log.Println("Interface allocated:", iface.Name())

	// set interface parameters
	runIP("link", "set", "dev", iface.Name(), "mtu", MTU)
	runIP("addr", "add", *localIP, "dev", iface.Name())
	runIP("link", "set", "dev", iface.Name(), "up")

	// resolve remote addr
	remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", *remoteIP, *port))
	if nil != err {
		log.Fatalln("Unable to resolve remote addr:", err)
	}

	// listen to local socket
	lstnAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%v", *port))
	if nil != err {
		log.Fatalln("Unable to get UDP socket:", err)
	}
	lstnConn, err := net.ListenUDP("udp", lstnAddr)
	if nil != err {
		log.Fatalln("Unable to listen on UDP socket:", err)
	}
	defer lstnConn.Close()

	// recv in separate thread
	go func() {
		buf := make([]byte, BUFFERSIZE)
		for {
			n, addr, err := lstnConn.ReadFromUDP(buf)
			// just debug
			header, _ := ipv4.ParseHeader(buf[:n])
			fmt.Printf("Received %d bytes from %v: %+v", n, addr, header)
			if err != nil || n == 0 {
				fmt.Println("Error: ", err)
				continue
			}
			// write to TUN interface
			nWrite, errWrite := iface.Write(buf[:n])
			if errWrite != nil || nWrite == 0 {
				fmt.Printf("Error(%d len): %s", nWrite, errWrite)
			}
		}
	}()

	// and one more loop
	packet := make([]byte, BUFFERSIZE)
	for {
		plen, err := iface.Read(packet)
		if err != nil {
			break
		}
		// debug :)
		header, _ := ipv4.ParseHeader(packet[:plen])
		fmt.Printf("Sending to remote: %+v (%+v)", header, err)
		// real send
		lstnConn.WriteToUDP(packet[:plen], remoteAddr)
	}
}
