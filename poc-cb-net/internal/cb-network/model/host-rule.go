package cbnet

// HostRule represents a host's rule of the cloud adaptive network.
type HostRule struct {
	CLADNetID          string `json:"cladNetID"`
	HostID             string `json:"hostID"`
	PrivateIPv4Network string `json:"privateIPv4Network"`
	PrivateIPv4Address string `json:"privateIPv4Address"`
	PublicIPv4Address  string `json:"publicIPv4Address"`
}
