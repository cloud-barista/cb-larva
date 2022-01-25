package etcdkey

const (
	// CloudAdaptiveNetwork is a constant variable of "/registry/cloud-adaptive-network" key
	CloudAdaptiveNetwork = "/registry/cloud-adaptive-network"

	// CLADNetSpecification is a constant variable of "/registry/cloud-adaptive-network/cladnet-specification" key
	CLADNetSpecification = CloudAdaptiveNetwork + "/cladnet-specification"

	// HostNetworkInformation is a constant variable of "/registry/cloud-adaptive-network/host-network-information" key
	HostNetworkInformation = CloudAdaptiveNetwork + "/host-network-information"

	// NetworkingRule is a constant variable of "/registry/cloud-adaptive-network/networking-rule" key
	NetworkingRule = CloudAdaptiveNetwork + "/networking-rule"

	// ControlCommand is a constant variable of "/registry/cloud-adaptive-network/control-command" key
	ControlCommand = CloudAdaptiveNetwork + "/control-command"

	// Status is a constant variable of "/registry/cloud-adaptive-network/status" key
	Status = CloudAdaptiveNetwork + "/status"

	// StatusTestSpecification is a constant variable of "/registry/cloud-adaptive-network/status/test-specification" key
	StatusTestSpecification = Status + "/test-specification"

	// StatusInformation is a constant variable of "/registry/cloud-adaptive-network/status/information" key
	StatusInformation = Status + "/information"

	// Secret is a constant variable of "/registry/cloud-adaptive-network/secret" key
	Secret = CloudAdaptiveNetwork + "/secret"
)
