package cbnet

// AvailableIPv4PrivateAddressSpaces represents the specification of a Cloud Adaptive Network (CLADNet).
type AvailableIPv4PrivateAddressSpaces struct {
	AddressSpaces10  []string `json:"AddressSpaces10"`
	AddressSpaces172 []string `json:"AddressSpaces172"`
	AddressSpaces192 []string `json:"AddressSpaces192"`
}
