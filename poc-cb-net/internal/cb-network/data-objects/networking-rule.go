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

// NetworkingRules represents a networking rules for tunneling between hosts(e.g., VMs).
type NetworkingRules struct {
	ID       []string
	CBNet    []string
	CBNetIP  []string
	PublicIP []string
}

// AppendRule represents a function to append a rule to the NetworkingRules
func (netrule *NetworkingRules) AppendRule(ID string, CBNet string, CBNetIP string, PublicIP string) {
	CBLogger.Infof("A rule: {%s, %s, %s, %s}\n", ID, CBNet, CBNetIP, PublicIP)
	if netrule.contains(netrule.ID, ID) { // If ID exists, update rule
		index := netrule.GetIndexOfID(ID)
		netrule.CBNet[index] = CBNet
		netrule.CBNetIP[index] = CBNetIP
		netrule.PublicIP[index] = PublicIP
	} else { // Else append rule
		netrule.ID = append(netrule.ID, ID)
		netrule.CBNet = append(netrule.CBNet, CBNet)
		netrule.CBNetIP = append(netrule.CBNetIP, CBNetIP)
		netrule.PublicIP = append(netrule.PublicIP, PublicIP)
	}
}

// GetIndexOfID represents a function to find and return an index of ID from NetworkingRules
func (netrule NetworkingRules) GetIndexOfID(ID string) int {
	return netrule.find(netrule.ID, ID)
}

// GetIndexOfCBNet represents a function to find and return an index of CBNet from NetworkingRules
func (netrule NetworkingRules) GetIndexOfCBNet(CBNet string) int {
	return netrule.find(netrule.CBNet, CBNet)
}

// GetIndexOfCBNetIP represents a function to find and return an index of CBNetIP from NetworkingRules
func (netrule NetworkingRules) GetIndexOfCBNetIP(CBNetIP string) int {
	return netrule.find(netrule.CBNetIP, CBNetIP)
}

// GetIndexOfPublicIP represents a function to find and return an index of PublicIP from NetworkingRules
func (netrule NetworkingRules) GetIndexOfPublicIP(PublicIP string) int {
	return netrule.find(netrule.PublicIP, PublicIP)
}

func (netrule NetworkingRules) find(a []string, x string) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return -1
}

func (netrule NetworkingRules) contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
