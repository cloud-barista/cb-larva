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
	"github.com/go-ping/ping"
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

func pingTest(outVal *dataobjects.InterHostNetworkStatus, wg *sync.WaitGroup, trialCount int) {
	CBLogger.Debug("Start.........")
	defer wg.Done()

	pinger, err := ping.NewPinger(outVal.DestinationIP)
	pinger.SetPrivileged(true)
	if err != nil {
		panic(err)
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

	var CLADNetID = config.CBNetwork.CLADNetID
	var hostID = config.CBNetwork.HostID

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	keyHostNetworkInformation := fmt.Sprint(etcdkey.HostNetworkInformation + "/" + CLADNetID + "/" + hostID)
	keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + CLADNetID)
	keyStatusTestSpecification := fmt.Sprint(etcdkey.StatusTestSpecification + "/" + CLADNetID)
	keyStatusInformation := fmt.Sprint(etcdkey.StatusInformation + "/" + CLADNetID + "/" + hostID)

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

	if config.DemoApp.IsRun {
		// Start RunTunneling and blocked by channel until setting up the cb-network
		wg.Add(1)
		go app.PitcherAndCatcher(&wg, CBNet, channel)
	}

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		wg.Done()
		// Watch "/registry/cloud-adaptive-network/statistics/{cladnet-id}" with version
		CBLogger.Debugf("Start to watch \"%v\"", keyStatusTestSpecification)
		watchChan1 := etcdClient.Watch(context.Background(), keyStatusTestSpecification)
		for watchResponse := range watchChan1 {
			for _, event := range watchResponse.Events {
				CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

				// Get the trial count
				var testSpecification dataobjects.TestSpecification
				errUnmarshalEvalSpec := json.Unmarshal(event.Kv.Value, &testSpecification)
				if errUnmarshalEvalSpec != nil {
					CBLogger.Error(errUnmarshalEvalSpec)
				}
				trialCount := testSpecification.TrialCount

				// Evaluate a CLADNet (with networking rule and trial count)
				list := CBNet.NetworkingRules
				idx := list.GetIndexOfPublicIP(CBNet.MyPublicIP)
				//sourceIP := CBNet.NetworkingRules.HostIPAddress[idx]

				listLen := len(list.HostIPAddress)

				var wg sync.WaitGroup
				out := make([]dataobjects.InterHostNetworkStatus, listLen-1) // Skip a test between self and self
				// Compensation value (c) is used because all values in the list are used except for the host itself.
				var c = 0
				for i := 0; i < listLen; i++ {
					if i == idx {
						CBLogger.Debug("Skip the case that source and destination are same.")
						c = -1
						continue
					}
					wg.Add(1)
					j := i + c
					out[j].SourceID = list.HostID[idx]
					out[j].SourceIP = list.HostIPAddress[idx]
					out[j].DestinationID = list.HostID[i]
					out[j].DestinationIP = list.HostIPAddress[i]
					go pingTest(&out[j], &wg, trialCount)
				}
				wg.Wait()

				// Gather the evaluation results
				var statistics dataobjects.NetworkStatus
				for i := 0; i < len(out); i++ {
					statistics.InterHostNetworkStatus = append(statistics.InterHostNetworkStatus, out[i])
				}

				// Put the configuration information of the CLADNet to the etcd
				//keyConfigurationInformationOfCLADNet := fmt.Sprint(etcdkey.ConfigurationInformation + "/" + cladNetConfInfo.CLADNetID)
				strStatistics, _ := json.Marshal(statistics)
				_, err = etcdClient.Put(context.Background(), keyStatusInformation, string(strStatistics))
				if err != nil {
					CBLogger.Fatal(err)
				}

				// Put to etcd

			}
		}
		CBLogger.Debugf("End to watch \"%v\"", keyNetworkingRule)
	}(&wg)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		wg.Done()
		// Watch "/registry/cloud-adaptive-network/networking-rule/{cladnet-id}" with version
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
