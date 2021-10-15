package cbnet

import (
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// A config for the both cb-network controller and agent as follows:

// ETCDConfig represents the configuration information for a etcd cluster
type ETCDConfig struct {
	Endpoints []string `yaml:"endpoints"`
}

// A config for the cb-network controller as follows:

// AdminWebConfig represents the configuration information for a AdminWeb
type AdminWebConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

// A config for the cb-network agent as follows:

// CBNetworkConfig represents the configuration information for a cloud adaptive network
type CBNetworkConfig struct {
	CLADNetID string `yaml:"cladnet_id"`
	HostID    string `yaml:"host_id"`
}

// A config for the grpc as follows:

// GRPCConfig represnets the configuration information for a gRPC server
type GRPCConfig struct {
	ServiceEndpoint string `yaml:"service_endpoint"`
	ServerPort      string `yaml:"server_port"`
	GatewayPort     string `yaml:"gateway_port"`
}

// DemoAppConfig represents the boolean of whether to run the demo app or not
type DemoAppConfig struct {
	IsRun bool `yaml:"is_run"`
}

// Config represents the configuration information for cb-network
type Config struct {
	ETCD      ETCDConfig      `yaml:"etcd_cluster"`
	AdminWeb  AdminWebConfig  `yaml:"admin_web"`
	CBNetwork CBNetworkConfig `yaml:"cb_network"`
	GRPC      GRPCConfig      `yaml:"grpc"`
	DemoApp   DemoAppConfig   `yaml:"demo_app"`
}

// LoadConfig represents a function to read the configuration information from a file
func LoadConfig(path string) (Config, error) {

	filename, _ := filepath.Abs(path)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var configTemp Config

	err = yaml.Unmarshal(yamlFile, &configTemp)
	if err != nil {
		panic(err)
	}

	return configTemp, err
}
