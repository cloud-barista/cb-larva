package cbnet

const (

	// Configuring is const for the configuring state
	Configuring = "configuring"

	// Tunneling is const for the tunneling state
	Tunneling = "tunneling"

	// Closing is const for the closing state
	Closing = "closing"

	// Released is const for the released state
	Released = "released"

	// Failed is const for the failed state
	Failed = "failed"
)

// Peer represents a host participating in a cloud adaptive network.
type Peer struct {
	CLADNetID          string `json:"cladNetID"`
	HostID             string `json:"hostID"`
	HostName           string `json:"hostName"`
	HostPrivateNetwork string `json:"hostPrivateNetwork"`
	HostPrivateIP      string `json:"hostPrivateIP"`
	HostPublicIP       string `json:"hostPublicIP"`
	IPNetwork          string `json:"ipNetwork"`
	IP                 string `json:"ip"`
	State              string `json:"state"`
}
