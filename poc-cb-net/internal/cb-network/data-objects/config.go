package cbnet

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

// Configs for the both cb-network controller and agent as follows:

// MQTTBrokerConfig represents the configuration information for a MQTT Broker
type MQTTBrokerConfig struct {
	Host             string `yaml:"host"`
	Port             string `yaml:"port"`
	PortForWebsocket string `yaml:"port_for_websocket"`
}

// ETCDConfig represents the configuration information for a etcd cluster
type ETCDConfig struct {
	Endpoints []string `yaml:"endpoints"`
}

// Configs for the cb-network controller as follows:

// AdminWebConfig represents the configuration information for a AdminWeb
type AdminWebConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

// Configs for the cb-network agent as follows:

// CBNetworkConfig represents the configuration information for a cloud adaptive network
type CBNetworkConfig struct {
	CLADNetID string `yaml:"cladnet_id"`
	HostID    string `yaml:"host_id"`
}

// DemoAppConfig represents the boolean of whether to run the demo app or not
type DemoAppConfig struct {
	IsRun bool `yaml:"is_run"`
}

// Config represents the configuration information for cb-network
type Config struct {
	MQTTBroker MQTTBrokerConfig `yaml:"mqtt_broker"`
	ETCD       ETCDConfig       `yaml:"etcd_cluster"`
	AdminWeb   AdminWebConfig   `yaml:"admin_web"`
	CBNetwork  CBNetworkConfig  `yaml:"cb_network"`
	DemoApp    DemoAppConfig    `yaml:"demo_app"`
}

// LoadConfig represents a function to read a MQTT Broker's configuration information from a file
func LoadConfig(path string) (Config, error) {

	filename, _ := filepath.Abs(path)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var config Config

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		panic(err)
	}

	return config, err
}
