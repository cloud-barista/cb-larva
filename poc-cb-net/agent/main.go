package main

import (
	"fmt"
	"time"

	//import the Paho Go MQTT library
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func main() {
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker("tcp://mqtt.eclipse.org:1883")
	opts.SetClientID("cb-net-agent")
	opts.SetDefaultPublishHandler(f)

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	////subscribe to the topic /go-mqtt/sample and request messages to be delivered
	////at a maximum qos of zero, wait for the receipt to confirm the subscription
	//if token := c.Subscribe("go-mqtt/sample", 0, nil); token.Wait() && token.Error() != nil {
	//	fmt.Println(token.Error())
	//	os.Exit(1)
	//}

	// Publish a message to /cb-net/network-host-information at qos 1 and wait for the receipt
	//from the server after sending each message
	dt := time.Now()
	text := fmt.Sprintf("[%s] This is msg AAA's network host setting information", dt.Format("01-02-2006 15:04:05"))
	token := c.Publish("cb-net/network-host-information", 0, false, text)
	token.Wait()

	//unsubscribe from /go-mqtt/sample
	//if token := c.Unsubscribe("go-mqtt/sample"); token.Wait() && token.Error() != nil {
	//	fmt.Println(token.Error())
	//	os.Exit(1)
	//}

	c.Disconnect(250)
}
