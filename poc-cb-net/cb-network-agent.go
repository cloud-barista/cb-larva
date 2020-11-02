package poc_cb_net

import (
	"fmt"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/data-objects"
	"log"
	"net"
)

type CBNetworkAgent struct {
	networkInterfaces []dataobjects.NetworkInterface	// A slice to put network interfaces
}

// Constructor
func NewCBNetworkAgent() *CBNetworkAgent {
	temp := &CBNetworkAgent{}
	temp.UpdateNetworkInterface()
	return temp
}

// Define a function to get network interfaces in a physical or virtual machine
func (agent *CBNetworkAgent) UpdateNetworkInterface() {

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

			// Get IP Address and Network ID
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

			// Print version, IPAddress, Network ID
			//fmt.Println("	 Version: ", version)
			//fmt.Println("	 IPAddr: ", ipAddr)
			//fmt.Println("	 NetworkID: ", networkID)

			// Create IP data object
			ip := dataobjects.IP{
				Version:   version,
				IPAddress: ipAddr.String(),
				NetworkID: networkID.String(),
			}

			// Append the IP data object to slice
			networkInterface.IPs = append(networkInterface.IPs, ip)
		}
		agent.networkInterfaces = append(agent.networkInterfaces, networkInterface)
	}
}

func (agent CBNetworkAgent) GetNetworkInterface() []dataobjects.NetworkInterface{
	return agent.networkInterfaces
}

//func main() {
//
//	temp := CBNetworkAgent{}.GetNetworkInterface()
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
//	//		fmt.Println("    net.Addr: ", addr.Network(), addrStr)
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
