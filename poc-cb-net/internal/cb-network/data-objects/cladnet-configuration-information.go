package cbnet

// CLADNetConfigurationInformation represents the configuration information of a Cloud Adaptive Network (CLADNet).
type CLADNetConfigurationInformation struct {
	CLADNetID   string `json:"CLADNetID"`
	CIDRBlock   string `json:"CIDRBlock"`
	GatewayIP   string `json:"gatewayIP"`
	Description string `json:"description"`
}
