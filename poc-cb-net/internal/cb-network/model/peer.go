package cbnet

const (
	// Running is const for the running state
	Running = "running"

	// Suspending is const for the suspended state
	Suspending = "suspending"

	// Suspended is const for the suspended state
	Suspended = "suspended"
)

// Peer represents a host's rule of the cloud adaptive network.
type Peer struct {
	CLADNetID          string `json:"cladNetID"`
	HostID             string `json:"hostID"`
	PrivateIPv4Network string `json:"privateIPv4Network"`
	PrivateIPv4Address string `json:"privateIPv4Address"`
	PublicIPv4Address  string `json:"publicIPv4Address"`
	State              string `json:"state"`
}
