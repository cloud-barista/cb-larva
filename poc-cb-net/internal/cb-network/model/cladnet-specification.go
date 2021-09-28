package cbnet

// CLADNetSpecification represents the specification of a Cloud Adaptive Network (CLADNet).
type CLADNetSpecification struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Ipv4AddressSpace string `json:"ipv4AddressSpace"`
	Description      string `json:"description"`
}
