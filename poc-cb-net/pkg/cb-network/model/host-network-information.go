package cbnet

// HostNetworkInformation represents the network information of VM, such as public IP and private networks
type HostNetworkInformation struct {
	HostName            string   `json:"hostName"`
	IsEncrypted         bool     `json:"isEncrypted"`
	PublicIP            string   `json:"publicIPAddress"`
	PrivateIPv4Networks []string `json:"privateIPv4Networks"`
}
