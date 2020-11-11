package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/data-objects"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"math/big"
	"os"
)

var CBNet *poc_cb_net.CBNetwork
var channel chan bool

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())

	if msg.Topic() == "cb-net/networking-rule" {

		var networkingRule dataobjects.NetworkingRule

		_ = json.Unmarshal([]byte(msg.Payload()), &networkingRule)
		fmt.Println("Unmarshalled JSON")
		fmt.Println(networkingRule)

		//prettyJSON, _ := json.MarshalIndent(networkingRule, "", "\t")
		//fmt.Println("Pretty JSON")
		//fmt.Println(string(prettyJSON))

		CBNet.SetNetworkingRule(networkingRule)
		fmt.Println("1")
		if !CBNet.IsRunning() {
			CBNet.StartCBNetworking(channel)
		}
		fmt.Println("3")
	}
}

func main() {

	channel = make(chan bool)

	// Random number to avoid MQTT client ID duplication
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		panic(err)
	}
	fmt.Println(n)

	// Create CBNetwork instance with port, which is tunneling port
	CBNet = poc_cb_net.NewCBNetwork("cbnet0", 20000)

	// Create a ClientOptions struct setting the broker address, clientid, turn
	// off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://mqtt.eclipse.org:1883")
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
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Get the VM network information
	temp := CBNet.GetVMNetworkInformation()
	doc, _ := json.Marshal(temp)
	fmt.Println(string(doc))

	// Publish a message to /cb-net/vm-network-information at qos 1 and wait for the receipt
	// from the server after sending each message
	token := c.Publish("cb-net/vm-network-information", 0, false, doc)
	token.Wait()

	go CBNet.RunEncapsulation(channel)
	go CBNet.RunDecapsulation(channel)
	go internal.PitcherAndCatcher(CBNet, channel)

	// Block to stop this program
	fmt.Println("Press the Enter Key to stop anytime")
	fmt.Scanln()

	//Unsubscribe from /cb-net/vm-network-information"
	if token := c.Unsubscribe("cb-net/networking-rule"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Disconnect MQTT Client
	c.Disconnect(250)
}
