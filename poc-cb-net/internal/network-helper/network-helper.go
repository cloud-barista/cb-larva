package nethelper

import (
	"fmt"
	"net"
)

var privateNetworks []*net.IPNet
var ip10, ip172, ip192 net.IP
var ipnet10, ipnet172, ipnet192 *net.IPNet

func init() {
	// Initialize private networks
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

	// Initialize IPs and networks of each private network
	ip10, ipnet10, _ = net.ParseCIDR("10.0.0.0/8")
	ip172, ipnet172, _ = net.ParseCIDR("172.16.0.0/12")
	ip192, ipnet192, _ = net.ParseCIDR("192.168.0.0/16")
}

// Models

// AvailableIPv4PrivateAddressSpaces represents the specification of a Cloud Adaptive Network (CLADNet).
type AvailableIPv4PrivateAddressSpaces struct {
	AddressSpaces10  []string `json:"AddressSpaces10"`
	AddressSpaces172 []string `json:"AddressSpaces172"`
	AddressSpaces192 []string `json:"AddressSpaces192"`
}

// Functions

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

// initMap initializes a map with an integer key starting at 1
func initMap(keyFrom int, keyTo int, initValue bool) map[int]bool {
	m := make(map[int]bool)
	for i := keyFrom; i <= keyTo; i++ {
		m[i] = initValue
	}
	return m
}

// GetAvailableIPv4PrivateAddressSpaces represents a function to check and return available CIDR blocks
func GetAvailableIPv4PrivateAddressSpaces(ips []string) AvailableIPv4PrivateAddressSpaces {
	// CBLogger.Debug("Start.........")

	// CBLogger.Tracef("IPs: %v", ips)

	prefixMap10 := initMap(8, 32, true)
	prefixMap172 := initMap(12, 32, true)
	prefixMap192 := initMap(16, 32, true)

	for _, ipStr := range ips {
		// CBLogger.Tracef("i: %v", i)
		// CBLogger.Tracef("IP: %v", ipStr)

		ip, ipnet, _ := net.ParseCIDR(ipStr)
		// Get CIDR Prefix
		cidrPrefix, _ := ipnet.Mask.Size()

		if ipnet10.Contains(ip) {
			// CBLogger.Tracef("'%s' contains '%s/%v'", ipnet10, ip, cidrPrefix)
			prefixMap10[cidrPrefix] = false

		} else if ipnet172.Contains(ip) {
			// CBLogger.Tracef("'%s' contains '%s/%v'", ipnet172, ip, cidrPrefix)
			prefixMap172[cidrPrefix] = false

		} else if ipnet192.Contains(ip) {
			// CBLogger.Tracef("'%s' contains '%s/%v'", ipnet192, ip, cidrPrefix)
			prefixMap192[cidrPrefix] = false

		} else {
			// CBLogger.Tracef("Nothing contains '%s/%v'", ip, cidrPrefix)
			fmt.Printf("Nothing contains '%s/%v'\n", ip, cidrPrefix)
		}
	}

	// net10
	availableIPNet10 := make([]string, 32)
	j := 0
	for cidrPrefix, isTrue := range prefixMap10 {
		if isTrue {
			ipNet := fmt.Sprint(ip10, "/", cidrPrefix)
			// CBLogger.Tracef("'%s' is possible for a virtual network.", ipNet)
			availableIPNet10[j] = ipNet
			j++
		}
	}

	// net172
	availableIPNet172 := make([]string, 32)
	j = 0
	for cidrPrefix, isTrue := range prefixMap172 {
		if isTrue {
			ipNet := fmt.Sprint(ip172, "/", cidrPrefix)
			// CBLogger.Tracef("'%s' is possible for a virtual network.", ipNet)
			availableIPNet172[j] = ipNet
			j++
		}
	}

	// net192
	availableIPNet192 := make([]string, 32)
	j = 0
	for cidrPrefix, isTrue := range prefixMap192 {
		if isTrue {
			ipNet := fmt.Sprint(ip192, "/", cidrPrefix)
			// CBLogger.Tracef("'%s' is possible for a virtual network.", ipNet)
			availableIPNet192[j] = ipNet
			j++
		}
	}

	// CBLogger.Tracef("Available IPNets in 10.0.0.0/8 : %v", availableIPNet10)
	// CBLogger.Tracef("Available IPNets in 172.16.0.0/12 : %v", availableIPNet172)
	// CBLogger.Tracef("Available IPNets in 192.168.0.0/16 : %v", availableIPNet192)

	fmt.Printf("Available IPNets in 10.0.0.0/8 : %v\n", availableIPNet10)
	fmt.Printf("Available IPNets in 172.16.0.0/12 : %v\n", availableIPNet172)
	fmt.Printf("Available IPNets in 192.168.0.0/16 : %v\n", availableIPNet192)

	// CBLogger.Debug("End.........")
	return AvailableIPv4PrivateAddressSpaces{
		AddressSpaces10:  availableIPNet10,
		AddressSpaces172: availableIPNet172,
		AddressSpaces192: availableIPNet192}
}
