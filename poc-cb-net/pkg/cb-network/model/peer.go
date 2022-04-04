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
)

// Peer represents a host's rule of the cloud adaptive network.
type Peer struct {
	CLADNetID          string `json:"cladNetID"`
	HostID             string `json:"hostID"`
	HostName           string `json:"hostName"`
	PrivateIPv4Network string `json:"privateIPv4Network"`
	PrivateIPv4Address string `json:"privateIPv4Address"`
	PublicIPv4Address  string `json:"publicIPv4Address"`
	State              string `json:"state"`
}
