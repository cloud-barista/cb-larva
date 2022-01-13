package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cbnet "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network"
	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	cmd "github.com/cloud-barista/cb-larva/poc-cb-net/internal/command"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/internal/etcd-key"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/go-ping/ping"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// CBNet represents a network for the multi-cloud.
var CBNet *cbnet.CBNetwork
var channel chan bool
var mutex = &sync.Mutex{}

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config model.Config

func init() {
	fmt.Println("Start......... init() of agent.go")
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	fmt.Printf("exePath: %v\n", exePath)

	// Load cb-log config from the current directory (usually for the production)
	logConfPath := filepath.Join(exePath, "config", "log_conf.yaml")
	fmt.Printf("logConfPath: %v\n", logConfPath)
	if !file.Exists(logConfPath) {
		// Load cb-log config from the project directory (usually for development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		logConfPath = filepath.Join(projectPath, "poc-cb-net", "config", "log_conf.yaml")
	}
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)
	CBLogger.Debugf("Load %v", logConfPath)

	// Load cb-network config from the current directory (usually for the production)
	configPath := filepath.Join(exePath, "config", "config.yaml")
	fmt.Printf("configPath: %v\n", configPath)
	if !file.Exists(configPath) {
		// Load cb-network config from the project directory (usually for the development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		configPath = filepath.Join(projectPath, "poc-cb-net", "config", "config.yaml")
	}
	config, _ = model.LoadConfig(configPath)
	CBLogger.Debugf("Load %v", configPath)
	fmt.Println("End......... init() of agent.go")
}

// Control the cb-network agent by commands from remote
func watchControlCommand(etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")

	defer wg.Done()

	cladnetID := CBNet.ID
	hostID := CBNet.HostID

	// Watch "/registry/cloud-adaptive-network/control-command/{cladnet-id}/{host-id}
	keyControlCommand := fmt.Sprint(etcdkey.ControlCommand + "/" + cladnetID + "/" + hostID)
	CBLogger.Tracef("Watch \"%v\"", keyControlCommand)
	watchChan1 := etcdClient.Watch(context.TODO(), keyControlCommand)
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

			controlCommand, controlCommandOption := cmd.ParseCommandMessage(string(event.Kv.Value))

			handleCommand(controlCommand, controlCommandOption, etcdClient)
		}
	}
	CBLogger.Debug("Start.........")
}

// Handle commands of the cb-network agent
func handleCommand(command string, commandOption string, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	CBLogger.Debugf("Command: %+v", command)
	CBLogger.Tracef("CommandOption: %+v", commandOption)
	switch command {
	case "suspend":
		CBNet.Shutdown()

	case "resume":

		// Watch the networking rule to update dynamically
		go watchNetworkingRule(etcdClient)

		// Start the cb-network
		go CBNet.Startup()

		// Sleep until the all routines are ready
		time.Sleep(3 * time.Second)

		// Try Compare-And-Swap (CAS) host-network-information by cladnetID and hostId
		compareAndSwapHostNetworkInformation(etcdClient)

	case "check-connectivity":
		checkConnectivity(commandOption, etcdClient)

	default:
		CBLogger.Trace("Default ?")
	}

	CBLogger.Debug("End.........")
}

func watchNetworkingRule(etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	// Watch "/registry/cloud-adaptive-network/networking-rule/{cladnet-id}" with version
	keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + CBNet.ID)
	CBLogger.Tracef("Watch \"%v\"", keyNetworkingRule)
	watchChan1 := etcdClient.Watch(context.TODO(), keyNetworkingRule)
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			CBNet.DecodeAndSetNetworkingRule(event.Kv.Value)
		}
	}
	CBLogger.Debug("End.........")
}

func compareAndSwapHostNetworkInformation(etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	cladnetID := CBNet.ID
	hostID := CBNet.HostID

	CBLogger.Debug("Get the host network information")
	CBNet.UpdateHostNetworkInformation()
	temp := CBNet.GetHostNetworkInformation()
	currentHostNetworkInformationBytes, _ := json.Marshal(temp)
	currentHostNetworkInformation := string(currentHostNetworkInformationBytes)
	CBLogger.Trace(currentHostNetworkInformation)

	CBLogger.Debug("Compare-And-Swap (CAS) the host network information")
	keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + cladnetID)
	keyHostNetworkInformation := fmt.Sprint(etcdkey.HostNetworkInformation + "/" + cladnetID + "/" + hostID)
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
			slicedKeys := strings.Split(string(respKey), "/")
			parsedHostID := slicedKeys[len(slicedKeys)-1]
			CBLogger.Tracef("ParsedHostID: %v", parsedHostID)

			networkingRule := txnResp.Responses[0].GetResponseRange().Kvs[0].Value
			CBLogger.Tracef("The networking rule: %v", networkingRule)
			CBLogger.Debug("Set the networking rule")

			CBNet.DecodeAndSetNetworkingRule(networkingRule)
		}
	}

	CBLogger.Debug("End.........")
}

func checkConnectivity(data string, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	cladnetID := CBNet.ID
	hostID := CBNet.HostID

	// Get the trial count
	var testSpecification model.TestSpecification
	errUnmarshalEvalSpec := json.Unmarshal([]byte(data), &testSpecification)
	if errUnmarshalEvalSpec != nil {
		CBLogger.Error(errUnmarshalEvalSpec)
	}
	trialCount := testSpecification.TrialCount

	// Check status of a CLADNet
	list := CBNet.NetworkingRules
	idx := list.GetIndexOfPublicIP(CBNet.HostPublicIP)

	// Perform a ping test to the host behind this host (in other words, behind idx)
	listLen := len(list.HostIPAddress)
	outSize := listLen - idx - 1
	var testwg sync.WaitGroup
	out := make([]model.InterHostNetworkStatus, outSize)

	for i := 0; i < len(out); i++ {
		testwg.Add(1)
		j := idx + i + 1
		out[i].SourceID = list.HostID[idx]
		out[i].SourceIP = list.HostIPAddress[idx]
		out[i].DestinationID = list.HostID[j]
		out[i].DestinationIP = list.HostIPAddress[j]
		go pingTest(&out[i], &testwg, trialCount)
	}
	testwg.Wait()

	// Gather the evaluation results
	var networkStatus model.NetworkStatus
	for i := 0; i < len(out); i++ {
		networkStatus.InterHostNetworkStatus = append(networkStatus.InterHostNetworkStatus, out[i])
	}

	if networkStatus.InterHostNetworkStatus == nil {
		networkStatus.InterHostNetworkStatus = make([]model.InterHostNetworkStatus, 0)
	}

	// Put the network status of the CLADNet to the etcd
	// Key: /registry/cloud-adaptive-network/status/information/{cladnet-id}/{host-id}
	keyStatusInformation := fmt.Sprint(etcdkey.StatusInformation + "/" + cladnetID + "/" + hostID)
	strNetworkStatus, _ := json.Marshal(networkStatus)
	_, err := etcdClient.Put(context.Background(), keyStatusInformation, string(strNetworkStatus))
	if err != nil {
		CBLogger.Error(err)
	}

	CBLogger.Debug("End.........")
}

func pingTest(outVal *model.InterHostNetworkStatus, wg *sync.WaitGroup, trialCount int) {
	CBLogger.Debug("Start.........")
	defer wg.Done()

	pinger, err := ping.NewPinger(outVal.DestinationIP)
	pinger.SetPrivileged(true)
	if err != nil {
		CBLogger.Error(err)
	}
	//pinger.OnRecv = func(pkt *ping.Packet) {
	//	fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n",
	//		pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
	//}

	pinger.Count = trialCount
	pinger.Run() // blocks until finished

	stats := pinger.Statistics() // get send/receive/rtt stats
	outVal.MininumRTT = stats.MinRtt.Seconds()
	outVal.AverageRTT = stats.AvgRtt.Seconds()
	outVal.MaximumRTT = stats.MaxRtt.Seconds()
	outVal.StdDevRTT = stats.StdDevRtt.Seconds()
	outVal.PacketsReceive = stats.PacketsRecv
	outVal.PacketsLoss = stats.PacketsSent - stats.PacketsRecv
	outVal.BytesReceived = stats.PacketsRecv * 24

	CBLogger.Tracef("round-trip min/avg/max/stddev/dupl_recv = %v/%v/%v/%v/%v bytes",
		stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt, stats.PacketsRecv*24)
	CBLogger.Debug("End.........")
}

func main() {
	CBLogger.Debug("Start.........")

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

	// Cloud Adaptive Network section
	// Config
	var cladnetID = config.CBNetwork.CLADNetID
	var hostID string
	if config.CBNetwork.HostID == "" {
		name, err := os.Hostname()
		if err != nil {
			CBLogger.Error(err)
		}
		hostID = name
	} else {
		hostID = config.CBNetwork.HostID
	}

	// Create CBNetwork instance with port, which is a tunneling port
	CBNet = cbnet.New("cbnet0", 20000)
	CBNet.ID = cladnetID
	CBNet.HostID = hostID

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	wg.Add(1)
	go watchControlCommand(etcdClient, &wg)

	// Sleep until the all routines are ready
	time.Sleep(2 * time.Second)

	// Resume cb-network
	handleCommand("resume", "", etcdClient)

	// Waiting for all goroutines to finish
	CBLogger.Info("Waiting for all goroutines to finish")
	wg.Wait()

	CBLogger.Debug("End cb-network agent .........")
}
