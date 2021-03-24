package cbnet

import (
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	"path/filepath"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger

func init() {
	// cblog is a global variable.
	configPath := filepath.Join("..", "..", "configs", "log_conf.yaml")
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", configPath)
}

// NetworkingRule represents a networking rules for tunneling between hosts(e.g., VMs).
type NetworkingRule struct {
	HostID          []string `json:"hostID"`
	HostIPCIDRBlock []string `json:"HostIPCIDRBlock"`
	HostIPAddress   []string `json:"HostIPAddress"`
	PublicIPAddress []string `json:"publicIPAddress"`
}

// UpdateRule represents a function to append a rule to the NetworkingRule
func (netrule *NetworkingRule) AppendRule(ID string, CBNet string, CBNetIP string, PublicIP string) {
	CBLogger.Infof("A rule: {%s, %s, %s, %s}\n", ID, CBNet, CBNetIP, PublicIP)
	if !netrule.Contain(ID) { // If HostID doesn't exists, append rule
		netrule.HostID = append(netrule.HostID, ID)
		netrule.HostIPCIDRBlock = append(netrule.HostIPCIDRBlock, CBNet)
		netrule.HostIPAddress = append(netrule.HostIPAddress, CBNetIP)
		netrule.PublicIPAddress = append(netrule.PublicIPAddress, PublicIP)
	}
}

// UpdateRule represents a function to update a rule to the NetworkingRule
func (netrule *NetworkingRule) UpdateRule(id string, hostIPCIDRBlock string, hostIPAddress string, publicIP string) {
	CBLogger.Infof("A rule: {%s, %s, %s, %s}\n", id, hostIPCIDRBlock, hostIPAddress, publicIP)
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

func (netrule NetworkingRule) Contain(x string) bool {
	for _, n := range netrule.HostID {
		if x == n {
			return true
		}
	}
	return false
}
