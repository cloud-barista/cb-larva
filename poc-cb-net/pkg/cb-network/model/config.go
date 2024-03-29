package cbnet

import (
	"io/ioutil"
	"path/filepath"

	yaml "gopkg.in/yaml.v3"
)

// A config for the both cb-network controller and agent as follows:

// ETCDConfig represents the configuration information for a etcd cluster
type ETCDConfig struct {
	Endpoints []string `yaml:"endpoints"`
}

// A config for the cb-network service and cb-network admin-web as follows:

// ServiceConfig represnets the configuration information for a gRPC server
type ServiceConfig struct {
	Endpoint string `yaml:"endpoint"`
	Port     string `yaml:"port"`
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
	CLADNetID string     `yaml:"cladnet_id"`
	Host      HostConfig `yaml:"host"`
}

// HostConfig represents the configuration information for a host in a cloud adaptvie network
type HostConfig struct {
	Name                 string `yaml:"name"`
	NetworkInterfaceName string `yaml:"network_interface_name"`
	TunnelingPort        string `yaml:"tunneling_port"`
	IsEncrypted          bool   `yaml:"is_encrypted"`
}

// Config represents the configuration information for cb-network
type Config struct {
	ETCD              ETCDConfig      `yaml:"etcd_cluster"`
	AdminWeb          AdminWebConfig  `yaml:"admin_web"`
	CBNetwork         CBNetworkConfig `yaml:"cb_network"`
	Service           ServiceConfig   `yaml:"service"`
	ServiceCallMethod string          `yaml:"service_call_method"`
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
