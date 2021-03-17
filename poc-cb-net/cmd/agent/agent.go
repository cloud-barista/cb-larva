package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/app"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/data-objects"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"os"
	"path/filepath"
	"time"
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

//// MyMQTTMessageHandler is a default message handler.
//func MyMQTTMessageHandler(client MQTT.Client, msg MQTT.Message) {
//	CBLogger.Debug("Start.........")
//	CBLogger.Debugf("Received TOPIC : %s\n", msg.Topic())
//	CBLogger.Debugf("MSG: %s\n", msg.Payload())
//
//	if msg.Topic() == "cb-net/networking-rule" {
//		var networkingRule dataobjects.NetworkingRule
//
//		err := json.Unmarshal(msg.Payload(), &networkingRule)
//		if err != nil {
//			CBLogger.Panic(err)
//		}
//		CBLogger.Trace("Unmarshalled JSON")
//		CBLogger.Trace(networkingRule)
//
//		prettyJSON, _ := json.MarshalIndent(networkingRule, "", "\t")
//		CBLogger.Trace("Pretty JSON")
//		CBLogger.Trace(string(prettyJSON))
//
//		CBLogger.Info("Update the networking rule")
//		CLADNetCIDRBlock.SetNetworkingRules(networkingRule)
//		if !CLADNetCIDRBlock.IsRunning() {
//			CLADNetCIDRBlock.StartCBNetworking(channel)
//		}
//	}
//	CBLogger.Debug("End.........")
//}
//
//var f MQTT.MessageHandler = MyMQTTMessageHandler

func main() {
	CBLogger.Debug("Start.........")

	var arg string
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	var groupID = "group1"
	var hostID = "host1"

	channel = make(chan bool)

	// Create CBNetwork instance with port, which is tunneling port
	CBNet = cbnet.NewCBNetwork("cbnet0", 20000)

	// Get the VM network information
	temp := CBNet.GetHostNetworkInformation()
	hostNetworkInformationBytes, _ := json.Marshal(temp)
	hostNetworkInformation := string(hostNetworkInformationBytes)
	CBLogger.Trace(hostNetworkInformation)

	// Load config
	configPath := filepath.Join("..", "..", "configs", "config.yaml")
	config, _ := dataobjects.LoadConfig(configPath)

	// etcd Section
	// Connect to the etcd cluster
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   config.ETCD.Endpoints,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		CBLogger.Fatal(err)
	}
	defer etcdClient.Close()

	CBLogger.Infoln("The etcdClient is connected.")

	requestTimeout := 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	hostKey := fmt.Sprint("/cladnet/" + groupID + "/host_network_information/" + hostID)

	//// Get host_network-information by host_id
	//var net1 dataobjects.HostNetworkInformation
	//resp, err := etcdClient.Get(ctx, hostKey)
	//cancel()
	//if err != nil {
	//	CBLogger.Fatal(err)
	//}
	//
	//for _, ev := range resp.Kvs {
	//	CBLogger.Tracef("%s : %s\n", ev.Key, ev.Value)
	//	err := json.Unmarshal(ev.Value, &net1)
	//		if err != nil {
	//			CBLogger.Panic(err)
	//		}
	//		CBLogger.Trace("Unmarshalled JSON")
	//		CBLogger.Trace(net1)
	//}

	// Put host_network-information by host_id
	// Compare-and-Swap (CAS)
	txResp, err := etcdClient.Txn(ctx).
		If(clientv3.Compare(clientv3.Value(hostKey), "=", hostNetworkInformation)).
		Then(clientv3.OpGet(hostKey)).
		Else(clientv3.OpPut(hostKey, hostNetworkInformation)).
		Commit()
	cancel()

	if err != nil {
		CBLogger.Error(err)
	}

	//if txResp.Succeeded {
	//	return nil
	//}

	CBLogger.Tracef("txResp: %v\n", txResp)
	//CBLogger.Trace(string(txResp.Responses[0].GetResponseRange().Kvs[0].Value))
	//return errors.New("release error")

	//_, err = etcdClient.Put(ctx, hostKey, string(hostNetworkInformationBytes))
	//cancel()
	//if err != nil {
	//	switch err {
	//	case context.Canceled:
	//		CBLogger.Errorf("ctx is canceled by another routine: %v\n", err)
	//	case context.DeadlineExceeded:
	//		CBLogger.Errorf("ctx is attached with a deadline is exceeded: %v\n", err)
	//	case rpctypes.ErrEmptyKey:
	//		CBLogger.Errorf("client-side error: %v\n", err)
	//	default:
	//		CBLogger.Errorf("bad cluster endpoints, which are not etcd servers: %v\n", err)
	//		// use the response
	//	}
	//}

	//// Create a endpoint link of MQTTBroker
	//server := "tcp://" + config.MQTTBroker.Host + ":" + config.MQTTBroker.Port
	//
	//// Create a ClientOptions struct setting the broker address, clientid, turn
	//// off trace output and set the default message handler
	//opts := MQTT.NewClientOptions().AddBroker(server)
	//opts.SetClientID(fmt.Sprint("cb-net-agent-", id))
	//opts.SetDefaultPublishHandler(f)
	//
	//// Create and start a client using the above ClientOptions
	//c := MQTT.NewClient(opts)
	//if token := c.Connect(); token.Wait() && token.Error() != nil {
	//	CBLogger.Panic(token.Error())
	//}
	//
	//// Subscribe to the topic /go-mqtt/sample and request messages to be delivered
	//// at a maximum qos of zero, wait for the receipt to confirm the subscription
	//if token := c.Subscribe("cb-net/networking-rule", 0, nil); token.Wait() && token.Error() != nil {
	//	CBLogger.Error(token.Error())
	//	os.Exit(1)
	//}

	//// Publish a message to /cb-net/vm-network-information at qos 1 and wait for the receipt
	//// from the server after sending each message
	//token := c.Publish("cb-net/vm-network-information", 0, false, hostNetworkInformationBytes)
	//token.Wait()

	//go CLADNetCIDRBlock.RunEncapsulation(channel)
	//go CLADNetCIDRBlock.RunDecapsulation(channel)
	go CBNet.RunTunneling(channel)
	if arg == "demo" {
		go app.PitcherAndCatcher(CBNet, channel)
	}

	// Block to stop this program
	CBLogger.Info("Press the Enter Key to stop anytime")
	fmt.Scanln()

	////Unsubscribe from /cb-net/vm-network-information"
	//if token := c.Unsubscribe("cb-net/networking-rule"); token.Wait() && token.Error() != nil {
	//	CBLogger.Error(token.Error())
	//	os.Exit(1)
	//}
	//
	//// Disconnect MQTT Client
	//c.Disconnect(250)
	CBLogger.Debug("End.........")
}
