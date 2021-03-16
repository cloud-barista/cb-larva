package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/app"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/data-objects"
	cblog "github.com/cloud-barista/cb-log"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
	"math/big"
	"os"
	"path/filepath"
)

// CBNet represents a network for the multi-cloud.
var CBNet *cbnet.CBNetwork
var channel chan bool

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger

func init() {
	// cblog is a global variable.
	configPath := filepath.Join("..", "..", "configs", "log_conf.yaml")
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", configPath)
}

// MyMQTTMessageHandler is a default message handler.
func MyMQTTMessageHandler(client MQTT.Client, msg MQTT.Message) {
	CBLogger.Debug("Start.........")
	CBLogger.Debugf("Received TOPIC : %s\n", msg.Topic())
	CBLogger.Debugf("MSG: %s\n", msg.Payload())

	if msg.Topic() == "cb-net/networking-rule" {
		var networkingRule dataobjects.NetworkingRules

		err := json.Unmarshal(msg.Payload(), &networkingRule)
		if err != nil {
			CBLogger.Panic(err)
		}
		CBLogger.Trace("Unmarshalled JSON")
		CBLogger.Trace(networkingRule)

		prettyJSON, _ := json.MarshalIndent(networkingRule, "", "\t")
		CBLogger.Trace("Pretty JSON")
		CBLogger.Trace(string(prettyJSON))

		CBLogger.Info("Update the networking rule")
		CBNet.SetNetworkingRules(networkingRule)
		if !CBNet.IsRunning() {
			CBNet.StartCBNetworking(channel)
		}
	}
	CBLogger.Debug("End.........")
}

var f MQTT.MessageHandler = MyMQTTMessageHandler

func main() {
	CBLogger.Debug("Start.........")

	var arg string
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	channel = make(chan bool)

	// Random number to avoid MQTT client ID duplication
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Random number: %d\t", n)

	// Create CBNetwork instance with port, which is tunneling port
	CBNet = cbnet.NewCBNetwork("cbnet0", 20000)

	// Load config
	configPath := filepath.Join("..", "..", "configs", "config.yaml")
	config, _ := dataobjects.LoadConfigs(configPath)
	// Create a endpoint link of MQTTBroker
	server := "tcp://" + config.MQTTBroker.Host + ":" + config.MQTTBroker.Port

	// Create a ClientOptions struct setting the broker address, clientid, turn
	// off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker(server)
	opts.SetClientID(fmt.Sprint("cb-net-agent-", n))
	opts.SetDefaultPublishHandler(f)

	// Create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		CBLogger.Panic(token.Error())
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
	CBLogger.Trace(string(doc))

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
	CBLogger.Info("Press the Enter Key to stop anytime")
	fmt.Scanln()

	//Unsubscribe from /cb-net/vm-network-information"
	if token := c.Unsubscribe("cb-net/networking-rule"); token.Wait() && token.Error() != nil {
		CBLogger.Error(token.Error())
		os.Exit(1)
	}

	// Disconnect MQTT Client
	c.Disconnect(250)
	CBLogger.Debug("End.........")
}
