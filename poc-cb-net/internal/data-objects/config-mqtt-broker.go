package dataobjects

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type ConfigMQTTBroker struct {
	MQTTBrokerHost             string `json:"MQTTBrokerHost"`
	MQTTBrokerPort             string `json:"MQTTBrokerPort"`
	MQTTBrokerPortForWebsocket string `json:"MQTTBrokerPortForWebSocket"`
}

func LoadConfigMQTTBroker() (ConfigMQTTBroker, error) {
	var config ConfigMQTTBroker

	path := filepath.Join("..", "..", "configs", "mqtt-broker.json")

	// Open config file
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("can't open config file: ", err)
	}
	// Perform error handling
	defer func() {
		err := file.Close()
		if err != nil {
			log.Fatal("can't close the file", err)
		}
	}()

	// Make decoder instance
	decoder := json.NewDecoder(file)
	// Decode config text to json
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("can't decode config JSON: ", err)
	}

	return config, err
}
