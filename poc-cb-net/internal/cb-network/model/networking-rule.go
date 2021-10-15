package cbnet

import (
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	fmt.Println("End......... init() of networking-rule.go")
}

// NetworkingRule represents a networking rules for tunneling between hosts(e.g., VMs).
type NetworkingRule struct {
	CLADNetID       string   `json:"CLADNetID"`
	HostID          []string `json:"hostID"`
	HostIPCIDRBlock []string `json:"hostIPCIDRBlock"`
	HostIPAddress   []string `json:"hostIPAddress"`
	PublicIPAddress []string `json:"publicIPAddress"`
}

// AppendRule represents a function to append a rule to the NetworkingRule
func (netrule *NetworkingRule) AppendRule(ID string, CBNet string, CBNetIP string, PublicIP string) {
	CBLogger.Infof("A rule: {%s, %s, %s, %s}", ID, CBNet, CBNetIP, PublicIP)
	if !netrule.Contain(ID) { // If HostID doesn't exists, append rule
		netrule.HostID = append(netrule.HostID, ID)
		netrule.HostIPCIDRBlock = append(netrule.HostIPCIDRBlock, CBNet)
		netrule.HostIPAddress = append(netrule.HostIPAddress, CBNetIP)
		netrule.PublicIPAddress = append(netrule.PublicIPAddress, PublicIP)
	}
}

// UpdateRule represents a function to update a rule to the NetworkingRule
func (netrule *NetworkingRule) UpdateRule(id string, hostIPCIDRBlock string, hostIPAddress string, publicIP string) {
	CBLogger.Infof("A rule: {%s, %s, %s, %s}", id, hostIPCIDRBlock, hostIPAddress, publicIP)
	if netrule.Contain(id) { // If HostID exists, update rule
		index := netrule.GetIndexOfID(id)
		if hostIPCIDRBlock != "" {
			netrule.HostIPCIDRBlock[index] = hostIPCIDRBlock
		}
		if hostIPAddress != "" {
			netrule.HostIPAddress[index] = hostIPAddress
		}
		netrule.PublicIPAddress[index] = publicIP
	}
}

// GetIndexOfID represents a function to find and return an index of HostID from NetworkingRule
func (netrule NetworkingRule) GetIndexOfID(id string) int {
	return netrule.find(netrule.HostID, id)
}

// GetIndexOfCBNet represents a function to find and return an index of HostIPCIDRBlock from NetworkingRule
func (netrule NetworkingRule) GetIndexOfCBNet(hostIPCIDRBlock string) int {
	return netrule.find(netrule.HostIPCIDRBlock, hostIPCIDRBlock)
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
