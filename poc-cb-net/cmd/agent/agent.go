package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/app"
	cbnet "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/data-objects"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/internal/etcd-key"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// CBNet represents a network for the multi-cloud.
var CBNet *cbnet.CBNetwork
var channel chan bool
var mutex = &sync.Mutex{}

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config dataobjects.Config

func init() {
	fmt.Println("Start......... init() of agent.go")
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	fmt.Printf("exePath: %v\n", exePath)

	// Load cb-log config from the current directory (usually for the production)
	logConfPath := filepath.Join(exePath, "configs", "log_conf.yaml")
	fmt.Printf("logConfPath: %v\n", logConfPath)
	if !file.Exists(logConfPath) {
		// Load cb-log config from the project directory (usually for development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		logConfPath = filepath.Join(projectPath, "poc-cb-net", "configs", "log_conf.yaml")
	}
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)
	CBLogger.Debugf("Load %v", logConfPath)

	// Load cb-network config from the current directory (usually for the production)
	configPath := filepath.Join(exePath, "configs", "config.yaml")
	fmt.Printf("configPath: %v\n", configPath)
	if !file.Exists(configPath) {
		// Load cb-network config from the project directory (usually for the development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		configPath = filepath.Join(projectPath, "poc-cb-net", "configs", "config.yaml")
	}
	config, _ = dataobjects.LoadConfig(configPath)
	CBLogger.Debugf("Load %v", configPath)
	fmt.Println("End......... init() of agent.go")
}

func decodeAndSetNetworkingRule(key string, value []byte, hostID string) {
	mutex.Lock()
	CBLogger.Debug("Start.........")
	slicedKeys := strings.Split(key, "/")
	parsedHostID := slicedKeys[len(slicedKeys)-1]
	CBLogger.Tracef("ParsedHostID: %v", parsedHostID)

	var networkingRule dataobjects.NetworkingRule

	err := json.Unmarshal(value, &networkingRule)
	if err != nil {
		CBLogger.Panic(err)
	}

	prettyJSON, _ := json.MarshalIndent(networkingRule, "", "\t")
	CBLogger.Trace("Pretty JSON")
	CBLogger.Trace(string(prettyJSON))

	if networkingRule.Contain(hostID) {
		CBNet.SetNetworkingRules(networkingRule)
		if !CBNet.IsRunning() {
			_, err := CBNet.StartCBNetworking(channel)
			if err != nil {
				CBLogger.Error(err)
			}
		}
	}
	CBLogger.Debug("End.........")
	mutex.Unlock()
}

func main() {
	CBLogger.Debug("Start.........")

	var arg string
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}

	var CLADNetID = "c2eau8atiahtscepc2dg"
	var hostID = "host1"

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	keyHostNetworkInformation := fmt.Sprint(etcdkey.HostNetworkInformation + "/" + CLADNetID + "/" + hostID)
	keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + CLADNetID)

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

	channel = make(chan bool)

	// Create CBNetwork instance with port, which is tunneling port
	CBNet = cbnet.NewCBNetwork("cbnet0", 20000)

	// Start RunTunneling and blocked by channel until setting up the cb-network
	wg.Add(1)
	go CBNet.RunTunneling(&wg, channel)

	if arg == "demo" {
		// Start RunTunneling and blocked by channel until setting up the cb-network
		wg.Add(1)
		go app.PitcherAndCatcher(&wg, CBNet, channel)
	}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		wg.Done()
		// Watch "/registry/cloud-adaptive-network/networking-rule/{group-id}" with version
		CBLogger.Debugf("Start to watch \"%v\"", keyNetworkingRule)
		watchChan1 := etcdClient.Watch(context.Background(), keyNetworkingRule)
		for watchResponse := range watchChan1 {
			for _, event := range watchResponse.Events {
				CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
				decodeAndSetNetworkingRule(string(event.Kv.Key), event.Kv.Value, hostID)
			}
		}
		CBLogger.Debugf("End to watch \"%v\"", keyNetworkingRule)
	}(&wg)

	// Try Compare-And-Swap (CAS) host-network-information by CLADNetID and hostId
	CBLogger.Debug("Get the host network information")
	temp := CBNet.GetHostNetworkInformation()
	currentHostNetworkInformationBytes, _ := json.Marshal(temp)
	currentHostNetworkInformation := string(currentHostNetworkInformationBytes)
	CBLogger.Trace(currentHostNetworkInformation)

	CBLogger.Debug("Compare-And-Swap (CAS) the host network information")
	// NOTICE: "!=" doesn't work..... It might be a temporal issue.
	txnResp, err := etcdClient.Txn(context.Background()).
		If(clientv3.Compare(clientv3.Value(keyHostNetworkInformation), "=", currentHostNetworkInformation)).
		Then(clientv3.OpGet(keyNetworkingRule)).
		Else(clientv3.OpPut(keyHostNetworkInformation, currentHostNetworkInformation)).
		Commit()

	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Transaction Response: %v", txnResp)

	// The CAS would be succeeded if the prev host network information and current host network information are same.
	// Then the networking rule will be returned. (The above "watch" will not be performed.)
	// If not, the host tries to put the current host network information.
	if txnResp.Succeeded {
		// Set the networking rule to the host
		if len(txnResp.Responses[0].GetResponseRange().Kvs) != 0 {
			respKey := txnResp.Responses[0].GetResponseRange().Kvs[0].Key
			respValue := txnResp.Responses[0].GetResponseRange().Kvs[0].Value
			CBLogger.Tracef("The networking rule: %v", respValue)
			CBLogger.Debug("Set the networking rule")
			decodeAndSetNetworkingRule(string(respKey), respValue, hostID)
		}
	}

	// Waiting for all goroutines to finish
	CBLogger.Info("Waiting for all goroutines to finish")
	wg.Wait()

	CBLogger.Debug("End cb-network agent .........")
}
