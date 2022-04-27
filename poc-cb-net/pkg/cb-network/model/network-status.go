package cbnet

// TestSpecification represents the specification of a Cloud Adaptive Network (CLADNet).
type TestSpecification struct {
	CladnetID  string `json:"cladnetID"`
	TrialCount int    `json:"trialCount"`
}

// NetworkStatus represents the statistics of a Cloud Adaptive Network (CLADNet).
type NetworkStatus struct {
	InterHostNetworkStatus []InterHostNetworkStatus `json:"interHostNetworkStatus"`
}

// InterHostNetworkStatus represents the network performance between two virtual machines in a CLADNet.
type InterHostNetworkStatus struct {
	SourceIP        string  `json:"sourceIP"`
	SourceName      string  `json:"sourceName"`
	DestinationIP   string  `json:"destinationIP"`
	DestinationName string  `json:"destinationName"`
	MininumRTT      float64 `json:"minimunRTT"`
	AverageRTT      float64 `json:"averageRTT"`
	MaximumRTT      float64 `json:"maximumRTT"`
	StdDevRTT       float64 `json:"stddevRTT"`
	PacketsReceive  int     `json:"packetsReceive"`
	PacketsLoss     int     `json:"packetLoss"`
	BytesReceived   int     `json:"bytesReceived"`
}
