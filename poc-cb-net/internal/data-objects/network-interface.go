package dataobjects

// IP represents a set of information related to IP address
type IP struct {
	Version   string `json:"Version"`
	IPAddress string `json:"IP"`
	CIDRBlock string `json:"CIDRBlock"`
}

// NetworkInterface represents a network interface in a system with assigned IPs (Typically IPv4, and IPv6)
type NetworkInterface struct {
	Name string `json:"Name"`
	IPs  []IP   `json:"IPs"`
}
