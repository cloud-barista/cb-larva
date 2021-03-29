package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/app"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/data-objects"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/internal/etcd-key"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"os"
	"path/filepath"
	"strings"
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

func decodeAndSetNetworkingRule(key string, value []byte) {
	slicedKeys := strings.Split(key, "/")
	parsedHostID := slicedKeys[len(slicedKeys)-1]
	CBLogger.Tracef("ParsedHostID: %v\n", parsedHostID)

	var networkingRule dataobjects.NetworkingRule

	err := json.Unmarshal(value, &networkingRule)
	if err != nil {
		CBLogger.Panic(err)
	}

	prettyJSON, _ := json.MarshalIndent(networkingRule, "", "\t")
	CBLogger.Trace("Pretty JSON")
	CBLogger.Trace(string(prettyJSON))

	CBLogger.Info("Update the networking rule")
	CBNet.SetNetworkingRules(networkingRule)
	if !CBNet.IsRunning() {
		CBNet.StartCBNetworking(channel)
	}
}

func main() {
	CBLogger.Debug("Start.........")

	var arg string
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	var groupID = "group1"
	var hostID = "host1"

	keyHostNetworkInformation := fmt.Sprint(etcdkey.HostNetworkInformation + "/" + groupID + "/" + hostID)
	keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + groupID)

	channel = make(chan bool)

	// Create CBNetwork instance with port, which is tunneling port
	CBNet = cbnet.NewCBNetwork("cbnet0", 20000)

	// Get the VM network information
	temp := CBNet.GetHostNetworkInformation()
	currentHostNetworkInformationBytes, _ := json.Marshal(temp)
	currentHostNetworkInformation := string(currentHostNetworkInformationBytes)
	CBLogger.Trace(currentHostNetworkInformation)

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

	defer func() {
		errClose := etcdClient.Close()
		if errClose != nil {
			CBLogger.Fatal("Can't close the etcd client", errClose)
		}
	}()

	CBLogger.Infoln("The etcdClient is connected.")

	go func() {
		// Watch "/registry/cloud-adaptive-network/networking-rule/{group-id}"
		CBLogger.Infof("The etcdClient is watching \"%v\"\n", keyNetworkingRule)
		watchChan1 := etcdClient.Watch(context.Background(), keyNetworkingRule)
		for watchResponse := range watchChan1 {
			for _, event := range watchResponse.Events {
				CBLogger.Tracef("Watch - %s %q : %q\n", event.Type, event.Kv.Key, event.Kv.Value)
				decodeAndSetNetworkingRule(string(event.Kv.Key), event.Kv.Value)

			}
		}
	}()

	CBLogger.Info("Here!!!!")

	//requestTimeout := 10 * time.Second
	//ctx, _ := context.WithTimeout(context.Background(), requestTimeout)

	_, err = etcdClient.Put(context.Background(), keyHostNetworkInformation, currentHostNetworkInformation)
	if err != nil {
		CBLogger.Panic(err)
	}

	// Keep this
	//// Compare-and-Swap (CAS) host-network-information by groupId and hostId
	//// This should be running recursively or event-driven
	//txResp, err := etcdClient.Txn(ctx).
	//	If(clientv3.Compare(clientv3.Value(keyHostNetworkInformation), "!=", currentHostNetworkInformation)).
	//	Then(clientv3.OpPut(keyHostNetworkInformation, currentHostNetworkInformation)).
	//	Else(clientv3.OpGet(keyHostNetworkInformation)).
	//	Commit()
	//
	//if err != nil {
	//	CBLogger.Error(err)
	//}
	//
	////if txResp.Succeeded {
	////	return nil
	////}
	//
	//CBLogger.Tracef("txResp: %v\n", txResp)

	//respRule, respRuleErr := etcdClient.Get(context.Background(), keyNetworkingRule)
	//if respRuleErr != nil{
	//	CBLogger.Error(respRuleErr)
	//}
	//
	//if len(respRule.Kvs) != 0 {
	//	decodeAndSetNetworkingRule(string(respRule.Kvs[0].Key), respRule.Kvs[0].Value)
	//}

	go CBNet.RunTunneling(channel)

	if arg == "demo" {
		go app.PitcherAndCatcher(CBNet, channel)
	}

	// Block to stop this program
	CBLogger.Info("Press the Enter Key to stop anytime")
	fmt.Scanln()

	CBLogger.Debug("End.........")
}
