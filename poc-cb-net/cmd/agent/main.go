package main

import (
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net"
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func main() {
	// Create a ClientOptions struct setting the broker address, clientid, turn
	// off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://mqtt.eclipse.org:1883")
	opts.SetClientID("cb-net-agent")
	opts.SetDefaultPublishHandler(f)

	// Create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Get the network interface info
	var CBNetAgent = poc_cb_net.NewCBNetworkAgent()
	temp := CBNetAgent.GetNetworkInterface()
	doc, _ := json.Marshal(temp)
	fmt.Println(string(doc))

	// Publish a message to /cb-net/network-host-information at qos 1 and wait for the receipt
	// from the server after sending each message
	token := c.Publish("cb-net/network-host-information", 0, false, doc)
	token.Wait()

	c.Disconnect(250)
}
