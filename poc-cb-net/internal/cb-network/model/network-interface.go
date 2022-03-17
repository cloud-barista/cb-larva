package cbnet

// IP represents a set of information related to IP address
type IP struct {
	Version     string `json:"Version"`
	IPAddress   string `json:"IP"`
	IPv4Network string `json:"IPv4Network"`
}

// NetworkInterface represents a network interface in a system with assigned IPs (Typically IPv4, and IPv6)
type NetworkInterface struct {
	Name string `json:"Name"`
	IPs  []IP   `json:"IPs"`
}
