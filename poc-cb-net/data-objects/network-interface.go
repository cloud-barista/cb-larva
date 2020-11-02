package dataobjects


type IP struct{
	Version		string	`json:"Version"`
	IPAddress	string	`json:"IP"`
	NetworkID	string	`json:"NetworkID"`
}

type NetworkInterface struct{
	Name	string  `json:"Name"`
	IPs		[]IP 	`json:"IPs"`
}

