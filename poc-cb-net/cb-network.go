package poc_cb_net

import (
	"flag"
	"fmt"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/data-objects"
	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
)

const (
	// I use TUN interface, so only plain IP packet, no ethernet header + mtu is set to 1300
	BUFFERSIZE = 1500
	MTU        = "1300"
)

type CBNetwork struct {
	CBNet             *water.Interface           // Assigned cbnet0 IP from the server
	name              string                     // Name of CBNet, e.g., cbnet0
	port              int                        // Port used for tunneling
	myPublicIP        string                     // Inquired public IP of VM/Host
	myPrivateNetworks []string                   // Inquired CIDR blocks of private network of VM/Host
	networkingRule    dataobjects.NetworkingRule // Networking rule for CBNet and tunneling
	isRunning         bool

	NetworkInterfaces []dataobjects.NetworkInterface // To be Deprecated
}

// Constructor
func NewCBNetwork(name string, port int) *CBNetwork {
	temp := &CBNetwork{name: name, port: port}
	temp.isRunning = false
	temp.inquiryVMPublicIP()
	temp.UpdateNetworkInterfaceInfo() // To be deprecated and update "updateCIDRBlocksOfPrivateNetwork"
	temp.updateCIDRBlocksOfPrivateNetwork()

	return temp
}

func (cbnet *CBNetwork) inquiryVMPublicIP() {
	resp, err := http.Get("http://icanhazip.com/")
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	// 결과 출력
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	//fmt.Printf("%s\n", string(data))

	cbnet.myPublicIP = string(data)
}

func (cbnet *CBNetwork) updateCIDRBlocksOfPrivateNetwork() {

	privateIPChecker := NewPrivateIPChecker()

	// Explore network interfaces
	for _, networkInterface := range cbnet.NetworkInterfaces {
		//fmt.Println(networkInterface)
		// Explore IPs
		for _, IP := range networkInterface.IPs {
			isPrivateIP := privateIPChecker.IsPrivateIP(net.ParseIP(IP.IPAddress))
			// Is private IP ?
			if isPrivateIP {
				if IP.Version == "IPv4" { // Is IPv4 ?
					cbnet.myPrivateNetworks = append(cbnet.myPrivateNetworks, IP.CIDRBlock)
					//fmt.Printf("True v4 %s, %s\n", IP.IPAddress, IP.CIDRBlock)
				} else if IP.Version == "IPv6" { // Is IPv6 ?
					//fmt.Printf("True v6 %s, %s\n", IP.IPAddress, IP.CIDRBlock)
				} else { // Unknown version
					//fmt.Printf("!!! Unknown version !!!")
				}
			} else {
				//fmt.Printf("PublicIP %s, %s\n", IP.IPAddress, IP.CIDRBlock)
			}
		}
	}
}

//func (self CBNetworkAgent) GetCIDRBlocksOfPrivateNetwork() []string {
//	return self.myPrivateNetworks
//}

func (cbnet CBNetwork) GetVMNetworkInformation() dataobjects.VMNetworkInformation {
	return dataobjects.VMNetworkInformation{
		PublicIP:        cbnet.myPublicIP,
		PrivateNetworks: cbnet.myPrivateNetworks,
	}
}

func (cbnet *CBNetwork) SetNetworkingRule(rule dataobjects.NetworkingRule) {
	cbnet.networkingRule = rule
}

func (cbnet *CBNetwork) initCBNet() {

	idx := cbnet.networkingRule.GetIndexOfPublicIP(cbnet.myPublicIP)
	localNetwork := cbnet.networkingRule.CBNet[idx]

	localIP := flag.String("local", localNetwork, "Local tun interface IP/MASK like 192.168.3.3⁄24")
	if "" == *localIP {
		flag.Usage()
		log.Fatalln("\nlocal ip is not specified")
	}

	iface, err := water.New(water.Config{
		DeviceType:             water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{InterfaceName: cbnet.name},
	})
	if nil != err {
		log.Fatalln("Unable to allocate TUN interface:", err)
	}
	log.Println("Interface allocated:", iface.Name())

	cbnet.CBNet = iface

	// Set interface parameters
	cbnet.runIP("link", "set", "dev", cbnet.CBNet.Name(), "mtu", MTU)
	cbnet.runIP("addr", "add", *localIP, "dev", cbnet.CBNet.Name())
	cbnet.runIP("link", "set", "dev", cbnet.CBNet.Name(), "up")
}

func (cbnet *CBNetwork) runIP(args ...string) {
	cmd := exec.Command("/sbin/ip", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if nil != err {
		log.Fatalln("Error running /sbin/ip:", err)
	}
}
func (cbnet CBNetwork) IsRunning() bool {
	return cbnet.isRunning
}

func (cbnet *CBNetwork) RunCBNetwork() {
	cbnet.isRunning = true
	// listen to local socket
	lstnAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%v", cbnet.port))
	if nil != err {
		log.Fatalln("Unable to get UDP socket:", err)
	}
	lstnConn, err := net.ListenUDP("udp", lstnAddr)
	if nil != err {
		log.Fatalln("Unable to listen on UDP socket:", err)
	}
	defer lstnConn.Close()

	// Decapsulation
	go func() {
		buf := make([]byte, BUFFERSIZE)
		for {
			n, addr, err := lstnConn.ReadFromUDP(buf)
			// just debug
			header, _ := ipv4.ParseHeader(buf[:n])
			fmt.Printf("Received %d bytes from %v: %+v\n", n, addr, header)
			if err != nil || n == 0 {
				fmt.Println("Error: ", err)
				continue
			}
			// write to TUN interface
			cbnet.CBNet.Write(buf[:n])
		}
	}()

	// Encapsulation
	go func() {
		packet := make([]byte, BUFFERSIZE)
		for {
			plen, err := cbnet.CBNet.Read(packet)
			if err != nil {
				break
			}

			// debug :)
			header, _ := ipv4.ParseHeader(packet[:plen])
			fmt.Printf("Sending to remote: %+v (%+v)\n", header, err)
			// real send

			idx := cbnet.networkingRule.GetIndexOfCBNetIP(header.Dst.String())

			var remoteIP string
			if idx != -1 {
				remoteIP = cbnet.networkingRule.PublicIP[idx]
			}
			// Retrieve from table
			// remoteIP = flag.String("remote", "3.128.34.227", "Remote server (external) IP like 8.8.8.8")

			// Reslove remote addr
			remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", remoteIP, cbnet.port))
			if nil != err {
				log.Fatalln("Unable to resolve remote addr:", err)
			}

			lstnConn.WriteToUDP(packet[:plen], remoteAddr)
		}
	}()
}

// To be deprecated
// Define a function to get network interfaces in a physical or virtual machine
func (cbnet *CBNetwork) UpdateNetworkInterfaceInfo() {

	// Get network interfaces
	ifaces, _ := net.Interfaces()

	// Recursively get network interface information
	for _, iface := range ifaces {
		// Print a network interface name
		//fmt.Println("Interface name:", iface.Name)

		// Declare a NetworkInterface variable
		var networkInterface dataobjects.NetworkInterface

		// Assign Interface Name
		networkInterface.Name = iface.Name

		// Get addresses
		addrs, _ := iface.Addrs()

		// Recursively get IP address
		for _, addr := range addrs {
			addrStr := addr.String()

			// Get IP Address and CIDRBlock ID
			ipAddr, networkID, err := net.ParseCIDR(addrStr)
			if err != nil {
				log.Fatal(err)
			}

			// Get version of IP (e.g., IPv4 or IPv6)
			var version string

			if ipAddr.To4() != nil {
				version = "IPv4"
			} else if ipAddr.To16() != nil {
				version = "IPv6"
			} else {
				version = "Unknown"
				fmt.Printf("Unknown version (IPAddr: %s)\n", ipAddr.String())
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
}

func (cbnet CBNetwork) GetNetworkInterfaces() []dataobjects.NetworkInterface {
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
