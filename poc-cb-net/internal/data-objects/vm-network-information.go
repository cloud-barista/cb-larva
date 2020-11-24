package dataobjects

type VMNetworkInformation struct {
	PublicIP        string   `json:"PublicIP"`
	PrivateNetworks []string `json:"PrivateNetworks"`
}
