package cbnet

// Peer represents a host participating in a cloud adaptive network.
type Peer struct {
	CladnetID           string           `json:"cladnetID"`
	HostID              string           `json:"hostID"`
	HostName            string           `json:"hostName"`
	HostPrivateIPv4CIDR string           `json:"hostPrivateIPv4CIDR"`
	HostPrivateIP       string           `json:"hostPrivateIP"`
	HostPublicIP        string           `json:"hostPublicIP"`
	IPv4CIDR            string           `json:"ipv4CIDR"`
	IP                  string           `json:"ip"`
	State               string           `json:"state"`
	Details             CloudInformation `json:"details"`
}

// CloudInformation represents cloud information.
type CloudInformation struct {
	ProviderName       string `json:"providerName"`
	RegionID           string `json:"regionID"`
	AvailabilityZoneID string `json:"availabilityZoneID"`
	VirtualNetworkID   string `json:"virtualNetworkID"`
	SubnetID           string `json:"subnetID"`
}
