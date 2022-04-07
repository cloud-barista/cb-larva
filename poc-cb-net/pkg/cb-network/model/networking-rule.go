package cbnet

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger

func init() {
	fmt.Println("Start......... init() of networking-rule.go")
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
		fmt.Printf("not exist - %v\n", logConfPath)
		// Load cb-log config from the project directory (usually for development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		fmt.Printf("projectRoot: %v\n", string(path))
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		logConfPath = filepath.Join(projectPath, "poc-cb-net", "config", "log_conf.yaml")
	}
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)
	CBLogger.Debugf("Load %v", logConfPath)
	fmt.Println("End......... init() of networking-rule.go")
}

// NetworkingRule represents a networking rule of the cloud adaptive network.
// It is used for tunneling between hosts(e.g., VMs).
type NetworkingRule struct {
	CLADNetID       string   `json:"CLADNetID"`
	HostID          []string `json:"hostID"`
	HostName        []string `json:"hostName"`
	HostIPv4Network []string `json:"hostIPv4Network"`
	HostIPAddress   []string `json:"hostIPAddress"`
	PublicIPAddress []string `json:"publicIPAddress"`
	State           []string `json:"state"`
}

// AppendRule represents a function to append a rule to the NetworkingRule
func (netrule *NetworkingRule) AppendRule(id, name, privateIPv4Network, privateIPv4Address, publicIPv4Addres, state string) {
	CBLogger.Infof("A rule: {%s, %s, %s, %s, %s, %s}", id, name, privateIPv4Network, privateIPv4Address, publicIPv4Addres, state)
	if !netrule.Contain(id) { // If HostID doesn't exists, append rule
		netrule.HostID = append(netrule.HostID, id)
		netrule.HostName = append(netrule.HostName, name)
		netrule.HostIPv4Network = append(netrule.HostIPv4Network, privateIPv4Network)
		netrule.HostIPAddress = append(netrule.HostIPAddress, privateIPv4Address)
		netrule.PublicIPAddress = append(netrule.PublicIPAddress, publicIPv4Addres)
		netrule.State = append(netrule.State, state)
	}
}

// UpdateRule represents a function to update a rule to the NetworkingRule
func (netrule *NetworkingRule) UpdateRule(id, name, privateIPv4Network, privateIPv4Address, publicIPv4Address, state string) {
	CBLogger.Infof("A rule: {%s, %s, %s, %s, %s, %s}", id, name, privateIPv4Network, privateIPv4Address, publicIPv4Address, state)
	if netrule.Contain(id) { // If HostID exists, update rule
		index := netrule.GetIndexOfHostID(id)
		if name != "" {
			netrule.HostName[index] = name
		}
		if privateIPv4Network != "" {
			netrule.HostIPv4Network[index] = privateIPv4Network
		}
		if privateIPv4Address != "" {
			netrule.HostIPAddress[index] = privateIPv4Address
		}
		if state != "" {
			netrule.State[index] = state
		}
		netrule.PublicIPAddress[index] = publicIPv4Address
	} else {
		netrule.AppendRule(id, name, privateIPv4Network, privateIPv4Address, publicIPv4Address, state)
	}
}

// GetIndexOfHostID represents a function to find and return an index of HostID from NetworkingRule
func (netrule NetworkingRule) GetIndexOfHostID(id string) int {
	return netrule.find(netrule.HostID, id)
}

// GetIndexOfHostName represents a function to find and return an index of HostID from NetworkingRule
func (netrule NetworkingRule) GetIndexOfHostName(name string) int {
	return netrule.find(netrule.HostName, name)
}

// GetIndexOfCBNet represents a function to find and return an index of HostIPv4Network from NetworkingRule
func (netrule NetworkingRule) GetIndexOfCBNet(hostIPv4Network string) int {
	return netrule.find(netrule.HostIPv4Network, hostIPv4Network)
}

// GetIndexOfCBNetIP represents a function to find and return an index of HostIPAddress from NetworkingRule
func (netrule NetworkingRule) GetIndexOfCBNetIP(hostIPAddress string) int {
	return netrule.find(netrule.HostIPAddress, hostIPAddress)
}

// GetIndexOfPublicIP represents a function to find and return an index of PublicIPAddress from NetworkingRule
func (netrule NetworkingRule) GetIndexOfPublicIP(publicIP string) int {
	return netrule.find(netrule.PublicIPAddress, publicIP)
}

func (netrule NetworkingRule) find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

// Contain represents a function to check if the host exists or not
func (netrule NetworkingRule) Contain(x string) bool {
	for _, n := range netrule.HostID {
		if x == n {
			return true
		}
	}
	return false
}
