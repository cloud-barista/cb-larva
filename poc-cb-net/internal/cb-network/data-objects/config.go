package cbnet

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

// MQTTBrokerConfig represents the configuration information for a MQTT Broker
type MQTTBrokerConfig struct {
	Host             string `yaml:"host"`
	Port             string `yaml:"port"`
	PortForWebsocket string `yaml:"port_for_websocket"`
}

// Config represents the configuration information for cb-network
type Config struct {
	MQTTBroker MQTTBrokerConfig `yaml:"mqtt_broker"`
}

// LoadConfigs represents a function to read a MQTT Broker's configuration information from a file
func LoadConfigs(path string) (Config, error) {

	filename, _ := filepath.Abs(path)
	yamlFile, err := ioutil.ReadFile(filename)

	var config Config

	err = yaml.Unmarshal(yamlFile, &config)

	if err != nil {
		panic(err)
	}

	return config, err
}
