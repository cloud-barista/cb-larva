package cbnet

// NetworkInterface represents a network interface in a system with assigned IPs (Typically IPv4, and IPv6)
type NetworkInterface struct {
	Name     string `json:"name"`
	Version  string `json:"version"`
	IPv4     string `json:"ipv4"`
	IPv4CIDR string `json:"ipv4Cidr"`
	IPv6     string `json:"ipv6"`
	IPv6CIDR string `json:"ipv6Cidr"`
}
