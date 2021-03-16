package cbnet

// VMNetworkInformation represents the network information of VM, such as public IP and private networks
type VMNetworkInformation struct {
	PublicIP        string   `json:"PublicIP"`
	PrivateNetworks []string `json:"PrivateNetworks"`
}
