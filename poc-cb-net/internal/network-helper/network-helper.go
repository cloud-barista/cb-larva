package nethelper

import (
	"fmt"
	"net"
)

var privateNetworks []*net.IPNet

func init() {
	for _, CIDRBlock := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, privateNetwork, err := net.ParseCIDR(CIDRBlock)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", CIDRBlock, err))
		}
		privateNetworks = append(privateNetworks, privateNetwork)
	}
}

// IsPrivateIP represents if the input IP is private or not.
func IsPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, privateNetwork := range privateNetworks {
		if privateNetwork.Contains(ip) {
			return true
		}
	}
	return false
}

// IncrementIP represents a function to increase IP by input number
func IncrementIP(ip net.IP, inc uint) net.IP {
	i := ip.To4()
	v := uint(i[0])<<24 + uint(i[1])<<16 + uint(i[2])<<8 + uint(i[3])
	v += inc
	v3 := byte(v & 0xFF)
	v2 := byte((v >> 8) & 0xFF)
	v1 := byte((v >> 16) & 0xFF)
	v0 := byte((v >> 24) & 0xFF)
	return net.IPv4(v0, v1, v2, v3)
}
