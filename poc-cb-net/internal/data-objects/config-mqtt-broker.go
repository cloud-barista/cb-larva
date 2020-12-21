package dataobjects

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

type ConfigMQTTBroker struct {
	MQTTBrokerIP   string `json:"MQTTBrokerIP"`
	MQTTBrokerPort string `json:"MQTTBrokerPort"`
}

func LoadConfigMQTTBroker() (ConfigMQTTBroker, error) {
	var config ConfigMQTTBroker
	var path string

	if runtime.GOOS == "windows" {
		path = filepath.Join("poc-cb-net","configs","mqtt-broker.json")
	} else {
		path = filepath.Join("..", "..", "configs","mqtt-broker.json")
	}

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

	return config, err
}
