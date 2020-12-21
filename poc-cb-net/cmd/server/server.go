package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/data-objects"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/labstack/echo"
	"math/big"
	"os"
)

var dscp *internal.DynamicSubnetConfigurator

// Define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Recieved TOPIC : %s\n", msg.Topic())
	//fmt.Printf("MSG: %s\n", msg.Payload())

	if msg.Topic() == "cb-net/vm-network-information" {

		// Unmarshal the VM network information
		var vmNetworkInfo dataobjects.VMNetworkInformation

		err := json.Unmarshal([]byte(msg.Payload()), &vmNetworkInfo)
		if err != nil {
			panic(err)
		}
		fmt.Println("Unmarshalled JSON")
		fmt.Println(vmNetworkInfo)

		//prettyJSON, _ := json.MarshalIndent(vmNetworkInfo, "", "\t")
		//fmt.Println("Pretty JSON")
		//fmt.Println(string(prettyJSON))

		// Update CBNetworking Rule
		dscp.UpdateCBNetworkingRule(vmNetworkInfo)

		doc, _ := json.Marshal(dscp.NetworkingRule)

		client.Publish("cb-net/networking-rule", 0, false, doc)

	}
}

func RunEchoServer() {
	e := echo.New()

	e.Static("/", "assets")
	e.Static("/js", "assets/js")
	e.Static("/css", "assets/css")
	e.Static("/introspect", "assets/introspect")
	e.File("/", "public/index.html")

	e.Logger.Fatal(e.Start(":8000"))
}

func main() {

	// Random number to avoid MQTT client ID duplication
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		panic(err)
	}
	fmt.Println(n)

	// Create DynamicSubnetConfigurator instance
	dscp = internal.NewDynamicSubnetConfigurator()

	// Create a ClientOptions struct setting the broker address, clientID, turn
	// off trace output and set the default message handler

	config, _ := dataobjects.LoadConfigMQTTBroker()

	server := "tcp://" + config.MQTTBrokerIP + ":" + config.MQTTBrokerPort

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
	if token := c.Subscribe("cb-net/vm-network-information", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	go RunEchoServer()

	// Block to stop this program
	fmt.Println("Press the Enter Key to stop anytime")
	fmt.Scanln()

	//Unsubscribe from /cb-net/vm-network-information"
	if token := c.Unsubscribe("cb-net/vm-network-information"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Disconnect MQTT Client
	c.Disconnect(250)
}
