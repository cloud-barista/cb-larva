package poc_cb_net

import (
	"fmt"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/data-objects"
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
	fmt.Println("Update CBNetwork")

	// Need to update? (A checking function is required)
	// if yes
	// update
	// if no
	// pass

	// To be updated below
	tempNetwork := fmt.Sprint(dscp.subnetIPs[dscp.seq], "/", dscp.subnetPrefix)

	dscp.NetworkingRule.AppendRule(strconv.Itoa(dscp.seq), tempNetwork, dscp.subnetIPs[dscp.seq], vmNetworkInfo.PublicIP)
	fmt.Println(dscp.NetworkingRule)
	dscp.seq = (dscp.seq+1)%98 + 2
}
