package dataobjects

type IP struct {
	Version   string `json:"Version"`
	IPAddress string `json:"IP"`
	CIDRBlock string `json:"CIDRBlock"`
}

type NetworkInterface struct {
	Name string `json:"Name"`
	IPs  []IP   `json:"IPs"`
}
