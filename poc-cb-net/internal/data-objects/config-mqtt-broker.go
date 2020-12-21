package dataobjects

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

type ConfigMQTTBroker struct {
	MQTTBrokerIP   string `json:"MQTTBrokerIP"`
	MQTTBrokerPort string `json:"MQTTBrokerPort"`
}

func LoadConfigMQTTBroker() (ConfigMQTTBroker, error) {
	var config ConfigMQTTBroker

	path := filepath.Join("poc-cb-net", "configs", "mqtt-broker.json")
	file, err := os.Open(path)
	if err != nil {
		log.Fatal("can't open config file: ", err)
	}
	defer func() {
		err := file.Close()
		if err != nil{
			log.Fatal("can't close the file", err)
		}
	}()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("can't decode config JSON: ", err)
	}

	if err := file.Close(); err != nil {
		return config, err
	}
	return config, err
}
