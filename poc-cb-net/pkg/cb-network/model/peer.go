package cbnet

// Peer represents a host participating in a cloud adaptive network.
type Peer struct {
	CladnetID           string           `json:"cladnetId"`
	HostID              string           `json:"hostId"`
	HostName            string           `json:"hostName"`
	HostPrivateIPv4CIDR string           `json:"hostPrivateIpv4Cidr"`
	HostPrivateIP       string           `json:"hostPrivateIp"`
	HostPublicIP        string           `json:"hostPublicIp"`
	IPv4CIDR            string           `json:"ipv4Cidr"`
	IP                  string           `json:"ip"`
	State               string           `json:"state"`
	Details             CloudInformation `json:"details"`
}

// Peers represents a list of peers.
type Peers struct {
	Peers []Peer `json:"peers"`
}

// CloudInformation represents cloud information.
type CloudInformation struct {
	ProviderName       string `json:"providerName"`
	RegionID           string `json:"regionId"`
	AvailabilityZoneID string `json:"availabilityZoneId"`
	VirtualNetworkID   string `json:"virtualNetworkId"`
	SubnetID           string `json:"subnetId"`
}
