package cbnet

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger

func init() {
	fmt.Println("\nStart......... init() of networking-rule.go")

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
	}
	fmt.Println("End......... init() of networking-rule.go")
	fmt.Println("")
}

// NetworkingRule represents a networking rule of the cloud adaptive network.
// It is used for tunneling between hosts(e.g., VMs).
type NetworkingRule struct {
	CladnetID  string   `json:"cladnetId"`
	HostID     []string `json:"hostId"`
	HostName   []string `json:"hostName"`
	PeerIP     []string `json:"peerIP"`
	SelectedIP []string `json:"selectedIP"`
	PeerScope  []string `json:"peerScope"`
	State      []string `json:"state"`
}

// AppendRule represents a function to append a rule to the NetworkingRule
func (netrule *NetworkingRule) AppendRule(id, name, peerIP, selectedIP, peerScope, state string) {
	CBLogger.Tracef("A rule: {%s, %s, %s, %s, %s, %s}", id, name, peerIP, selectedIP, peerScope, state)
	if !netrule.Contain(id) { // If HostID doesn't exists, append rule
		netrule.HostID = append(netrule.HostID, id)
		netrule.HostName = append(netrule.HostName, name)
		netrule.PeerIP = append(netrule.PeerIP, peerIP)
		netrule.SelectedIP = append(netrule.SelectedIP, selectedIP)
		netrule.PeerScope = append(netrule.PeerScope, peerScope)
		netrule.State = append(netrule.State, state)
	}
}

// UpdateRule represents a function to update a rule to the NetworkingRule
func (netrule *NetworkingRule) UpdateRule(id, name, peerIP, selectedIP, peerScope, state string) {
	CBLogger.Tracef("A rule: {%s, %s, %s, %s, %s, %s}", id, name, peerIP, selectedIP, peerScope, state)
	if netrule.Contain(id) { // If HostID exists, update rule
		index := netrule.GetIndexOfHostID(id)
		if name != "" {
			netrule.HostName[index] = name
		}
		if state != "" {
			netrule.State[index] = state
		}
		if peerScope != "" {
			netrule.PeerScope[index] = peerScope
		}
		netrule.SelectedIP[index] = selectedIP
	} else {
		netrule.AppendRule(id, name, peerIP, selectedIP, peerScope, state)
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

// GetIndexOfPeerIP represents a function to find and return an index of a peer IP address from NetworkingRule
func (netrule NetworkingRule) GetIndexOfPeerIP(hostIPAddress string) int {
	return netrule.find(netrule.PeerIP, hostIPAddress)
}

// GetIndexOfSelectedIP represents a function to find and return an index of a selected IP address from NetworkingRule
func (netrule NetworkingRule) GetIndexOfSelectedIP(publicIP string) int {
	return netrule.find(netrule.SelectedIP, publicIP)
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
