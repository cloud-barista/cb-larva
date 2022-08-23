package cbnet

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	ruletype "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/rule-type"
	secutil "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/secret-util"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
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

func init() {
	fmt.Println("\nStart......... init() of cb-network.go")
	// Set cb-log
	env := os.Getenv("CBLOG_ROOT")
	if env != "" {
		// Load cb-log config from the environment variable path (default)
		fmt.Printf("CBLOG_ROOT: %v\n", env)
		CBLogger = cblog.GetLogger("cb-network")
	} else {

		// Load cb-log config from the current directory (usually for the production)
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exePath := filepath.Dir(ex)
		// fmt.Printf("exe path: %v\n", exePath)

		logConfPath := filepath.Join(exePath, "config", "log_conf.yaml")
		if file.Exists(logConfPath) {
			fmt.Printf("path of log_conf.yaml: %v\n", logConfPath)
			CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)

		} else {
			// Load cb-log config from the project directory (usually for development)
			logConfPath = filepath.Join(exePath, "..", "..", "config", "log_conf.yaml")
			if file.Exists(logConfPath) {
				fmt.Printf("path of log_conf.yaml: %v\n", logConfPath)
				CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)
			} else {
				err := errors.New("fail to load log_conf.yaml")
				panic(err)
			}
		}

		CBLogger.Debugf("Load %v", logConfPath)
	}
	fmt.Println("End......... init() of cb-network.go")
	fmt.Println("")
}

type ifReq struct {
	Name  [0x10]byte
	Flags uint16
	pad   [0x28 - 0x10 - 2]byte
}

// CBNetwork represents a network for the multi-cloud
type CBNetwork struct {
	// Variables for the cb-network
	CLADNetID           string               // ID for a cloud adaptive network
	isEncryptionEnabled bool                 // Status if encryption is applied or not.
	NetworkingRule      model.NetworkingRule // Networking rule for a network interface and tunneling
	networkingRuleMutex *sync.Mutex          // mutex for networking-rule

	// Variables for the cb-network controller
	// TBD

	// Variables for the cb-network agents
	HostID                string                    // HostID in a cloud adaptive network
	HostName              string                    // HostName in a cloud adaptive network
	HostPublicIP          string                    // Inquired public IP of VM/Host
	ThisPeer              model.Peer                // Peer object for this host
	OtherPeers            map[string]model.Peer     // Peers map for the other hosts
	Interface             *os.File                  // Assigned cbnet0 IP from the controller
	name                  string                    // Name of a network interface, e.g., cbnet0
	port                  int                       // Port used for tunneling
	isInterfaceConfigured bool                      // Status if a network interface is configured or not
	notificationChannel   chan bool                 // Channel to notify the status of a network interface
	privateKey            *rsa.PrivateKey           // Private key
	keyring               map[string]*rsa.PublicKey // Keyring for secrets
	keyringMutex          *sync.Mutex               // Mutex for keyring
	peersMutex            *sync.Mutex               // Mutex for peers
	listenConnection      *net.UDPConn              // Listen connection for encapsulation and decapsulation

	// Models
	hostNetworkInterfaces []model.NetworkInterface // Inquired network interfaces of VM/Host
}

// New represents a constructor of CBNetwork
func New(name string, port string) *CBNetwork {
	CBLogger.Debug("Start.........")

	// Default
	tunDeviceName := "cbnet0"
	tunnelingPort := 8055

	// Set network interface name
	if name != "" {
		tunDeviceName = name
	}

	// Set tunneling port
	var err error
	if port != "" {
		tunnelingPort, err = strconv.Atoi(port)
		if err != nil {
			CBLogger.Error(err)
		}
	}

	temp := &CBNetwork{
		name:                  tunDeviceName,
		port:                  tunnelingPort,
		isEncryptionEnabled:   false,
		networkingRuleMutex:   new(sync.Mutex),
		OtherPeers:            make(map[string]model.Peer),
		isInterfaceConfigured: false,
		notificationChannel:   make(chan bool),
		keyring:               make(map[string]*rsa.PublicKey),
		keyringMutex:          new(sync.Mutex),
		peersMutex:            new(sync.Mutex),
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
	cbnetwork.getHostNetworkInterfaces()
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
		CBLogger.Debug("Try to inquire public IP address")
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

func (cbnetwork *CBNetwork) getHostNetworkInterfaces() {
	CBLogger.Debug("Start.........")

	var networkInterfaces []model.NetworkInterface

	// Get network interfaces
	ifaces, _ := net.Interfaces()

	// Recursively get network interface information
	for _, iface := range ifaces {
		// Print a network interface name
		CBLogger.Trace("Interface name: ", iface.Name)

		// Declare a NetworkInterface variable
		var networkInterface model.NetworkInterface

		// Assign Interface Interface Name
		networkInterface.Name = iface.Name

		// Get addresses
		addrs, _ := iface.Addrs()

		// Recursively get IP address
		for _, addr := range addrs {
			ipCIDR := addr.String()

			// Get IP Address and IP Network
			ipAddr, _, err := net.ParseCIDR(ipCIDR)
			if err != nil {
				CBLogger.Error(err)
			}

			// To string
			ipAddrStr := ipAddr.String()
			// ipNetworkStr := ipNetwork.String()

			// Filter local IPs to avoid collision between the IPs and the CLADNet
			if ipAddr.IsPrivate() || ipAddr.IsLoopback() || ipAddr.IsLinkLocalUnicast() || ipAddr.IsLinkLocalMulticast() {

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

				// Append the IP network to a list for local IP network
				if version == IPv4 { // Is IPv4 ?
					CBLogger.Tracef("IPv4: %s, IPv4CIDR: %s", ipAddrStr, ipCIDR)
					networkInterface.IPv4 = ipAddrStr
					networkInterface.IPv4CIDR = ipCIDR
				} else if version == IPv6 { // Is IPv6 ?
					CBLogger.Tracef("IPv6: %s, IPv6CIDR: %s", ipAddrStr, ipCIDR)
					networkInterface.IPv6 = ipAddrStr
					networkInterface.IPv6CIDR = ipCIDR
				} else { // Unknown version
					CBLogger.Trace("!!! Unknown version !!!")
				}
			} else {
				CBLogger.Tracef("PublicIPAddress %s, %s", ipAddrStr, ipCIDR)
			}
		}
		networkInterfaces = append(networkInterfaces, networkInterface)
	}
	cbnetwork.hostNetworkInterfaces = networkInterfaces
}

// GetHostNetworkInformation represents a function to get the network information of a VM.
func (cbnetwork CBNetwork) GetHostNetworkInformation() model.HostNetworkInformation {
	CBLogger.Debug("Start.........")

	temp := model.HostNetworkInformation{
		HostName:          cbnetwork.HostName,
		IsEncrypted:       cbnetwork.isEncryptionEnabled,
		PublicIP:          cbnetwork.HostPublicIP,
		NetworkInterfaces: cbnetwork.hostNetworkInterfaces,
	}
	CBLogger.Trace(temp)

	CBLogger.Debug("End.........")
	return temp
}

// UpdateNetworkingRule represents a function to update networking rule.
func (cbnetwork *CBNetwork) UpdateNetworkingRule(networkingRule model.NetworkingRule) {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("Lock to update the networking rule")
	cbnetwork.networkingRuleMutex.Lock()
	cbnetwork.NetworkingRule = networkingRule
	CBLogger.Debug("Unlock to update the networking rule")
	cbnetwork.networkingRuleMutex.Unlock()

	CBLogger.Debug("End.........")
}

// func (cbnetwork *CBNetwork) updateNetworkingRule(peer model.Peer) {
// 	CBLogger.Debug("Start.........")

// 	CBLogger.Debug("Lock to update the networking rule")
// 	mutex.Lock()
// 	cbnetwork.NetworkingRule.CLADNetID = peer.CLADNetID
// 	cbnetwork.NetworkingRule.UpdateRule(peer.HostID, peer.HostName, peer.IP, peer.HostPublicIP, peer.State)
// 	CBLogger.Debug("Unlock to update the networking rule")
// 	mutex.Unlock()

// 	CBLogger.Debug("End.........")
// }

// // UpdateNetworkingRule represents a function to decode binary of networking rule and set it.
// func (cbnetwork *CBNetwork) UpdateNetworkingRule(peer model.Peer) {
// 	CBLogger.Debug("Start.........")

// 	prettyJSON, _ := json.MarshalIndent(peer, "", "\t")
// 	CBLogger.Trace("Pretty JSON")
// 	CBLogger.Trace(string(prettyJSON))

// 	cbnetwork.updateNetworkingRule(peer)

// 	CBLogger.Debug("End.........")
// }

// ThisPeerState represents the state of this host (peer)
func (cbnetwork CBNetwork) ThisPeerState() string {
	CBLogger.Debugf("Current peer state: %s", cbnetwork.ThisPeer.State)
	return cbnetwork.ThisPeer.State
}

// ConfigureCBNetworkInterface represents a function to configure a network interface (default: cbnet0)
// for Cloud Adaptive Network
func (cbnetwork *CBNetwork) ConfigureCBNetworkInterface() error {
	CBLogger.Debug("Start.........")

	// Open TUN device
	fd, err := syscall.Open("/dev/net/tun", os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		log.Fatal(err)
	}
	fdInt := uintptr(fd)

	// Setup a file descriptor
	var flags uint16 = syscall.IFF_NO_PI
	flags |= syscall.IFF_TUN

	// Create an interface
	var req ifReq

	req.Flags = flags
	copy(req.Name[:], cbnetwork.name)

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fdInt, uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		return err
	}

	createdIFName := strings.Trim(string(req.Name[:]), "\x00")
	CBLogger.Tracef("Created interface name: %s\n", createdIFName)
	CBLogger.Info("Interface allocated: ", cbnetwork.name)

	// Open TUN Interface
	tunFd := os.NewFile(fdInt, "tun")
	cbnetwork.Interface = tunFd

	// Get HostIPv4CIDR
	thisPeerIPv4CIDR := cbnetwork.ThisPeer.IPv4CIDR
	CBLogger.Trace("=== cb-network.HostIPv4CIDR: ", thisPeerIPv4CIDR)

	// Set interface parameters
	cbnetwork.runIP("link", "set", "dev", cbnetwork.name, "mtu", MTU)
	cbnetwork.runIP("addr", "add", thisPeerIPv4CIDR, "dev", cbnetwork.name)
	cbnetwork.runIP("link", "set", "dev", cbnetwork.name, "up")

	time.Sleep(1 * time.Second)

	cbnetwork.isInterfaceConfigured = true
	cbnetwork.notificationChannel <- cbnetwork.isInterfaceConfigured

	// Wait until tunneling() is started
	time.Sleep(1 * time.Second)

	CBLogger.Debug("End.........")
	return nil
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

// Run represents a function to start the cloud-barista network.
func (cbnetwork *CBNetwork) Run() {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("Blocked till the networking rule setup")
	cbnetwork.notificationChannel = make(chan bool)
	<-cbnetwork.notificationChannel

	cbnetwork.initializeTunneling()

	CBLogger.Debug("End.........")
}

// initializeTunneling represents a function to be performing tunneling between hosts (e.g., VMs).
func (cbnetwork *CBNetwork) initializeTunneling() {

	CBLogger.Debug("Start.........")

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
	cbnetwork.listenConnection = lstnConn

	// Perform error handling
	defer func() {
		errClose := cbnetwork.listenConnection.Close()
		if errClose != nil {
			CBLogger.Error("can't close the listen connection", errClose)
		}
	}()

	var wg sync.WaitGroup

	// Decapsulation
	wg.Add(1)
	go cbnetwork.decapsulate(&wg)

	// Encapsulation
	wg.Add(1)
	go cbnetwork.encapsulate(&wg)

	wg.Wait()

	CBLogger.Debug("End.........")
}

func (cbnetwork *CBNetwork) encapsulate(wg *sync.WaitGroup) error {
	CBLogger.Debug("Start.........")
	defer wg.Done()

	packet := make([]byte, BUFFERSIZE)
	for {

		// Read packet from the interface "cbnet0"
		plen, err := cbnetwork.Interface.Read(packet[:])
		if err != nil {
			CBLogger.Error("Error Read() in encapsulation: ", err)
			return err
		}

		// Parse header
		header, _ := ipv4.ParseHeader(packet[:plen])
		CBLogger.Tracef("[Encapsulation] Received %d bytes from %v", plen, header.Src.String())
		CBLogger.Tracef("[Encapsulation] Header: %+v", header)

		// Search and change destination (Public IP of target VM)
		idx := cbnetwork.NetworkingRule.GetIndexOfPeerIP(header.Dst.String())

		if idx != -1 {

			// Get the corresponding host's IP address
			remoteIP := cbnetwork.NetworkingRule.SelectedIP[idx]

			// Resolve remote addr
			remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", remoteIP, cbnetwork.port))
			CBLogger.Tracef("Remote Endpoint: %+v", remoteAddr)
			if nil != err {
				CBLogger.Fatal("Unable to resolve remote addr:", err)
			}

			bufToWrite := packet[:plen]

			if cbnetwork.isEncryptionEnabled {

				peerScope := cbnetwork.NetworkingRule.PeerScope[idx]
				if peerScope == "inter" {

					// Get the corresponding host's public key
					HostID := cbnetwork.NetworkingRule.HostID[idx]
					CBLogger.Tracef("HostID: %+v", HostID)
					publicKey := cbnetwork.GetKey(HostID)

					// Encrypt plaintext by corresponidng public key
					ciphertext, err := rsa.EncryptPKCS1v15(
						rand.Reader,
						publicKey,
						[]byte(packet[:plen]),
					)
					CBLogger.Tracef("[Encapsulation] Ciphertext (encrypted) %d bytes", len(ciphertext))

					if err != nil {
						CBLogger.Error("could not encrypt plaintext")
						continue
					}

					bufToWrite = ciphertext
					plen = len(ciphertext)
				}
			}

			// Send packet
			nWriteToUDP, errWriteToUDP := cbnetwork.listenConnection.WriteToUDP(bufToWrite[:plen], remoteAddr)
			if errWriteToUDP != nil || nWriteToUDP == 0 {
				CBLogger.Errorf("Error(%d len): %s", nWriteToUDP, errWriteToUDP)
			}
		}
		// CBLogger.Debug("End.........")
	}
}

func (cbnetwork *CBNetwork) decapsulate(wg *sync.WaitGroup) error {
	CBLogger.Debug("Start.........")
	defer wg.Done()

	// Decapsulation
	buf := make([]byte, BUFFERSIZE)
	for {
		// ReadFromUDP acts like ReadFrom but returns a UDPAddr.
		n, addr, err := cbnetwork.listenConnection.ReadFromUDP(buf)
		if err != nil {
			CBLogger.Error("Error in cbnetwork.listenConnection.ReadFromUDP(buf): ", err)
			return err
		}
		CBLogger.Tracef("[Decapsulation] Received %d bytes from %v", n, addr)

		bufToWrite := buf[:n]
		// if n < BUFFERSIZE-1 {
		// 	buf[n+1] = '\n'
		// }

		if cbnetwork.isEncryptionEnabled {

			// Search and change destination (Public IP of target VM)
			idx := cbnetwork.NetworkingRule.GetIndexOfSelectedIP(addr.IP.String())

			if idx != -1 {
				// Get the corresponding peer's scope
				peerScope := cbnetwork.NetworkingRule.PeerScope[idx]

				if peerScope == "inter" {
					// Decrypt ciphertext by private key
					plaintext, err := rsa.DecryptPKCS1v15(
						rand.Reader,
						cbnetwork.privateKey,
						buf[:n],
					)
					CBLogger.Tracef("[Decapsulation] Plaintext (decrypted) %d bytes", len(plaintext))

					if err != nil {
						CBLogger.Error("could not decrypt ciphertext")
						continue
					}
					bufToWrite = plaintext
					n = len(plaintext)
				}
			}

		}

		// Parse header
		header, _ := ipv4.ParseHeader(bufToWrite)
		CBLogger.Tracef("[Decapsulation] Header: %+v", header)

		// It might be necessary to handle or route packets to the specific destination
		// based on the NetworkingRule table
		// To be determined.

		// Write to TUN interface
		nWrite, errWrite := cbnetwork.Interface.Write(bufToWrite[:n])
		if errWrite != nil || nWrite == 0 {
			CBLogger.Errorf("Error(%d len): %s", nWrite, errWrite)
		}

	}
	// CBLogger.Debug("End.........")
}

// CloseCBNetworkInterface represents a function to stop the cloud-barista network.
func (cbnetwork *CBNetwork) CloseCBNetworkInterface() {
	CBLogger.Debug("Start.........")

	// [To be improved] Stop tunneling routines
	// Currently just return func() when an error occur

	CBLogger.Debug("close the listen connection")
	cbnetwork.listenConnection.Close()

	CBLogger.Debugf("down interface (%s)", cbnetwork.name)
	cbnetwork.runIP("link", "set", "dev", cbnetwork.name, "down")

	CBLogger.Debug("close interface")
	cbnetwork.Interface.Close()

	CBLogger.Debug("set flag (isInterfaceConfigured) false")
	cbnetwork.isInterfaceConfigured = false

	CBLogger.Debug("close channel (notificationChannel)")
	close(cbnetwork.notificationChannel)

	CBLogger.Debug("End.........")
}

// EnableEncryption represents a function to set a status for message encryption.
func (cbnetwork *CBNetwork) EnableEncryption(isTrue bool) {
	if isTrue {
		err := cbnetwork.configureRSAKey()
		if err != nil {
			CBLogger.Error(err)
		}
		cbnetwork.isEncryptionEnabled = true
	}
}

// DisableEncryption represents a function to set a status for message encryption.
func (cbnetwork *CBNetwork) DisableEncryption() {
	cbnetwork.isEncryptionEnabled = false
}

// IsEncryptionEnabled represents a function to check if a message is encrypted or not.
func (cbnetwork CBNetwork) IsEncryptionEnabled() bool {
	return cbnetwork.isEncryptionEnabled
}

// GetPublicKeyBase64 represents a function to get a public key.
func (cbnetwork CBNetwork) GetPublicKeyBase64() (string, error) {
	return secutil.PublicKeyToBase64(&cbnetwork.privateKey.PublicKey)
}

// GenerateRSAKey represents a function to generate RSA key
func (cbnetwork *CBNetwork) configureRSAKey() error {
	CBLogger.Debug("Start.........")

	// Set directory
	ex, err := os.Executable()
	if err != nil {
		CBLogger.Error(err)
	}
	exePath := filepath.Dir(ex)
	CBLogger.Tracef("exePath: %v\n", exePath)

	// Set secret path
	secretPath := filepath.Join(exePath, "secret")

	// Set file and path for private key
	privateKeyFile := cbnetwork.HostID + ".pem"
	privateKeyPath := filepath.Join(secretPath, privateKeyFile)
	CBLogger.Tracef("privateKeyPath: %+v", privateKeyPath)

	// Set file and path for public key
	publicKeyFile := cbnetwork.HostID + ".pub"
	publicKeyPath := filepath.Join(secretPath, publicKeyFile)
	CBLogger.Tracef("publicKeyPath: %+v", publicKeyPath)

	if !file.Exists(privateKeyPath) {
		CBLogger.Debug("Generage and save RSA key to files")
		// Create directory or folder if not exist
		_, err := os.Stat(secretPath)

		if os.IsNotExist(err) {
			errDir := os.MkdirAll(secretPath, 0600)
			if errDir != nil {
				log.Fatal(err)
			}

		}

		// Generate RSA key
		privateKey, publicKey, err := secutil.GenerateRSAKey()
		if err != nil {
			return err
		}

		// Set member data in CBNetwork
		cbnetwork.privateKey = privateKey

		// To bytes
		privateKeyBytes, err := secutil.PrivateKeyToBytes(privateKey)
		if err != nil {
			return err
		}

		// Save private key
		err = secutil.SavePrivateKeyToFile(privateKeyBytes, privateKeyPath)
		if err != nil {
			return err
		}

		// To bytes
		publicKeyBytes, err := secutil.PublicKeyToBytes(publicKey)
		if err != nil {
			return err
		}

		// Save public key
		err = secutil.SavePublicKeyToFile(publicKeyBytes, publicKeyPath)
		if err != nil {
			return err
		}

	} else {
		CBLogger.Debug("Load RSA key from files")
		privateKey, err := secutil.LoadPrivateKeyFromFile(privateKeyPath)
		if err != nil {
			return err
		}

		publicKey, err := secutil.LoadPublicKeyFromFile(publicKeyPath)
		if err != nil {
			return err
		}

		privateKey.PublicKey = *publicKey

		// Set member data in CBNetwork
		cbnetwork.privateKey = privateKey
	}

	CBLogger.Debug("End.........")

	return nil
}

// UpdateKeyring updates a public key with a host ID
func (cbnetwork *CBNetwork) UpdateKeyring(hostID string, base64PublicKey string) error {
	CBLogger.Debug("Start.........")
	publicKey, err := secutil.PublicKeyFromBase64(base64PublicKey)
	if err != nil {
		return err
	}

	cbnetwork.keyringMutex.Lock()
	cbnetwork.keyring[hostID] = publicKey
	cbnetwork.keyringMutex.Unlock()
	CBLogger.Debug("End.........")

	return nil
}

// GetKey returns a public key by a host ID
func (cbnetwork CBNetwork) GetKey(hostID string) *rsa.PublicKey {
	CBLogger.Debug("Start.........")
	cbnetwork.keyringMutex.Lock()
	key := cbnetwork.keyring[hostID]
	cbnetwork.keyringMutex.Unlock()
	CBLogger.Debug("End.........")

	return key
}

// ConfigureHostID represents a function to set a unique host ID
func (cbnetwork *CBNetwork) ConfigureHostID() error {
	CBLogger.Debug("Start.........")

	// Set directory
	ex, err := os.Executable()
	if err != nil {
		CBLogger.Error(err)
	}
	exePath := filepath.Dir(ex)
	CBLogger.Tracef("exePath: %v\n", exePath)

	// Set secret path
	secretPath := filepath.Join(exePath, "secret")

	// Set file and path for the host ID
	hostIDFile := "hostID"
	hostIDPath := filepath.Join(secretPath, hostIDFile)
	CBLogger.Tracef("hostIDPath: %+v", hostIDPath)

	if !file.Exists(hostIDPath) {
		CBLogger.Debug("Generate and save host ID to file")
		// Create directory or folder if not exist
		_, err := os.Stat(secretPath)

		if os.IsNotExist(err) {
			errDir := os.MkdirAll(secretPath, 0600)
			if errDir != nil {
				log.Fatal(err)
			}

		}

		// Generate host ID
		guid := xid.New()
		hostID := guid.String()

		cbnetwork.HostID = hostID

		// Dump host ID to file
		err = ioutil.WriteFile(hostIDPath, []byte(hostID), 0644)
		if err != nil {
			CBLogger.Error(err)
			return err
		}

	} else {
		CBLogger.Debug("Load host ID from file")

		dat, err := ioutil.ReadFile(hostIDPath)
		if err != nil {
			CBLogger.Error(err)
			return err
		}

		cbnetwork.HostID = string(dat)

	}

	CBLogger.Debug("End.........")

	return nil
}

// SelectDestinationByRuleType represents a function to set a unique host ID
func SelectDestinationByRuleType(ruleType string, sourcePeer model.Peer, destinationPeer model.Peer) (string, string, error) {
	CBLogger.Tracef("Start.........")

	var err error

	CBLogger.Tracef("Rule type: %+v", ruleType)
	switch ruleType {
	case ruletype.Basic:
		return destinationPeer.HostPublicIP, "inter", nil

	case ruletype.CostPrioritized:
		selectedDestination, peerScope, err2 := selectByCostPrioritizedRule(sourcePeer, destinationPeer)
		if err2 != nil {
			CBLogger.Debugf("Public IP is selected (due to %v)", err2)
			return destinationPeer.HostPublicIP, "inter", nil
		}
		return selectedDestination, peerScope, nil

	default:
		err = errors.New("unknown rule type")
		CBLogger.Error(err)
	}

	CBLogger.Tracef("End.........")
	return destinationPeer.HostPublicIP, "inter", err
}

func selectByCostPrioritizedRule(sourcePeer model.Peer, destinationPeer model.Peer) (string, string, error) {
	// Check if cloud information is set or not
	if sourcePeer.Details == (model.CloudInformation{}) || destinationPeer.Details == (model.CloudInformation{}) {
		err := fmt.Errorf("no cloud information => src (%v), des (%v)", sourcePeer.Details, destinationPeer.Details)
		return "", "", err
	}

	srcInfo := sourcePeer.Details
	desInfo := destinationPeer.Details

	if srcInfo.VirtualNetworkID == "" || desInfo.VirtualNetworkID == "" {
		err := fmt.Errorf("no vNet/VPC ID => src (%v), des (%v)", srcInfo.VirtualNetworkID, desInfo.VirtualNetworkID)
		return "", "", err
	}

	if srcInfo.VirtualNetworkID != desInfo.VirtualNetworkID {
		return destinationPeer.HostPublicIP, "inter", nil
	}

	switch srcInfo.ProviderName {
	case "aws":
		// If there is no value, return public IP
		if srcInfo.SubnetID == "" || desInfo.SubnetID == "" {
			err := fmt.Errorf("no SubnetID => src (%v), des (%v)", srcInfo.SubnetID, desInfo.SubnetID)
			return "", "", err
		}

		// If both SubnetIDs are the same, return private IP.
		if srcInfo.SubnetID == desInfo.SubnetID {
			return destinationPeer.HostPrivateIP, "intra", nil
		}
		// If not the same, return public IP.
		return destinationPeer.HostPublicIP, "inter", nil

	case "azure", "gcp":
		// If there is no value, return public IP
		if srcInfo.AvailabilityZoneID == "" || desInfo.AvailabilityZoneID == "" {
			err := fmt.Errorf("no AvailabilityZoneID => src (%v), des (%v)", srcInfo.AvailabilityZoneID, desInfo.AvailabilityZoneID)
			return "", "", err
		}
		// If both Availability Zones are the same, return private IP.
		if srcInfo.AvailabilityZoneID == desInfo.AvailabilityZoneID {
			return destinationPeer.HostPrivateIP, "intra", nil
		}
		// If not the same, return public IP.
		return destinationPeer.HostPublicIP, "inter", nil

	case "alibaba": // IBM may be added here.
		// If both vNets/VPCs are the same, return private IP.
		return destinationPeer.HostPrivateIP, "intra", nil

	default:
		err := fmt.Errorf("unknown name of cloud service provider (ProviderName: %v)", srcInfo.ProviderName)
		return "", "", err
	}
}

// StorePeer represents a function to add, synchronize, manage peers in local (memory)
func (cbnetwork *CBNetwork) StorePeer(peer model.Peer) {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("Lock to update peers")
	cbnetwork.peersMutex.Lock()

	// Store and synchronize peers to manage them in local
	if peer.HostID == cbnetwork.HostID {
		// Assign peer
		cbnetwork.ThisPeer = peer

	} else {
		// Assign peer
		cbnetwork.OtherPeers[peer.HostID] = peer
	}

	CBLogger.Debug("Unlock to update peers")
	cbnetwork.peersMutex.Unlock()

	CBLogger.Debug("End.........")
}

// GetPeer represents a function to find and return a peer in the local map (data structure)
func (cbnetwork *CBNetwork) GetPeer(hostID string) (model.Peer, error) {
	CBLogger.Debug("Start.........")

	// CBLogger.Debug("Lock to update peers")
	// cbnetwork.peersMutex.Lock()

	if hostID == cbnetwork.HostID {
		CBLogger.Debug("End.........")
		return cbnetwork.ThisPeer, nil
	}
	tempPeer, exist := cbnetwork.OtherPeers[hostID]
	if !exist {
		CBLogger.Debug("End.........")
		return model.Peer{}, errors.New("could not find the peer")
	}
	CBLogger.Debug("End.........")
	return tempPeer, nil
}
