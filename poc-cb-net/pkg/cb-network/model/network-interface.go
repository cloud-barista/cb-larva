package cbnet

// NetworkInterface represents a network interface in a system with assigned IPs (Typically IPv4, and IPv6)
type NetworkInterface struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	IPv4        string `json:"iPv4"`
	IPv4Network string `json:"iPv4Network"`
	IPv6        string `json:"iPv6"`
	IPv6Network string `json:"iPv6Network"`
}
