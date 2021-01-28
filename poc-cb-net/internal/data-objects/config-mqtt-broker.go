package dataobjects

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

// ConfigMQTTBroker represents the configuration information for a MQTT Broker
type ConfigMQTTBroker struct {
	MQTTBrokerHost             string `json:"MQTTBrokerHost"`
	MQTTBrokerPort             string `json:"MQTTBrokerPort"`
	MQTTBrokerPortForWebsocket string `json:"MQTTBrokerPortForWebSocket"`
}

// LoadConfigMQTTBroker represents a function to read a MQTT Broker's configuration information from a file
func LoadConfigMQTTBroker() (ConfigMQTTBroker, error) {
	var config ConfigMQTTBroker

	path := filepath.Join("..", "..", "configs", "mqtt-broker.json")

	// Open config file
	file, errOpen := os.Open(path)
	if errOpen != nil {
		log.Fatal("can't open config file: ", errOpen)
	}

	// Perform error handling
	defer func() {
		errClose := file.Close()
		if errClose != nil {
			log.Fatal("can't close the file", errClose)
		}
	}()

	// Make decoder instance
	decoder := json.NewDecoder(file)
	// Decode config text to json
	errOpen = decoder.Decode(&config)
	if errOpen != nil {
		log.Fatal("can't decode config JSON: ", errOpen)
	}

	return config, errOpen
}
