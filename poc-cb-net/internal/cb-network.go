package internal

import (
	"flag"
	"fmt"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/data-objects"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	// I use TUN interface, so only plain IP packet, no ethernet header + mtu is set to 1300
	BUFFERSIZE = 1500
	MTU        = "1300"
)

var CBLogger *logrus.Logger

func init() {
	// cblog is a global variable.
	configPath := filepath.Join("..", "..", "configs", "log_conf.yaml")
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", configPath)
}

type CBNetwork struct {
	CBNet             *water.Interface           // Assigned cbnet0 IP from the server
	name              string                     // InterfaceName of CBNet, e.g., cbnet0
	port              int                        // Port used for tunneling
	MyPublicIP        string                     // Inquired public IP of VM/Host
	myPrivateNetworks []string                   // Inquired CIDR blocks of private network of VM/Host
	listenConnection  *net.UDPConn               // Connection for encapsulation and decapsulation
	NetworkingRule    dataobjects.NetworkingRule // Networking rule for CBNet and tunneling
	isRunning         bool

	NetworkInterfaces []dataobjects.NetworkInterface // To be Deprecated
}

// Constructor
func NewCBNetwork(name string, port int) *CBNetwork {
	CBLogger.Debug("Start.........")

	temp := &CBNetwork{name: name, port: port}
	temp.isRunning = false
	temp.inquiryVMPublicIP()
	temp.UpdateNetworkInterfaceInfo() // To be deprecated and update "updateCIDRBlocksOfPrivateNetwork"
	temp.updateCIDRBlocksOfPrivateNetwork()

	CBLogger.Debug("End.........")
	return temp
}

func (cbnet *CBNetwork) inquiryVMPublicIP() {
	CBLogger.Debug("Start.........")

	resp, err := http.Get("https://ifconfig.co/")
	if err != nil {
		CBLogger.Panic(err)
	}

	defer resp.Body.Close()

	// 결과 출력
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		CBLogger.Panic(err)
	}
	CBLogger.Tracef("%s\n", string(data))

	cbnet.MyPublicIP = string(data[:len(data)-1]) // Remove last '\n'

	CBLogger.Debug("End.........")
}

func (cbnet *CBNetwork) updateCIDRBlocksOfPrivateNetwork() {
	CBLogger.Debug("Start.........")

	privateIPChecker := NewPrivateIPChecker()

	// Explore network interfaces
	for _, networkInterface := range cbnet.NetworkInterfaces {
		CBLogger.Trace(networkInterface)
		// Explore IPs
		for _, IP := range networkInterface.IPs {
			isPrivateIP := privateIPChecker.IsPrivateIP(net.ParseIP(IP.IPAddress))
			// Is private IP ?
			if isPrivateIP {
				if IP.Version == "IPv4" { // Is IPv4 ?
					cbnet.myPrivateNetworks = append(cbnet.myPrivateNetworks, IP.CIDRBlock)
					CBLogger.Tracef("True v4 %s, %s\n", IP.IPAddress, IP.CIDRBlock)
				} else if IP.Version == "IPv6" { // Is IPv6 ?
					CBLogger.Tracef("True v6 %s, %s\n", IP.IPAddress, IP.CIDRBlock)
				} else { // Unknown version
					CBLogger.Trace("!!! Unknown version !!!")
				}
			} else {
				CBLogger.Tracef("PublicIP %s, %s\n", IP.IPAddress, IP.CIDRBlock)
			}
		}
	}

	CBLogger.Debug("End.........")
}

//func (self CBNetworkAgent) GetCIDRBlocksOfPrivateNetwork() []string {
//	return self.myPrivateNetworks
//}

func (cbnet CBNetwork) GetVMNetworkInformation() dataobjects.VMNetworkInformation {
	CBLogger.Debug("Start.........")

	temp := dataobjects.VMNetworkInformation{
		PublicIP:        cbnet.MyPublicIP,
		PrivateNetworks: cbnet.myPrivateNetworks,
	}
	CBLogger.Trace(temp)

	CBLogger.Debug("End.........")
	return temp
}

func (cbnet *CBNetwork) SetNetworkingRule(rule dataobjects.NetworkingRule) {
	CBLogger.Debug("Start.........")

	cbnet.NetworkingRule = rule

	CBLogger.Debug("End.........")
}

func (cbnet *CBNetwork) initCBNet() {
	CBLogger.Debug("Start.........")

	idx := cbnet.NetworkingRule.GetIndexOfPublicIP(cbnet.MyPublicIP)
	localNetwork := cbnet.NetworkingRule.CBNet[idx]

	localIP := flag.String("local", localNetwork, "Local tun interface IP/MASK like 192.168.3.3⁄24")
	if "" == *localIP {
		flag.Usage()
		CBLogger.Fatal("local ip is not specified")
	}

	iface, err := water.New(water.Config{
		DeviceType:             water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{Name: cbnet.name},
	})
	if nil != err {
		CBLogger.Fatal("Unable to allocate TUN interface:", err)
	}
	CBLogger.Info("Interface allocated:", iface.Name())

	cbnet.CBNet = iface
	CBLogger.Trace("=== *cbnet.CBNet: ", *cbnet.CBNet)
	CBLogger.Trace("=== cbnet.CBNet: ",cbnet.CBNet)

	// Set interface parameters
	cbnet.runIP("link", "set", "dev", cbnet.CBNet.Name(), "mtu", MTU)
	cbnet.runIP("addr", "add", *localIP, "dev", cbnet.CBNet.Name())
	cbnet.runIP("link", "set", "dev", cbnet.CBNet.Name(), "up")

	CBLogger.Debug("End.........")
}

func (cbnet *CBNetwork) runIP(args ...string) {
	CBLogger.Debug("Start.........")

	CBLogger.Trace(args)

	cmd := exec.Command("/sbin/ip", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if nil != err {
		CBLogger.Fatal("Error running /sbin/ip:", err)
	}

	CBLogger.Debug("End.........")
}

func (cbnet CBNetwork) IsRunning() bool {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("IsRunning? ", cbnet.isRunning)

	CBLogger.Debug("End.........")
	return cbnet.isRunning
}

func (cbnet *CBNetwork) StartCBNetworking(channel chan bool) {
	CBLogger.Debug("Start.........")

	CBLogger.Info("Run CBNetworking between VMs")

	cbnet.initCBNet()
	channel <- true
	cbnet.isRunning = true

	CBLogger.Debug("End.........")
}

func (cbnet *CBNetwork) RunDecapsulation(channel chan bool) {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("Blocked till Networking Rule setup")
	<-channel

	CBLogger.Debug("Start decapsulation")
	// Decapsulation

	// Listen to local socket
	// Create network address to listen
	lstnAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%v", cbnet.port))
	if nil != err {
		CBLogger.Fatal("Unable to get UDP socket:", err)
	}

	// Create connection to network address
	lstnConn, err := net.ListenUDP("udp", lstnAddr)
	if nil != err {
		CBLogger.Fatal("Unable to listen on UDP socket:", err)
	}
	defer lstnConn.Close()

	buf := make([]byte, BUFFERSIZE)
	for {

		// ReadFromUDP acts like ReadFrom but returns a UDPAddr.
		n, addr, err := lstnConn.ReadFromUDP(buf)
		if err != nil {
			CBLogger.Debug("Error in cbnet.listenConnection.ReadFromUDP(buf): ", err)
		}

		// Parse header
		header, _ := ipv4.ParseHeader(buf[:n])
		CBLogger.Debugf("Received %d bytes from %v: %+v\n", n, addr, header)

		// It might be necessary to handle or route packets to the specific destination
		// based on the NetworkingRule table
		// To be determined.

		// Write to TUN interface
		nWrite, errWrite := cbnet.CBNet.Write(buf[:n])
		if errWrite != nil || nWrite == 0 {
			CBLogger.Debugf("Error(%d len): %s", nWrite, errWrite)
		}
	}
}

func (cbnet *CBNetwork) RunEncapsulation(channel chan bool) {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("Blocked till Networking Rule setup")
	<-channel

	CBLogger.Debug("Start encapsulation")
	packet := make([]byte, BUFFERSIZE)
	for {

		// Read packet from CBNet interface "cbnet0"
		//fmt.Println("=== *cbnet.CBNet: ", *cbnet.CBNet)
		//fmt.Println("=== cbnet.CBNet: ",cbnet.CBNet)
		plen, err := cbnet.CBNet.Read(packet)
		if err != nil {
			CBLogger.Error("Error Read() in encapsulation:", err)
		}

		// Parse header
		header, err := ipv4.ParseHeader(packet[:plen])
		CBLogger.Tracef("Sending to remote: %+v (%+v)\n", header, err)

		// Search and change destination (Public IP of target VM)
		idx := cbnet.NetworkingRule.GetIndexOfCBNetIP(header.Dst.String())

		var remoteIP string
		if idx != -1 {
			remoteIP = cbnet.NetworkingRule.PublicIP[idx]
		}

		// Resolve remote addr
		remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", remoteIP, cbnet.port))
		if nil != err {
			CBLogger.Fatal("Unable to resolve remote addr:", err)
		}

		// Send packet
		nWriteToUDP, errWriteToUDP := cbnet.listenConnection.WriteToUDP(packet[:plen], remoteAddr)
		if errWriteToUDP != nil || nWriteToUDP == 0 {
			CBLogger.Fatalf("Error(%d len): %s", nWriteToUDP, errWriteToUDP)
		}
	}

}

func (cbnet *CBNetwork) RunTunneling(channel chan bool) {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("Blocked till Networking Rule setup")
	<-channel

	CBLogger.Debug("Start decapsulation")
	// Decapsulation

	// Listen to local socket
	// Create network address to listen
	lstnAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%v", cbnet.port))
	if nil != err {
		CBLogger.Fatal("Unable to get UDP socket:", err)
	}

	// Create connection to network address
	lstnConn, err := net.ListenUDP("udp", lstnAddr)
	if nil != err {
		CBLogger.Fatal("Unable to listen on UDP socket:", err)
	}
	defer lstnConn.Close()

	go func() {
		buf := make([]byte, BUFFERSIZE)
		for {
			// ReadFromUDP acts like ReadFrom but returns a UDPAddr.
			n, _, err := lstnConn.ReadFromUDP(buf)
			if err != nil {
				CBLogger.Error("Error in cbnet.listenConnection.ReadFromUDP(buf): ", err)
			}

			// Parse header
			header, _ := ipv4.ParseHeader(buf[:n])
			CBLogger.Tracef("Header received: %+v\n", header)
			//fmt.Printf("Received %d bytes from %v: %+v\n", n, addr, header)

			// It might be necessary to handle or route packets to the specific destination
			// based on the NetworkingRule table
			// To be determined.

			// Write to TUN interface
			nWrite, errWrite := cbnet.CBNet.Write(buf[:n])
			if errWrite != nil || nWrite == 0 {
				CBLogger.Errorf("Error(%d len): %s", nWrite, errWrite)
			}
		}
	}()

	CBLogger.Debug("Start encapsulation")
	// Encapsulation
	packet := make([]byte, BUFFERSIZE)
	for {

		// Read packet from CBNet interface "cbnet0"
		//fmt.Println("=== *cbnet.CBNet: ", *cbnet.CBNet)
		//fmt.Println("=== cbnet.CBNet: ",cbnet.CBNet)
		plen, err := cbnet.CBNet.Read(packet)
		if err != nil {
			CBLogger.Error("Error Read() in encapsulation:", err)
		}

		// Parse header
		header, _ := ipv4.ParseHeader(packet[:plen])
		CBLogger.Tracef("Sending to remote: %+v (%+v)\n", header, err)

		// Search and change destination (Public IP of target VM)
		idx := cbnet.NetworkingRule.GetIndexOfCBNetIP(header.Dst.String())

		var remoteIP string
		if idx != -1 {
			remoteIP = cbnet.NetworkingRule.PublicIP[idx]
		}

		// Resolve remote addr
		remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", remoteIP, cbnet.port))
		if nil != err {
			CBLogger.Fatal("Unable to resolve remote addr:", err)
		}

		// Send packet
		nWriteToUDP, errWriteToUDP := lstnConn.WriteToUDP(packet[:plen], remoteAddr)
		if errWriteToUDP != nil || nWriteToUDP == 0 {
			CBLogger.Errorf("Error(%d len): %s", nWriteToUDP, errWriteToUDP)
		}

	}

}

// To be deprecated
// Define a function to get network interfaces in a physical or virtual machine
func (cbnet *CBNetwork) UpdateNetworkInterfaceInfo() {
	CBLogger.Debug("Start.........")

	// Get network interfaces
	ifaces, _ := net.Interfaces()

	// Recursively get network interface information
	for _, iface := range ifaces {
		// Print a network interface name
		CBLogger.Debug("Interface name:", iface.Name)

		// Declare a NetworkInterface variable
		var networkInterface dataobjects.NetworkInterface

		// Assign Interface Interface Name
		networkInterface.Name = iface.Name

		// Get addresses
		addrs, _ := iface.Addrs()

		// Recursively get IP address
		for _, addr := range addrs {
			addrStr := addr.String()

			// Get IP Address and CIDRBlock ID
			ipAddr, networkID, err := net.ParseCIDR(addrStr)
			if err != nil {
				CBLogger.Fatal(err)
			}

			// Get version of IP (e.g., IPv4 or IPv6)
			var version string

			if ipAddr.To4() != nil {
				version = "IPv4"
			} else if ipAddr.To16() != nil {
				version = "IPv6"
			} else {
				version = "Unknown"
				CBLogger.Tracef("Unknown version (IPAddr: %s)\n", ipAddr.String())
			}

			// Print version, IPAddress, CIDRBlock ID
			//fmt.Println("	 Version: ", version)
			//fmt.Println("	 IPAddr: ", ipAddr)
			//fmt.Println("	 CIDRBlock: ", networkID)

			// Create IP data object
			ip := dataobjects.IP{
				Version:   version,
				IPAddress: ipAddr.String(),
				CIDRBlock: networkID.String(),
			}

			// AppendRule the IP data object to slice
			networkInterface.IPs = append(networkInterface.IPs, ip)
		}
		cbnet.NetworkInterfaces = append(cbnet.NetworkInterfaces, networkInterface)
	}
	CBLogger.Debug("End.........")
}

func (cbnet CBNetwork) GetNetworkInterfaces() []dataobjects.NetworkInterface {
	CBLogger.Debug("Start.........")

	CBLogger.Trace("cbnet.NetworkInterfaces")
	CBLogger.Trace(cbnet.NetworkInterfaces)

	CBLogger.Debug("End.........")
	return cbnet.NetworkInterfaces
}

//func main() {
//
//	temp := CBNetworkAgent{}.GetNetworkInterfaces()
//
//	fmt.Println("Print the network interfaces")
//
//	// Marshal the network interfaces
//	doc, _ := json.Marshal(temp)
//	fmt.Println(string(doc))
//
//	// Unmarshal the network interfaces
//	var temp2 []dataobjects.NetworkInterface
//
//	json.Unmarshal([]byte(doc), &temp2)
//
//	fmt.Println(temp2)
//
//	//tt, _ := json.MarshalIndent(temp, "", "   ")
//	//fmt.Println(tt)
//	//fmt.Println("=== interfaces ===")
//	//
//	//ifaces, _ := net.Interfaces()
//	//for _, iface := range ifaces {
//	//	fmt.Println("net.Interface:", iface)
//	//
//	//	addrs, _ := iface.Addrs()
//	//	for _, addr := range addrs {
//	//		addrStr := addr.String()
//	//		fmt.Println("    net.Addr: ", addr.CIDRBlock(), addrStr)
//	//
//	//		// Must drop the stuff after the slash in order to convert it to an IP instance
//	//		split := strings.Split(addrStr, "/")
//	//		addrStr0 := split[0]
//	//
//	//		// Parse the string to an IP instance
//	//		ip := net.ParseIP(addrStr0)
//	//		if ip.To4() != nil {
//	//			fmt.Println("       ", addrStr0, "is ipv4")
//	//		} else {
//	//			fmt.Println("       ", addrStr0, "is ipv6")
//	//		}
//	//		fmt.Println("       ", addrStr0, "is interface-local multicast :", ip.IsInterfaceLocalMulticast())
//	//		fmt.Println("       ", addrStr0, "is link-local multicast      :", ip.IsLinkLocalMulticast())
//	//		fmt.Println("       ", addrStr0, "is link-local unicast        :", ip.IsLinkLocalUnicast())
//	//		fmt.Println("       ", addrStr0, "is global unicast            :", ip.IsGlobalUnicast())
//	//		fmt.Println("       ", addrStr0, "is multicast                 :", ip.IsMulticast())
//	//		fmt.Println("       ", addrStr0, "is loopback                  :", ip.IsLoopback())
//	//	}
//	//}
//
//}
