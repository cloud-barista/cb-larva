package internal

import (
	"fmt"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/data-objects"
	"strconv"
)

// Dynamic Subnet Configuration Protocol
type DynamicSubnetConfigurator struct {
	NetworkingRule dataobjects.NetworkingRule
	subnetIPs      []string
	subnetPrefix   string
	seq            int
}

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

func (dscp *DynamicSubnetConfigurator) UpdateCBNetworkingRule(vmNetworkInfo dataobjects.VMNetworkInformation) {
	CBLogger.Debug("Start.........")

	// Need to update? (A checking function is required)
	// if yes
	// update
	// if no
	// pass

	// To be updated below
	tempNetwork := fmt.Sprint(dscp.subnetIPs[dscp.seq], "/", dscp.subnetPrefix)

	dscp.NetworkingRule.AppendRule(strconv.Itoa(dscp.seq), tempNetwork, dscp.subnetIPs[dscp.seq], vmNetworkInfo.PublicIP)
	CBLogger.Infof("The updated networking rules: %s", dscp.NetworkingRule)
	dscp.seq = (dscp.seq+1)%98 + 2

	CBLogger.Debug("End.........")
}
