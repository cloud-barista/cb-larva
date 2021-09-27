package cbnet

// HostNetworkInformation represents the network information of VM, such as public IP and private networks
type HostNetworkInformation struct {
	PublicIP                 string   `json:"publicIPAddress"`
	PrivateNetworkCIDRBlocks []string `json:"privateNetworkCIDRBlocks"`
}
