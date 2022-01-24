package cbnet

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	secutil "github.com/cloud-barista/cb-larva/poc-cb-net/internal/secret-util"
	cblog "github.com/cloud-barista/cb-log"
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

type ifReq struct {
	Name  [0x10]byte
	Flags uint16
	pad   [0x28 - 0x10 - 2]byte
}

// CBNetwork represents a network for the multi-cloud
type CBNetwork struct {
	// Variables for the cb-network
	NetworkingRules     model.NetworkingRule // Networking rule for a network interface and tunneling
	ID                  string               // ID for a cloud adaptive network
	isEncryptionEnabled bool                 // Status if encryption is applied or not.

	// Variables for the cb-network controller
	// TBD

	// Variables for the cb-network agents
	HostID                  string                    // HostID in a cloud adaptive network
	HostPublicIP            string                    // Inquired public IP of VM/Host
	HostPrivateIPv4Networks []string                  // Inquired private IPv4 networks of VM/Host (e.g. ["192.168.10.4/24", ...])
	Interface               *os.File                  // Assigned cbnet0 IP from the controller
	name                    string                    // Name of a network interface, e.g., cbnet0
	port                    int                       // Port used for tunneling
	isInterfaceConfigured   bool                      // Status if a network interface is configured or not
	notificationChannel     chan bool                 // Channel to notify the status of a network interface
	privateKey              *rsa.PrivateKey           // Private key
	keyring                 map[string]*rsa.PublicKey // Keyring for secrets
	keyringMutex            *sync.Mutex               // Mutext for keyring

	//listenConnection  *net.UDPConn                // Connection for encapsulation and decapsulation
	//NetworkInterfaces []model.NetworkInterface // Deprecated
}

// New represents a constructor of CBNetwork
func New(name string, port int) *CBNetwork {
	CBLogger.Debug("Start.........")

	temp := &CBNetwork{
		name:                  name,
		port:                  port,
		isEncryptionEnabled:   false,
		isInterfaceConfigured: false,
		notificationChannel:   make(chan bool),
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
		CBLogger.Debug("Try to inuire public IP address")
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
		CBLogger.Trace("Interface name: ", iface.Name)

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
			ipAddr, ipNework, err := net.ParseCIDR(addrStr)
			if err != nil {
				CBLogger.Error(err)
			}

			// To string
			ipAddrStr := ipAddr.String()
			networkIDStr := ipNework.String()

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
					tempIPNetworks = append(tempIPNetworks, networkIDStr)
					CBLogger.Tracef("IPv4: %s, IPv4Network: %s", ipAddrStr, networkIDStr)
				} else if version == IPv6 { // Is IPv6 ?
					CBLogger.Tracef("IPv6: %s, IPv6Network: %s", ipAddrStr, networkIDStr)
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
		IsEncrypted:         cbnetwork.isEncryptionEnabled,
		PublicIP:            cbnetwork.HostPublicIP,
		PrivateIPv4Networks: cbnetwork.HostPrivateIPv4Networks,
	}
	CBLogger.Trace(temp)

	CBLogger.Debug("End.........")
	return temp
}

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
		if !cbnetwork.isInterfaceConfigured {
			err := cbnetwork.configureCBNetworkInterface()
			if err != nil {
				CBLogger.Error(err)
				return
			}
			cbnetwork.isInterfaceConfigured = true
			cbnetwork.notificationChannel <- cbnetwork.isInterfaceConfigured
		}
	}
	CBLogger.Debug("End.........")
}

func (cbnetwork *CBNetwork) configureCBNetworkInterface() error {
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
	CBLogger.Tracef("createdInterfaceName: %s\n", createdIFName)
	CBLogger.Info("Interface allocated:", cbnetwork.name)

	// Open TUN Interface
	tunFd := os.NewFile(fdInt, "tun")
	cbnetwork.Interface = tunFd

	// Get HostIPv4Network
	idx := cbnetwork.NetworkingRules.GetIndexOfPublicIP(cbnetwork.HostPublicIP)
	if idx < 0 || idx >= len(cbnetwork.NetworkingRules.HostID) {
		return errors.New("index out of range")
	}
	localNetwork := cbnetwork.NetworkingRules.HostIPv4Network[idx]

	CBLogger.Trace("=== cb-network.HostIPv4Network: ", localNetwork)

	// Set interface parameters
	cbnetwork.runIP("link", "set", "dev", cbnetwork.name, "mtu", MTU)
	cbnetwork.runIP("addr", "add", localNetwork, "dev", cbnetwork.name)
	cbnetwork.runIP("link", "set", "dev", cbnetwork.name, "up")

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

// Startup represents a function to start the cloud-barista network.
func (cbnetwork *CBNetwork) Startup() {
	CBLogger.Debug("Start.........")

	CBLogger.Debug("Blocked till the networking rule setup")
	<-cbnetwork.notificationChannel

	cbnetwork.runTunneling()

	CBLogger.Debug("End.........")
}

// runTunneling represents a function to be performing tunneling between hosts (e.g., VMs).
func (cbnetwork *CBNetwork) runTunneling() {

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

	// Perform error handling
	defer func() {
		errClose := lstnConn.Close()
		if errClose != nil {
			CBLogger.Fatal("can't close the listen connection", errClose)
		}
	}()

	// Decapsulation
	go func() {
		CBLogger.Debug("Start decapsulation")
		buf := make([]byte, BUFFERSIZE)
		for {
			// ReadFromUDP acts like ReadFrom but returns a UDPAddr.
			n, _, err := lstnConn.ReadFromUDP(buf)
			if err != nil {
				CBLogger.Error("Error in cbnetwork.listenConnection.ReadFromUDP(buf): ", err)
				return
			}

			// Decrypt ciphertext by private key
			CBLogger.Tracef("Ciphertext (To be decrypted): %+v", buf[:n])
			plaintext, err := rsa.DecryptPKCS1v15(
				rand.Reader,
				cbnetwork.privateKey,
				buf[:n],
			)
			CBLogger.Tracef("Plaintext (decrypted): %+v", plaintext)
			if err != nil {
				continue
			}

			// Parse header
			header, _ := ipv4.ParseHeader(plaintext)
			CBLogger.Tracef("[Decapsulation] Header: %+v", header)

			//fmt.Printf("Received %d bytes from %v: %+v", n, addr, header)

			// It might be necessary to handle or route packets to the specific destination
			// based on the NetworkingRule table
			// To be determined.

			// Write to TUN interface
			nWrite, errWrite := cbnetwork.Interface.Write(plaintext)
			if errWrite != nil || nWrite == 0 {
				CBLogger.Errorf("Error(%d len): %s", nWrite, errWrite)
			}
		}
	}()

	// Encapsulation
	CBLogger.Debug("Start encapsulation")
	packet := make([]byte, BUFFERSIZE)
	for {
		// Read packet from the interface "cbnet0"
		plen, err := cbnetwork.Interface.Read(packet[:])
		if err != nil {
			CBLogger.Error("Error Read() in encapsulation:", err)
			return
		}

		// Parse header
		header, _ := ipv4.ParseHeader(packet[:plen])
		CBLogger.Tracef("[Encapsulation] Header: %+v", header)

		// Search and change destination (Public IP of target VM)
		idx := cbnetwork.NetworkingRules.GetIndexOfCBNetIP(header.Dst.String())

		if idx != -1 {

			// Get the corresponding host's IP address
			remoteIP := cbnetwork.NetworkingRules.PublicIPAddress[idx]

			// Resolve remote addr
			remoteAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%v", remoteIP, cbnetwork.port))
			CBLogger.Tracef("Remote Endpoint: %+v", remoteAddr)
			if nil != err {
				CBLogger.Fatal("Unable to resolve remote addr:", err)
			}

			// Get the corresponding host's public key
			publicKey := cbnetwork.GetKey(cbnetwork.NetworkingRules.HostID[idx])

			// Encrypt plaintext by corresponidng public key
			CBLogger.Tracef("Plaintext (To be encrypted): %+v", packet[:plen])
			ciphertext, err := rsa.EncryptPKCS1v15(
				rand.Reader,
				publicKey,
				[]byte(packet[:plen]),
			)
			CBLogger.Tracef("Ciphertext (Encrypted): %+v", ciphertext)

			// Send packet
			nWriteToUDP, errWriteToUDP := lstnConn.WriteToUDP(ciphertext, remoteAddr)
			if errWriteToUDP != nil || nWriteToUDP == 0 {
				CBLogger.Errorf("Error(%d len): %s", nWriteToUDP, errWriteToUDP)
			}
		}
	}

	// Unreachable
	// CBLogger.Debug("End.........")
}

// Shutdown represents a function to stop the cloud-barista network.
func (cbnetwork *CBNetwork) Shutdown() {
	CBLogger.Debug("Start.........")

	// [To be improved] Stop tunneling routines
	// Currently just return func() when an error occur

	cbnetwork.Interface.Close()
	cbnetwork.isInterfaceConfigured = false

	CBLogger.Debug("End.........")
}

// EnableEncryption represents a function to set a status for message encryption.
func (cbnetwork *CBNetwork) EnableEncryption(b bool) {
	cbnetwork.configureRSAKey()
	cbnetwork.keyring = make(map[string]*rsa.PublicKey)
	cbnetwork.keyringMutex = new(sync.Mutex)
	cbnetwork.isEncryptionEnabled = b
}

// IsEncrypionEnabled represents a function to check if a message is encrypted or not.
func (cbnetwork CBNetwork) IsEncrypionEnabled() bool {
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

	// Set file and path for public key
	publicKeyFile := cbnetwork.HostID + ".pub"
	publicKeyPath := filepath.Join(secretPath, publicKeyFile)

	if !file.Exists(secretPath + cbnetwork.HostID + ".pem") {

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
		privateKey, err := secutil.LoadPrivateKeyFromFile(privateKeyPath)
		if err != nil {
			return err
		}

		publicKey, err := secutil.LoadPublicKeyFromFile(privateKeyPath)
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

func (cbnetwork CBNetwork) GetKey(hostID string) *rsa.PublicKey {
	CBLogger.Debug("Start.........")
	cbnetwork.keyringMutex.Lock()
	key := cbnetwork.keyring[hostID]
	cbnetwork.keyringMutex.Unlock()
	CBLogger.Debug("End.........")

	return key
}
