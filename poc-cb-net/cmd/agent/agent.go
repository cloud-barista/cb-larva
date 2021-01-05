package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/app"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/data-objects"
	cblog "github.com/cloud-barista/cb-log"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"math/big"
	"os"
	"path/filepath"
)

var CBNet *internal.CBNetwork
var channel chan bool
var CBLogger *logrus.Logger

func init() {
	// cblog is a global variable.
	configPath := filepath.Join("..", "..", "configs", "log_conf.yaml")
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", configPath)
}

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	CBLogger.Info("Start.........")
	CBLogger.Debugf("Received TOPIC : %s\n", msg.Topic())
	CBLogger.Debugf("MSG: %s\n", msg.Payload())

	if msg.Topic() == "cb-net/networking-rule" {

		var networkingRule dataobjects.NetworkingRule

		err := json.Unmarshal(msg.Payload(), &networkingRule)
		if err != nil {
			panic(err)
		}
		CBLogger.Debug("Unmarshalled JSON")
		CBLogger.Debug(networkingRule)

		prettyJSON, _ := json.MarshalIndent(networkingRule, "", "\t")
		CBLogger.Debug("Pretty JSON")
		CBLogger.Debug(string(prettyJSON))

		CBLogger.Debug("Update the networking rule")
		CBNet.SetNetworkingRule(networkingRule)
		if !CBNet.IsRunning() {
			CBNet.StartCBNetworking(channel)
		}
		CBLogger.Info("End.........")
	}
}

func main() {
	CBLogger.Info("Start.........")

	var arg string
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	channel = make(chan bool)

	// Random number to avoid MQTT client ID duplication
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		panic(err)
	}
	CBLogger.Debugf("Random number: %d\t", n)

	// Create CBNetwork instance with port, which is tunneling port
	CBNet = internal.NewCBNetwork("cbnet0", 20000)

	// Load a config of MQTTBroker
	config, _ := dataobjects.LoadConfigMQTTBroker()
	// Create a endpoint link of MQTTBroker
	server := "tcp://" + config.MQTTBrokerHost + ":" + config.MQTTBrokerPort

	// Create a ClientOptions struct setting the broker address, clientid, turn
	// off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker(server)
	opts.SetClientID(fmt.Sprint("cb-net-agent-", n))
	opts.SetDefaultPublishHandler(f)

	// Create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe to the topic /go-mqtt/sample and request messages to be delivered
	// at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe("cb-net/networking-rule", 0, nil); token.Wait() && token.Error() != nil {
		CBLogger.Error(token.Error())
		os.Exit(1)
	}

	// Get the VM network information
	temp := CBNet.GetVMNetworkInformation()
	doc, _ := json.Marshal(temp)
	CBLogger.Debug(string(doc))

	// Publish a message to /cb-net/vm-network-information at qos 1 and wait for the receipt
	// from the server after sending each message
	token := c.Publish("cb-net/vm-network-information", 0, false, doc)
	token.Wait()

	//go CBNet.RunEncapsulation(channel)
	//go CBNet.RunDecapsulation(channel)
	go CBNet.RunTunneling(channel)
	if arg == "demo" {
		go app.PitcherAndCatcher(CBNet, channel)
	}

	// Block to stop this program
	fmt.Println("Press the Enter Key to stop anytime")
	fmt.Scanln()

	//Unsubscribe from /cb-net/vm-network-information"
	if token := c.Unsubscribe("cb-net/networking-rule"); token.Wait() && token.Error() != nil {
		CBLogger.Error(token.Error())
		os.Exit(1)
	}

	// Disconnect MQTT Client
	c.Disconnect(250)
}
