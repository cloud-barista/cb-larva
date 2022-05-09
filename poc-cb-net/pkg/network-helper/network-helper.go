package nethelper

import (
	"errors"
	"fmt"
	"math"
	"net"
	"sort"
)

var privateNetworks []*net.IPNet
var ip10, ip172, ip192 net.IP
var ipnet10, ipnet172, ipnet192 *net.IPNet

func init() {
	// Initialize private networks
	for _, IPNetwork := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, privateNetwork, err := net.ParseCIDR(IPNetwork)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", IPNetwork, err))
		}
		privateNetworks = append(privateNetworks, privateNetwork)
	}

	// Initialize IPs and networks of each private network
	ip10, ipnet10, _ = net.ParseCIDR("10.0.0.0/8")
	ip172, ipnet172, _ = net.ParseCIDR("172.16.0.0/12")
	ip192, ipnet192, _ = net.ParseCIDR("192.168.0.0/16")
}

// Models

// IPNetworks represents a list of IP Network, such as 10.10.10.10/10, 172.16.10.10/16, and 192.168.10.10/24.
type IPNetworks struct {
	IPNetworks []string `json:"ipNetworks"`
}

// AvailableIPv4PrivateAddressSpaces represents the specification of a Cloud Adaptive Network (CLADNet).
type AvailableIPv4PrivateAddressSpaces struct {
	RecommendedIPv4PrivateAddressSpace string   `json:"recommendedIPv4PrivateAddressSpace"`
	AddressSpace10s                    []string `json:"addressSpace10S"`
	AddressSpace172s                   []string `json:"addressSpace172S"`
	AddressSpace192s                   []string `json:"addressSpace192S"`
}

// Functions

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

// getDescendingOrderedKeysOfMap
func getDescendingOrderedKeysOfMap(m map[int]bool) []int {
	// 	m := map[string]int{
	// 	"Alice": 2,
	// 	"Cecil": 1,
	// 	"Bob":   3,
	// }

	sortedKeys := make([]int, 0, len(m))

	for k := range m {
		// fmt.Println("k", k)
		sortedKeys = append(sortedKeys, k)
	}

	// Sort keys in descending order
	sort.Sort(sort.Reverse(sort.IntSlice(sortedKeys)))

	return sortedKeys
}

// GetAvailableIPv4PrivateAddressSpaces represents a function to check and return the available IPv4 private address spaces
func GetAvailableIPv4PrivateAddressSpaces(strIPv4CIDRs []string) *AvailableIPv4PrivateAddressSpaces {
	// CBLogger.Debug("Start.........")

	// CBLogger.Tracef("IPs: %v", ips)

	prefixMap10 := initMap(9, 32, true)
	prefixMap172 := initMap(13, 32, true)
	prefixMap192 := initMap(17, 32, true)

	// Mark a CIDR prefix in the received list of IP networks
	for _, ipStr := range strIPv4CIDRs {
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
	availableIPNet10s := []string{}
	// Get sorted keys of a map because the Go map doesn't support ordering
	sortedCIDRPrefixes10s := getDescendingOrderedKeysOfMap(prefixMap10)
	for _, cidrPrefix := range sortedCIDRPrefixes10s {
		if prefixMap10[cidrPrefix] {
			ipNet := fmt.Sprint(ip10, "/", cidrPrefix)
			// CBLogger.Tracef("'%s' is possible for a virtual network.", ipNet)
			availableIPNet10s = append(availableIPNet10s, ipNet)
		}
	}

	// net172
	availableIPNet172s := []string{}
	// Get sorted keys of a map because the Go map doesn't support ordering
	sortedCIDRPrefixes172s := getDescendingOrderedKeysOfMap(prefixMap172)
	for _, cidrPrefix := range sortedCIDRPrefixes172s {
		if prefixMap172[cidrPrefix] {
			ipNet := fmt.Sprint(ip172, "/", cidrPrefix)
			// CBLogger.Tracef("'%s' is possible for a virtual network.", ipNet)
			availableIPNet172s = append(availableIPNet172s, ipNet)
		}
	}

	// net192
	availableIPNet192s := []string{}
	// Get sorted keys of a map because the Go map doesn't support ordering
	sortedCIDRPrefixes192s := getDescendingOrderedKeysOfMap(prefixMap192)
	for _, cidrPrefix := range sortedCIDRPrefixes192s {
		if prefixMap172[cidrPrefix] {
			ipNet := fmt.Sprint(ip192, "/", cidrPrefix)
			// CBLogger.Tracef("'%s' is possible for a virtual network.", ipNet)
			availableIPNet192s = append(availableIPNet192s, ipNet)
		}
	}

	// CBLogger.Tracef("Available IPNets in 10.0.0.0/8 : %v", availableIPNet10)
	// CBLogger.Tracef("Available IPNets in 172.16.0.0/12 : %v", availableIPNet172)
	// CBLogger.Tracef("Available IPNets in 192.168.0.0/16 : %v", availableIPNet192)

	fmt.Printf("Available IPNets in 10.0.0.0/8 : %v\n", availableIPNet10s)
	fmt.Printf("Available IPNets in 172.16.0.0/12 : %v\n", availableIPNet172s)
	fmt.Printf("Available IPNets in 192.168.0.0/16 : %v\n", availableIPNet192s)

	availableIPv4PrivateAddressSpaces := &AvailableIPv4PrivateAddressSpaces{}

	availableIPv4PrivateAddressSpaces.AddressSpace10s = availableIPNet10s
	availableIPv4PrivateAddressSpaces.AddressSpace172s = availableIPNet172s
	availableIPv4PrivateAddressSpaces.AddressSpace192s = availableIPNet192s

	// Recommend an IPv4AddressSpace
	numberOfHosts := len(strIPv4CIDRs) + 3 // Network address, Broadcase address, and a reserved IP for gateway
	fmt.Printf("NumberOfHosts: %v\n", numberOfHosts)
	neededPrefix := 32 - int(math.Ceil(math.Log2(float64(numberOfHosts))))
	fmt.Printf("NeededPrefix: %v\n", neededPrefix)
	strIPNet, err := recommendAnIPv4AddressSpace(neededPrefix, availableIPv4PrivateAddressSpaces)
	if err != nil {
		fmt.Print(err)
	}

	availableIPv4PrivateAddressSpaces.RecommendedIPv4PrivateAddressSpace = strIPNet

	fmt.Printf("Available IPv4 private address spaces: %v\n", availableIPv4PrivateAddressSpaces)

	// CBLogger.Debug("End.........")
	return availableIPv4PrivateAddressSpaces
}

func recommendAnIPv4AddressSpace(neededPrefix int, availableIPv4PrivateAddressSpaces *AvailableIPv4PrivateAddressSpaces) (string, error) {

	// Recommended IPv4 address space from small space to large space

	// Find and return a recommended IPv4 address space under 192.168.0.0/16
	for _, ipnetStr := range availableIPv4PrivateAddressSpaces.AddressSpace192s {
		_, ipnet, _ := net.ParseCIDR(ipnetStr)
		// Get CIDR Prefix
		cidrPrefix, _ := ipnet.Mask.Size()
		if cidrPrefix <= neededPrefix {
			fmt.Printf("Recommended an IPv4 address space: %v\n", ipnetStr)
			return ipnetStr, nil
		}
	}

	// Find and return a recommended IPv4 address space under 172.16.0.0/12
	for _, ipnetStr := range availableIPv4PrivateAddressSpaces.AddressSpace172s {
		_, ipnet, _ := net.ParseCIDR(ipnetStr)
		// Get CIDR Prefix
		cidrPrefix, _ := ipnet.Mask.Size()
		if cidrPrefix <= neededPrefix {
			fmt.Printf("Recommended an IPv4 address space: %v\n", ipnetStr)
			return ipnetStr, nil
		}
	}

	// Find and return a recommended IPv4 address space under 10.0.0.0/8
	for _, ipnetStr := range availableIPv4PrivateAddressSpaces.AddressSpace10s {
		_, ipnet, _ := net.ParseCIDR(ipnetStr)
		// Get CIDR Prefix
		cidrPrefix, _ := ipnet.Mask.Size()
		if cidrPrefix <= neededPrefix {
			fmt.Printf("Recommended an IPv4 address space: %v\n", ipnetStr)
			return ipnetStr, nil
		}
	}
	return "", errors.New("no appropriate IPv4PrivateAddressSpace exists")
}
