package main

import (
	"encoding/json"
	"fmt"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/data-objects"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"os"
)

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("Recieved TOPIC : %s\n", msg.Topic())
	//fmt.Printf("MSG: %s\n", msg.Payload())

	if msg.Topic() == "cb-net/network-host-information" {

		// Unmarshal the network interfaces
		var temp2 []dataobjects.NetworkInterface

		_ = json.Unmarshal([]byte(msg.Payload()), &temp2)
		fmt.Println("Unmarshalled JSON")
		fmt.Println(temp2)

		prettyJSON, _ := json.MarshalIndent(temp2, "", "\t")
		fmt.Println("Pretty JSON")
		fmt.Println(string(prettyJSON))
	}
}

func main() {
	// Create a ClientOptions struct setting the broker address, clientid, turn
	// off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://mqtt.eclipse.org:1883")
	opts.SetClientID("cb-net-server")
	opts.SetDefaultPublishHandler(f)

	// Create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe to the topic /go-mqtt/sample and request messages to be delivered
	// at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe("cb-net/network-host-information", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	fmt.Println("Press the Enter Key to stop anytime")
	fmt.Scanln()

	//unsubscribe from /go-mqtt/sample
	if token := c.Unsubscribe("cb-net/network-host-information"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	c.Disconnect(250)
}
