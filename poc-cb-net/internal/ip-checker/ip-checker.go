package ipchkr

import (
	"fmt"
	"net"
)

var privateNetworks []*net.IPNet

func init(){
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
