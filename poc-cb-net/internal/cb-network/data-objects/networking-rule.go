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
	HostID           []string `json:"hostID"`
	CLADNetCIDRBlock []string `json:"CLADNetCIDRBlock"`
	CLADNetIPAddress []string `json:"CLADNetIPAddress"`
	PublicIPAddress  []string `json:"publicIPAddress"`
}

// AppendRule represents a function to append a rule to the NetworkingRule
func (netrule *NetworkingRule) AppendRule(ID string, CBNet string, CBNetIP string, PublicIP string) {
	CBLogger.Infof("A rule: {%s, %s, %s, %s}\n", ID, CBNet, CBNetIP, PublicIP)
	if netrule.contains(netrule.HostID, ID) { // If HostID exists, update rule
		index := netrule.GetIndexOfID(ID)
		netrule.CLADNetCIDRBlock[index] = CBNet
		netrule.CLADNetIPAddress[index] = CBNetIP
		netrule.PublicIPAddress[index] = PublicIP
	} else { // Else append rule
		netrule.HostID = append(netrule.HostID, ID)
		netrule.CLADNetCIDRBlock = append(netrule.CLADNetCIDRBlock, CBNet)
		netrule.CLADNetIPAddress = append(netrule.CLADNetIPAddress, CBNetIP)
		netrule.PublicIPAddress = append(netrule.PublicIPAddress, PublicIP)
	}
}

// GetIndexOfID represents a function to find and return an index of HostID from NetworkingRule
func (netrule NetworkingRule) GetIndexOfID(ID string) int {
	return netrule.find(netrule.HostID, ID)
}

// GetIndexOfCBNet represents a function to find and return an index of CLADNetCIDRBlock from NetworkingRule
func (netrule NetworkingRule) GetIndexOfCBNet(CBNet string) int {
	return netrule.find(netrule.CLADNetCIDRBlock, CBNet)
}

// GetIndexOfCBNetIP represents a function to find and return an index of CLADNetIPAddress from NetworkingRule
func (netrule NetworkingRule) GetIndexOfCBNetIP(CBNetIP string) int {
	return netrule.find(netrule.CLADNetIPAddress, CBNetIP)
}

// GetIndexOfPublicIP represents a function to find and return an index of PublicIPAddress from NetworkingRule
func (netrule NetworkingRule) GetIndexOfPublicIP(PublicIP string) int {
	return netrule.find(netrule.PublicIPAddress, PublicIP)
}

func (netrule NetworkingRule) find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

func (netrule NetworkingRule) contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
