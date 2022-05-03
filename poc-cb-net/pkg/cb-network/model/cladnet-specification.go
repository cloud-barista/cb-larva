package cbnet

// CLADNetSpecification represents the specification of a Cloud Adaptive Network (CLADNet).
type CLADNetSpecification struct {
	CladnetID        string `json:"cladnetId"`
	Name             string `json:"name"`
	Ipv4AddressSpace string `json:"ipv4AddressSpace"`
	Description      string `json:"description"`
	RuleType         string `json:"ruleType"`
}
