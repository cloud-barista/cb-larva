package cbnet

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	nethelper "github.com/cloud-barista/cb-larva/poc-cb-net/internal/network-helper"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	"github.com/songgao/water"
	"golang.org/x/net/ipv4"
)

// I use TUN interface, so only plain IP packet, no ethernet header + mtu is set to 1300
const (
	// BUFFERSIZE represents a size of read buffer.
	BUFFERSIZE = 1500
	// MTU represents a maximum transmission unit.
	MTU = "1300"
	// IPv4 represents a version of IP address
	IPv4 = "IPv4"
	// IPv6 represents a version of IP address
	IPv6 = "IPv6"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var mutex = new(sync.Mutex)

func init() {
	fmt.Println("Start......... init() of cb-network.go")
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	fmt.Printf("exePath: %v\n", exePath)

	// Load cb-log config from the current directory (usually for the production)
	logConfPath := filepath.Join(exePath, "config", "log_conf.yaml")
	fmt.Printf("logConfPath: %v\n", logConfPath)
	if !file.Exists(logConfPath) {
		// Load cb-log config from the project directory (usually for development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		logConfPath = filepath.Join(projectPath, "poc-cb-net", "config", "log_conf.yaml")
	}
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)
	CBLogger.Debugf("Load %v", logConfPath)
	fmt.Println("End......... init() of cb-network.go")
}

// CBNetwork represents a network for the multi-cloud
type CBNetwork struct {
	// Variables for the cb-network
	NetworkingRules model.NetworkingRule // Networking rule for Interface and tunneling
	ID              string               // ID for a cloud adaptive network

	// Variables for the cb-network controller

	// Variables for the cb-network agents
	HostID                  string           // HostID in a cloud adaptive network
	HostPublicIP            string           // Inquired public IP of VM/Host
	HostPrivateIPv4Networks []string         // Inquired private IPv4 networks of VM/Host (e.g. ["192.168.10.4/24", ...])
	Interface               *water.Interface // Assigned cbnet0 IP from the controller
	name                    string           // InterfaceName of Interface, e.g., cbnet0
	port                    int              // Port used for tunneling
	isRunning               bool             // Status if a cloud adaptive network is running
	notificationChannel     chan bool        // Notification channel to start tunneling

	//listenConnection  *net.UDPConn                // Connection for encapsulation and decapsulation
	//NetworkInterfaces []model.NetworkInterface // Deprecated
}

// New represents a constructor of CBNetwork
func New(name string, port int) *CBNetwork {
	CBLogger.Debug("Start.........")

	temp := &CBNetwork{
		name:                name,
		port:                port,
		isRunning:           false,
		notificationChannel: make(chan bool),
	}
	temp.UpdateHostNetworkInformation()

	CBLogger.Debug("End.........")
	return temp
}

// UpdateHostNetworkInformation represents a function to update the host network information, such as
// public IP address of VM and private IPv4 networks.
func (cbnetwork *CBNetwork) UpdateHostNetworkInformation() {
	CBLogger.Debug("Start.........")
	cbnetwork.inquireVMPublicIP()
	cbnetwork.getPrivateIPv4Networks()
	CBLogger.Debug("End.........")
}

func (cbnetwork *CBNetwork) inquireVMPublicIP() {
	CBLogger.Debug("Start.........")

	urls := []string{"https://ifconfig.co/",
		"https://api.ipify.org?format=text",
		"https://www.ipify.org",
		"http://myexternalip.com",
		"http://api.ident.me",
		"http://whatismyipaddress.com/api",
	}

	for _, url := range urls {

		// Try to inquire public IP address
		CBLogger.Info("Try to inuire public IP address")
		CBLogger.Tracef("by %s", url)

		resp, err := http.Get(url)
		if err != nil {
			CBLogger.Error(err)
		}

		// Perform error handling
		defer func() {
			errClose := resp.Body.Close()
			if errClose != nil {
				CBLogger.Fatal("can't close the response", errClose)
			}
		}()

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			CBLogger.Error(err)
		}

		trimmed := strings.TrimSuffix(string(data), "\n") // Remove '\n' if exist
		CBLogger.Tracef("Returned: %s", trimmed)

		// Check if it's IP address or not
		if net.ParseIP(trimmed) != nil {
			CBLogger.Info("Public IP address is acquired.")
			CBLogger.Tracef("Public IP address: %s", string(trimmed))
			cbnetwork.HostPublicIP = trimmed
			break
		}
	}

	// If "", fail to acquire public IP address
	if cbnetwork.HostPublicIP == "" {
		CBLogger.Fatal("Fail to acquire public IP address")
	}

	CBLogger.Debug("End.........")
}

func (cbnetwork *CBNetwork) getPrivateIPv4Networks() {
	CBLogger.Debug("Start.........")

	var tempIPNetworks []string

	// Get network interfaces
	ifaces, _ := net.Interfaces()

	// Recursively get network interface information
	for _, iface := range ifaces {
		// Print a network interface name
		CBLogger.Trace("Interface name:", iface.Name)

		// Declare a NetworkInterface variable
		var networkInterface model.NetworkInterface

		// Assign Interface Interface Name
		networkInterface.Name = iface.Name

		// Get addresses
		addrs, _ := iface.Addrs()

		// Recursively get IP address
		for _, addr := range addrs {
			addrStr := addr.String()

			// Get IP Address and IP Network
			ipAddr, networkID, err := net.ParseCIDR(addrStr)
			if err != nil {
				CBLogger.Error(err)
			}

			// Get version of IP (e.g., IPv4 or IPv6)
			var version string

			if ipAddr.To4() != nil {
				version = IPv4
			} else if ipAddr.To16() != nil {
				version = IPv6
			} else {
				version = "Unknown"
				CBLogger.Tracef("Unknown version (IPAddr: %s)", ipAddr.String())
			}

			// To string
			ipAddrStr := ipAddr.String()
			networkIDStr := networkID.String()

			isPrivateIP := nethelper.IsPrivateIP(ipAddr)
			// Filter privateIPv4 to avoid collision between those IPs and the CLADNet
			if isPrivateIP {
				if version == IPv4 { // Is IPv4 ?
					tempIPNetworks = append(tempIPNetworks, networkIDStr)
					CBLogger.Tracef("True v4 %s, %s", ipAddrStr, networkIDStr)
				} else if version == IPv6 { // Is IPv6 ?
					CBLogger.Tracef("True v6 %s, %s", ipAddrStr, networkIDStr)
				} else { // Unknown version
					CBLogger.Trace("!!! Unknown version !!!")
				}
			} else {
				CBLogger.Tracef("PublicIPAddress %s, %s", ipAddrStr, networkIDStr)
			}
		}
	}
	cbnetwork.HostPrivateIPv4Networks = tempIPNetworks
}

// GetHostNetworkInformation represents a function to get the network information of a VM.
func (cbnetwork CBNetwork) GetHostNetworkInformation() model.HostNetworkInformation {
	CBLogger.Debug("Start.........")

	temp := model.HostNetworkInformation{
		PublicIP:            cbnetwork.HostPublicIP,
		PrivateIPv4Networks: cbnetwork.HostPrivateIPv4Networks,
	}
	CBLogger.Trace(temp)

	CBLogger.Debug("End.........")
	return temp
}

//func (cbnetwork CBNetwork) IsSameNetworkInformation(net1 model.HostNetworkInformation, net2 model.HostNetworkInformation) bool {
//
//	isSame := false
//	if net1.PublicIP == net2.PublicIP {
//		isSame = true
//		for i, privateNetwork := range net1.PrivateIPv4Networks {
//			if privateNetwork != net2.PrivateIPv4Networks[i] {
//				isSame = false
//				break
//			}
//		}
//	}
//	return isSame
//}

// SetNetworkingRules represents a function to set a networking rule
func (cbnetwork *CBNetwork) SetNetworkingRules(rules model.NetworkingRule) {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("Lock to update the networking rule")
	mutex.Lock()
	cbnetwork.NetworkingRules = rules
	CBLogger.Debug("Unlock to update the networking rule")
	mutex.Unlock()

	CBLogger.Debug("End.........")
}

func (cbnetwork *CBNetwork) initCBNet() (int, error) {
	CBLogger.Debug("Start.........")

	idx := cbnetwork.NetworkingRules.GetIndexOfPublicIP(cbnetwork.HostPublicIP)
	if idx < 0 || idx >= len(cbnetwork.NetworkingRules.HostID) {
		return -1, errors.New("index out of range")
	}
	localNetwork := cbnetwork.NetworkingRules.HostIPv4Network[idx]

	localIP := flag.String("local", localNetwork, "Local tun interface IP/MASK like 192.168.3.3⁄24")
	if *localIP == "" {
		flag.Usage()
		CBLogger.Fatal("local ip is not specified")
	}

	iface, err := water.New(water.Config{
		DeviceType:             water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{Name: cbnetwork.name},
	})
	if nil != err {
		CBLogger.Fatal("Unable to allocate TUN interface:", err)
	}
	CBLogger.Info("Interface allocated:", iface.Name())

	cbnetwork.Interface = iface
	CBLogger.Trace("=== cb-network.HostIPv4Network: ", cbnetwork.Interface)

	// Set interface parameters
	cbnetwork.runIP("link", "set", "dev", cbnetwork.Interface.Name(), "mtu", MTU)
	cbnetwork.runIP("addr", "add", *localIP, "dev", cbnetwork.Interface.Name())
	cbnetwork.runIP("link", "set", "dev", cbnetwork.Interface.Name(), "up")

	CBLogger.Debug("End.........")
	return 0, nil
}

func (cbnetwork *CBNetwork) runIP(args ...string) {
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

// IsRunning represents a status of CBNetwork
func (cbnetwork CBNetwork) IsRunning() bool {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("IsRunning? ", cbnetwork.isRunning)

	CBLogger.Debug("End.........")
	return cbnetwork.isRunning
}

// StartCBNetworking represents a function to start networking by networking rules
func (cbnetwork *CBNetwork) StartCBNetworking() (int, error) {
	CBLogger.Debug("Start.........")

	CBLogger.Info("Run CBNetworking between VMs")
	ret, err := cbnetwork.initCBNet()
	if err != nil {
		return ret, err
	}
	cbnetwork.isRunning = true
	cbnetwork.notificationChannel <- cbnetwork.isRunning

	CBLogger.Debug("End.........")
	return 0, nil
}

// RunTunneling represents a function to be performing tunneling between hosts (e.g., VMs).
func (cbnetwork *CBNetwork) RunTunneling() {
	// defer wg.Done()
	CBLogger.Debug("Start.........")

	CBLogger.Debug("Blocked till Networking Rule setup")
	<-cbnetwork.notificationChannel

	CBLogger.Debug("Start decapsulation")
	// Decapsulation

	// Listen to local socket
	// Create network address to listen
	lstnAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%v", cbnetwork.port))
	if err != nil {
		CBLogger.Fatal("Unable to get UDP socket:", err)
	}

	// Create connection to network address
	lstnConn, err := net.ListenUDP("udp", lstnAddr)
	if err != nil {
		CBLogger.Fatal("Unable to listen on UDP socket:", err)
	}

	// Perform error handling
	defer func() {
		errClose := lstnConn.Close()
		if errClose != nil {
			CBLogger.Fatal("can't close the listen connection", errClose)
		}
	}()

	go func() {
		buf := make([]byte, BUFFERSIZE)
		for {
			// ReadFromUDP acts like ReadFrom but returns a UDPAddr.
			n, _, err := lstnConn.ReadFromUDP(buf)
			if err != nil {
				CBLogger.Error("Error in cbnetwork.listenConnection.ReadFromUDP(buf): ", err)
			}

			// Parse header
			header, _ := ipv4.ParseHeader(buf[:n])
			CBLogger.Tracef("Header received: %+v", header)

			//fmt.Printf("Received %d bytes from %v: %+v", n, addr, header)

			// It might be necessary to handle or route packets to the specific destination
			// based on the NetworkingRule table
			// To be determined.

			// Write to TUN interface
			nWrite, errWrite := cbnetwork.Interface.Write(buf[:n])
			if errWrite != nil || nWrite == 0 {
				CBLogger.Errorf("Error(%d len): %s", nWrite, errWrite)
			}
		}
	}()

	CBLogger.Debug("Start encapsulation")
	// Encapsulation
	packet := make([]byte, BUFFERSIZE)
	for {
		// Read packet from HostIPv4Network interface "cbnet0"
		//fmt.Println("=== *cbnetwork.HostIPv4Network: ", *cbnetwork.HostIPv4Network)
		//fmt.Println("=== cbnetwork.HostIPv4Network: ",cbnetwork.HostIPv4Network)
		plen, err := cbnetwork.Interface.Read(packet)
		if err != nil {
			CBLogger.Error("Error Read() in encapsulation:", err)
		}

		// Parse header
		header, _ := ipv4.ParseHeader(packet[:plen])
		CBLogger.Tracef("Sending to remote: %+v (%+v)", header, err)

		// Search and change destination (Public IP of target VM)
		idx := cbnetwork.NetworkingRules.GetIndexOfCBNetIP(header.Dst.String())

		var remoteIP string
		if idx != -1 {
			remoteIP = cbnetwork.NetworkingRules.PublicIPAddress[idx]
		}

		// Resolve remote addr
		remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", remoteIP, cbnetwork.port))
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

// DecodeAndSetNetworkingRule represents a function to decode binary of networking rule and set it.
func (cbnetwork *CBNetwork) DecodeAndSetNetworkingRule(value []byte) {
	CBLogger.Debug("Start.........")

	var networkingRule model.NetworkingRule

	err := json.Unmarshal(value, &networkingRule)
	if err != nil {
		CBLogger.Error(err)
	}

	prettyJSON, _ := json.MarshalIndent(networkingRule, "", "\t")
	CBLogger.Trace("Pretty JSON")
	CBLogger.Trace(string(prettyJSON))

	if networkingRule.Contain(cbnetwork.HostID) {
		cbnetwork.SetNetworkingRules(networkingRule)
		if !cbnetwork.IsRunning() {
			_, err := cbnetwork.StartCBNetworking()
			if err != nil {
				CBLogger.Error(err)
			}
		}
	}
	CBLogger.Debug("End.........")
}
