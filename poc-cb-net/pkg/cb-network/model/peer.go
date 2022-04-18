package cbnet

// Peer represents a host participating in a cloud adaptive network.
type Peer struct {
	CLADNetID            string `json:"cladNetID"`
	HostID               string `json:"hostID"`
	HostName             string `json:"hostName"`
	HostPrivateIPNetwork string `json:"hostPrivateIPNetwork"`
	HostPrivateIP        string `json:"hostPrivateIP"`
	HostPublicIP         string `json:"hostPublicIP"`
	IPNetwork            string `json:"ipNetwork"`
	IP                   string `json:"ip"`
	State                string `json:"state"`
}
