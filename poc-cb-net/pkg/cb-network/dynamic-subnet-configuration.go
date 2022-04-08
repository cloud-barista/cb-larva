package cbnet

import (
	"fmt"
	"strconv"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
)

// DynamicSubnetConfigurator represents a configurator for Dynamic Subnet Configuration Protocol
type DynamicSubnetConfigurator struct {
	NetworkingRules model.NetworkingRule
	subnetIPs       []string
	subnetPrefix    string
	seq             int
}

// NewDynamicSubnetConfigurator represents a constructor.
func NewDynamicSubnetConfigurator() *DynamicSubnetConfigurator {
	temp := &DynamicSubnetConfigurator{}

	// To be updated below
	temp.subnetPrefix = "23"
	for i := 0; i < 100; i++ {
		temp.subnetIPs = append(temp.subnetIPs, fmt.Sprint("192.168.10.", i))
	}
	temp.seq = 2

	return temp
}

// UpdateCBNetworkingRules represents a function to update networking rules
func (dscp *DynamicSubnetConfigurator) UpdateCBNetworkingRules(vmNetworkInfo model.HostNetworkInformation) {
	CBLogger.Debug("Start.........")

	// Need to update? (A checking function is required)
	// if yes
	// update
	// if no
	// pass

	// To be updated below
	tempNetwork := fmt.Sprint(dscp.subnetIPs[dscp.seq], "/", dscp.subnetPrefix)

	dscp.NetworkingRules.AppendRule(strconv.Itoa(dscp.seq), "tempName", tempNetwork, dscp.subnetIPs[dscp.seq], vmNetworkInfo.PublicIP, model.Configuring)
	CBLogger.Infof("The updated networking rules: %s", dscp.NetworkingRules)
	dscp.seq = (dscp.seq+1)%98 + 2

	CBLogger.Debug("End.........")
}
