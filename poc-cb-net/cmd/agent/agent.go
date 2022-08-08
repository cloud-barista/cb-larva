package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	cbnet "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network"
	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
	cmdtype "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/command-type"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/etcd-key"
	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	netstate "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/network-state"
	testtype "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/test-type"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/go-ping/ping"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// CBNet represents a network for the multi-cloud.
var CBNet *cbnet.CBNetwork

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config model.Config

func init() {
	fmt.Println("\nStart......... init() of agent.go")

	// Set cb-log
	env := os.Getenv("CBLOG_ROOT")
	if env != "" {
		// Load cb-log config from the environment variable path (default)
		fmt.Printf("CBLOG_ROOT: %v\n", env)
		CBLogger = cblog.GetLogger("cb-network")
	} else {

		// Load cb-log config from the current directory (usually for the production)
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exePath := filepath.Dir(ex)
		// fmt.Printf("exe path: %v\n", exePath)

		logConfPath := filepath.Join(exePath, "config", "log_conf.yaml")
		if file.Exists(logConfPath) {
			fmt.Printf("path of log_conf.yaml: %v\n", logConfPath)
			CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)

		} else {
			// Load cb-log config from the project directory (usually for development)
			logConfPath = filepath.Join(exePath, "..", "..", "config", "log_conf.yaml")
			if file.Exists(logConfPath) {
				fmt.Printf("path of log_conf.yaml: %v\n", logConfPath)
				CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)
			} else {
				err := errors.New("fail to load log_conf.yaml")
				panic(err)
			}
		}
		CBLogger.Debugf("Load %v", logConfPath)

	}

	// Load cb-network config from the current directory (usually for the production)
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	// fmt.Printf("exe path: %v\n", exePath)

	configPath := filepath.Join(exePath, "config", "config.yaml")
	if file.Exists(configPath) {
		fmt.Printf("path of config.yaml: %v\n", configPath)
		config, _ = model.LoadConfig(configPath)
	} else {
		// Load cb-network config from the project directory (usually for the development)
		configPath = filepath.Join(exePath, "..", "..", "config", "config.yaml")

		if file.Exists(configPath) {
			config, _ = model.LoadConfig(configPath)
		} else {
			err := errors.New("fail to load config.yaml")
			panic(err)
		}
	}

	CBLogger.Debugf("Load %v", configPath)

	fmt.Println("End......... init() of agent.go")
	fmt.Println("")
}

// Control the cb-network agent by commands from remote
func watchControlCommand(ctx context.Context, etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")

	defer wg.Done()

	cladnetID := CBNet.CLADNetID
	hostID := CBNet.HostID

	// Watch "/registry/cloud-adaptive-network/control-command/{cladnet-id}/{host-id}
	keyControlCommand := fmt.Sprint(etcdkey.ControlCommand + "/" + cladnetID + "/" + hostID)
	CBLogger.Tracef("Watch \"%v\"", keyControlCommand)
	watchChan1 := etcdClient.Watch(ctx, keyControlCommand)
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

			controlCommand := cmdtype.ParseCommandMessage(string(event.Kv.Value))

			handleCommand(controlCommand, etcdClient)
		}
	}
	CBLogger.Debug("End.........")
}

// Handle commands of the cb-network agent
func handleCommand(controlCommand string, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	CBLogger.Debugf("Command: %#v", controlCommand)
	switch controlCommand {
	case cmdtype.Down:
		CBLogger.Debug("close the cb-network interface in 'tunneling' state")

		state := CBNet.State()
		CBLogger.Debugf("current state: %+v", state)
		if state == netstate.Tunneling {
			updatePeerState(netstate.Closing, etcdClient)
			CBNet.CloseCBNetworkInterface()
			updatePeerState(netstate.Released, etcdClient)
		}

	case cmdtype.Up:
		CBLogger.Debug("configure the cb-network interface in 'released' or '' state")

		state := CBNet.State()
		CBLogger.Debugf("current state: %+v", state)
		if state == "" || state == netstate.Released {
			// Run the cb-network
			go CBNet.Run()
			// Wait until the goroutine is started
			time.Sleep(200 * time.Millisecond)

			// Try Compare-And-Swap (CAS) an agent's secret (RSA public keys)
			if CBNet.IsEncryptionEnabled() {
				initializeSecret(etcdClient)
			}

			// Try Compare-And-Swap (CAS) a host-network-information by cladnetID and hostId
			initializeAgent(etcdClient)
		}

	case cmdtype.EnableEncryption:
		CBLogger.Debug("enable end-to-end encryption")

		CBNet.EnableEncryption(true)
		if CBNet.IsEncryptionEnabled() {
			initializeSecret(etcdClient)
		}

	case cmdtype.DisableEncryption:
		CBLogger.Debug("disable end-to-end encryption")

		CBNet.DisableEncryption()

	default:
		CBLogger.Errorf("unknown control-command => %v\n", controlCommand)
	}

	CBLogger.Debug("End.........")
}

// Watch test request for a Cloud Adaptive Network
func watchTestRequest(ctx context.Context, etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")

	defer wg.Done()

	cladnetID := CBNet.CLADNetID
	hostID := CBNet.HostID

	// Watch "/registry/cloud-adaptive-network/test-request/{cladnet-id}/{host-id}
	keyTestRequest := fmt.Sprint(etcdkey.TestRequest + "/" + cladnetID + "/" + hostID)
	CBLogger.Tracef("Watch \"%v\"", keyTestRequest)
	watchChan1 := etcdClient.Watch(ctx, keyTestRequest)
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

			testType, testSpec := testtype.ParseTestMessage(string(event.Kv.Value))

			handleTest(testType, testSpec, etcdClient)
		}
	}
	CBLogger.Debug("End.........")
}

// Handle testing a Cloud Adaptive Network
func handleTest(testType string, testSpec string, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	CBLogger.Debugf("TestType: %#v", testType)
	CBLogger.Debugf("TestSpec: %#v", testSpec)

	switch testType {
	case testtype.Connectivity:
		CBLogger.Debug("check connectivity in 'tunneling' state")

		state := CBNet.State()
		CBLogger.Debugf("current state: %+v", state)
		if state == netstate.Tunneling {
			checkConnectivity(testSpec, etcdClient)
		}

	default:
		CBLogger.Errorf("unknown test-request => %v\n", testType)
	}

	CBLogger.Debug("End.........")
}

func checkConnectivity(data string, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	cladnetID := CBNet.CLADNetID
	hostID := CBNet.HostID

	// Get the trial count
	var testSpecification model.TestSpecification
	errUnmarshalEvalSpec := json.Unmarshal([]byte(data), &testSpecification)
	if errUnmarshalEvalSpec != nil {
		CBLogger.Error(errUnmarshalEvalSpec)
	}
	trialCount := testSpecification.TrialCount

	// Check status of a CLADNet
	networkingRule := CBNet.NetworkingRule
	// idx := networkingRule.GetIndexOfPublicIP(CBNet.HostPublicIP)
	sourceName := CBNet.ThisPeer.HostName
	sourceIP := CBNet.ThisPeer.IP

	// Perform ping test from this host to another host
	listLen := len(networkingRule.PeerIP)
	outSize := listLen + 1 // to include this peer
	var testwg sync.WaitGroup
	out := make([]model.InterHostNetworkStatus, outSize)

	// Test with the other peers
	for i := 0; i < listLen; i++ {
		out[i].SourceName = sourceName
		out[i].SourceIP = sourceIP
		out[i].DestinationName = networkingRule.HostName[i]
		out[i].DestinationIP = networkingRule.PeerIP[i]

		testwg.Add(1)
		go pingTest(&out[i], &testwg, trialCount)
	}

	// Self test
	out[listLen].SourceName = sourceName
	out[listLen].SourceIP = sourceIP
	out[listLen].DestinationName = sourceName
	out[listLen].DestinationIP = sourceIP
	testwg.Add(1)
	go pingTest(&out[listLen], &testwg, trialCount)

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

	CBLogger.Tracef("Ping to %s", outVal.DestinationIP)

	pinger, err := ping.NewPinger(outVal.DestinationIP)
	pinger.SetPrivileged(true)
	if err != nil {
		CBLogger.Error(err)
	}
	//pinger.OnRecv = func(pkt *ping.Packet) {
	//	fmt.Printf("%d bytes from %s: icmp_seq=%d time=%v\n",
	//		pkt.Nbytes, pkt.IPAddr, pkt.Seq, pkt.Rtt)
	//}

	size := 64 // default 64 bytes
	pinger.Size = size
	pinger.Count = trialCount
	pinger.Run() // blocks until finished

	stats := pinger.Statistics() // get send/receive/rtt stats
	outVal.MininumRTT = stats.MinRtt.Seconds()
	outVal.AverageRTT = stats.AvgRtt.Seconds()
	outVal.MaximumRTT = stats.MaxRtt.Seconds()
	outVal.StdDevRTT = stats.StdDevRtt.Seconds()
	outVal.PacketsReceive = stats.PacketsRecv
	outVal.PacketsLoss = stats.PacketsSent - stats.PacketsRecv
	outVal.BytesReceived = stats.PacketsRecv * size

	CBLogger.Tracef("round-trip min/avg/max/stddev/dupl_recv = %v/%v/%v/%v/%v bytes",
		stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt, stats.PacketsRecv*size)
	CBLogger.Debug("End.........")
}

func watchThisPeer(ctx context.Context, etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")

	defer wg.Done()

	// Watch "/registry/cloud-adaptive-network/peer/{cladnet-id}/{host-id}" with version
	keyThisPeer := fmt.Sprint(etcdkey.Peer + "/" + CBNet.CLADNetID + "/" + CBNet.HostID)
	CBLogger.Tracef("Watch \"%v\"", keyThisPeer)
	watchChan1 := etcdClient.Watch(ctx, keyThisPeer, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

			key := string(event.Kv.Key)

			var peer model.Peer
			err := json.Unmarshal(event.Kv.Value, &peer)
			if err != nil {
				CBLogger.Error(err)
			}

			CBNet.ThisPeer = peer

			if peer.State == netstate.Configuring {
				err := CBNet.ConfigureCBNetworkInterface()
				if err != nil {
					CBLogger.Error(err)

				} else {
					peer.State = netstate.Tunneling
					doc, _ := json.Marshal(peer)

					CBLogger.Debugf("Put - \"%v\"", key)
					if _, err := etcdClient.Put(context.TODO(), key, string(doc)); err != nil {
						CBLogger.Error(err)
					}
				}
			}
		}
	}
	CBLogger.Debug("End.........")
}

func watchNetworkingRule(ctx context.Context, etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")

	defer wg.Done()

	// Watch "/registry/cloud-adaptive-network/networking-rule/{cladnet-id}/{host-id}"
	keyNetworkingRuleOfPeer := fmt.Sprint(etcdkey.NetworkingRule + "/" + CBNet.CLADNetID + "/" + CBNet.HostID)
	CBLogger.Tracef("Watch \"%v\"", keyNetworkingRuleOfPeer)
	watchChan1 := etcdClient.Watch(ctx, keyNetworkingRuleOfPeer)
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

			// key := string(event.Kv.Key)

			var networkingRule model.NetworkingRule
			err := json.Unmarshal(event.Kv.Value, &networkingRule)
			if err != nil {
				CBLogger.Error(err)
			}

			// Update a host's configuration in the networking rule
			CBNet.UpdateNetworkingRule(networkingRule)

			// if peer.HostID == CBNet.HostID && peer.State == netstate.Configuring {
			// 	err := CBNet.ConfigureCBNetworkInterface()
			// 	if err != nil {
			// 		CBLogger.Error(err)

			// 	} else {
			// 		peer.State = netstate.Tunneling
			// 		doc, _ := json.Marshal(peer)

			// 		CBLogger.Debugf("Put - \"%v\"", key)
			// 		if _, err := etcdClient.Put(context.TODO(), key, string(doc)); err != nil {
			// 			CBLogger.Error(err)
			// 		}
			// 	}
			// }
		}
	}
	CBLogger.Debug("End.........")
}

func initializeAgent(etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	cladnetID := CBNet.CLADNetID
	hostID := CBNet.HostID

	// // Create a sessions to acquire a lock
	// session, _ := concurrency.NewSession(etcdClient)
	// defer session.Close()

	// keyPrefix := fmt.Sprint(etcdkey.LockNetworkingRule + "/" + cladnetID)

	// lock := concurrency.NewMutex(session, keyPrefix)
	// ctx := context.TODO()

	// // Acquire lock (or wait to have it)
	// CBLogger.Debug("Acquire a lock")
	// if err := lock.Lock(ctx); err != nil {
	// 	CBLogger.Error(err)
	// }
	// CBLogger.Tracef("Acquired lock for '%s'", keyPrefix)

	// // Get the networking rule
	// keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + cladnetID)
	// CBLogger.Debugf("Get - %v", keyNetworkingRule)

	// resp, etcdErr := etcdClient.Get(context.Background(), keyNetworkingRule, clientv3.WithPrefix())

	// if etcdErr != nil {
	// 	CBLogger.Error(etcdErr)
	// }
	// CBLogger.Tracef("GetResponse: %v", resp)

	// // Set the other hosts' networking rule
	// for _, kv := range resp.Kvs {
	// 	// Key
	// 	key := string(kv.Key)
	// 	CBLogger.Tracef("Key: %v", key)

	// 	// Value
	// 	peerBytes := kv.Value
	// 	var peer model.Peer
	// 	if err := json.Unmarshal(peerBytes, &peer); err != nil {
	// 		CBLogger.Error(err)
	// 	}
	// 	CBLogger.Tracef("A host's configuration: %v", peer)

	// 	if peer.HostID != hostID {
	// 		// Update a host's configuration in the networking rule
	// 		CBLogger.Debug("Update a host's configuration")
	// 		CBNet.UpdateNetworkingRule(peer)
	// 	}
	// }

	// Get this host's network information
	CBLogger.Debug("Get the host network information")
	CBNet.UpdateHostNetworkInformation()
	temp := CBNet.GetHostNetworkInformation()
	currentHostNetworkInformationBytes, _ := json.Marshal(temp)
	currentHostNetworkInformation := string(currentHostNetworkInformationBytes)
	CBLogger.Trace(currentHostNetworkInformation)

	keyHostNetworkInformation := fmt.Sprint(etcdkey.HostNetworkInformation + "/" + cladnetID + "/" + hostID)

	if _, err := etcdClient.Put(context.TODO(), keyHostNetworkInformation, currentHostNetworkInformation); err != nil {
		CBLogger.Error(err)
	}

	// // Release lock
	// CBLogger.Debug("Release a lock")
	// if err := lock.Unlock(ctx); err != nil {
	// 	CBLogger.Error(err)
	// }
	// CBLogger.Tracef("Released lock for '%s'", keyPrefix)

	CBLogger.Debug("End.........")
}

func updatePeerState(state string, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	CBNet.ThisPeer.State = state

	keyPeer := fmt.Sprint(etcdkey.Peer + "/" + CBNet.CLADNetID + "/" + CBNet.HostID)

	CBLogger.Debugf("Put \"%v\"", keyPeer)
	doc, _ := json.Marshal(CBNet.ThisPeer)

	if _, err := etcdClient.Put(context.TODO(), keyPeer, string(doc)); err != nil {
		CBLogger.Error(err)
	}

	CBLogger.Debug("End.........")
}

func watchSecret(ctx context.Context, etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")

	defer wg.Done()

	// Watch "/registry/cloud-adaptive-network/secret/{cladnet-id}"
	keySecretGroup := fmt.Sprint(etcdkey.Secret + "/" + CBNet.CLADNetID)
	CBLogger.Tracef("Watch \"%v\"", keySecretGroup)
	watchChan1 := etcdClient.Watch(ctx, keySecretGroup, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			slicedKeys := strings.Split(string(event.Kv.Key), "/")
			parsedHostID := slicedKeys[len(slicedKeys)-1]
			CBLogger.Tracef("ParsedHostID: %v", parsedHostID)

			// Update keyring (including add)
			CBNet.UpdateKeyring(parsedHostID, string(event.Kv.Value))
		}
	}
	CBLogger.Debug("End.........")
}

func initializeSecret(etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	cladnetID := CBNet.CLADNetID
	hostID := CBNet.HostID

	// Create a sessions to acquire a lock
	session, _ := concurrency.NewSession(etcdClient)
	defer session.Close()

	keyPrefix := fmt.Sprint(etcdkey.LockSecret + "/" + cladnetID)

	lock := concurrency.NewMutex(session, keyPrefix)
	ctx := context.TODO()

	// Acquire lock (or wait to have it)
	CBLogger.Debug("Acquire a lock")
	if err := lock.Lock(ctx); err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Acquired lock for '%s'", keyPrefix)

	// Get the secret
	keySecret := fmt.Sprint(etcdkey.Secret + "/" + cladnetID)
	CBLogger.Debugf("Get - %v", keySecret)
	resp, etcdErr := etcdClient.Get(context.TODO(), keySecret, clientv3.WithPrefix())
	CBLogger.Tracef("GetResponse: %v", resp)
	if etcdErr != nil {
		CBLogger.Error(etcdErr)
	}

	// Set the other hosts' secrets
	for _, kv := range resp.Kvs {
		// Key
		key := string(kv.Key)
		CBLogger.Tracef("Key: %v", key)

		slicedKeys := strings.Split(key, "/")
		parsedHostID := slicedKeys[len(slicedKeys)-1]
		CBLogger.Tracef("ParsedHostID: %v", parsedHostID)

		if parsedHostID != hostID {
			// Update keyring (including add)
			CBLogger.Debug("Update keyring")
			CBNet.UpdateKeyring(parsedHostID, string(kv.Value))
		}
	}

	base64PublicKey, _ := CBNet.GetPublicKeyBase64()
	CBLogger.Tracef("Base64PublicKey: %+v", base64PublicKey)

	// Transaction (compare-and-swap(CAS)) the secret
	keySecretHost := fmt.Sprint(etcdkey.Secret + "/" + cladnetID + "/" + hostID)
	CBLogger.Debugf("Transaction (compare-and-swap(CAS)) - %v", keySecretHost)

	// NOTICE: "!=" doesn't work..... It might be a temporal issue.
	txnResp, err := etcdClient.Txn(context.TODO()).
		If(clientv3.Compare(clientv3.Value(keySecretHost), "=", base64PublicKey)).
		Else(clientv3.OpPut(keySecretHost, base64PublicKey)).
		Commit()

	if err != nil {
		CBLogger.Error(err)
	}

	CBLogger.Tracef("TransactionResponse: %#v", txnResp)
	CBLogger.Tracef("ResponseHeader: %#v", txnResp.Header)

	// Release lock
	CBLogger.Debug("Release a lock")
	if err := lock.Unlock(ctx); err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Released lock for '%s'", keyPrefix)

	CBLogger.Debug("End.........")
}

/////////////////////////////////////////////
/////////////////////////////////////////////
/////////////////////////////////////////////

func watchPeer(wg *sync.WaitGroup, etcdClient *clientv3.Client, controllerID string) {
	defer wg.Done()
	// Watch "/registry/cloud-adaptive-network/host-network-information"
	CBLogger.Debugf("Start to watch \"%v\"", etcdkey.Peer)

	// Create a session to acquire a lock
	session, _ := concurrency.NewSession(etcdClient)
	defer session.Close()

	watchChan2 := etcdClient.Watch(context.Background(), etcdkey.Peer, clientv3.WithPrefix())
	for watchResponse := range watchChan2 {
		for _, event := range watchResponse.Events {
			switch event.Type {
			case mvccpb.PUT:
				CBLogger.Tracef("\n[cb-network controller (%s)]\nWatch - %s %q : %q",
					controllerID, event.Type, event.Kv.Key, event.Kv.Value)

				// Try to acquire a workload by multiple cb-network controllers
				isAcquired := tryToAcquireWorkload(etcdClient, controllerID, string(event.Kv.Key), watchResponse.Header.GetRevision())

				// Proceed the following by a cb-network controller acquiring the workload
				if isAcquired {
					// Parse HostID and CLADNetID from the Key
					slicedKeys := strings.Split(string(event.Kv.Key), "/")
					// parsedHostID := slicedKeys[len(slicedKeys)-1]
					// CBLogger.Tracef("ParsedHostId: %v", parsedHostID)
					parsedCLADNetID := slicedKeys[len(slicedKeys)-2]
					CBLogger.Tracef("ParsedCLADNetId: %v", parsedCLADNetID)

					// Prepare lock
					keyPrefix := fmt.Sprint(etcdkey.LockPeer + "/" + parsedCLADNetID)

					lock := concurrency.NewMutex(session, keyPrefix)
					ctx := context.TODO()

					// Acquire lock (or wait to have it)
					CBLogger.Debug("Acquire a lock")
					if err := lock.Lock(ctx); err != nil {
						CBLogger.Error(err)
					}
					CBLogger.Tracef("Acquired lock for '%s'", keyPrefix)

					updateNetworkingRule(parsedCLADNetID, etcdClient, controllerID)

					// Release lock
					CBLogger.Debug("Release a lock")
					if err := lock.Unlock(ctx); err != nil {
						CBLogger.Error(err)
					}
					CBLogger.Tracef("Released lock for '%s'", keyPrefix)
				}

			case mvccpb.DELETE: // The watched key has been deleted.
				CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			default:
				CBLogger.Errorf("Known event (%s), Key(%q), Value(%q)", event.Type, event.Kv.Key, event.Kv.Value)
			}

		}
	}
	CBLogger.Debug("End.........")
}

func updateNetworkingRule(cladnetID string, etcdClient *clientv3.Client, controllerID string) {
	CBLogger.Debug("Start.........")
	// Get peers in a Cloud Adaptive Network
	keyPeersInCLADNet := fmt.Sprint(etcdkey.Peer + "/" + cladnetID)
	CBLogger.Tracef("Get - %v", keyPeersInCLADNet)

	respPeers, etcdErr := etcdClient.Get(context.Background(), keyPeersInCLADNet, clientv3.WithPrefix())
	if etcdErr != nil {
		CBLogger.Error(etcdErr)
	}
	CBLogger.Tracef("GetResponse: %v", respPeers)
	CBLogger.Tracef("The number of peers (Count): %v", respPeers.Count)

	if respPeers.Count >= 2 {

		// Get a specification of a cloud adaptive network
		keyCLADNetSpec := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + cladnetID)
		CBLogger.Tracef("Get - %v", keyCLADNetSpec)

		respCLADNetSpec, etcdErr := etcdClient.Get(context.Background(), keyCLADNetSpec)
		if etcdErr != nil {
			CBLogger.Error(etcdErr)
		}
		CBLogger.Tracef("GetResponse: %v", respCLADNetSpec)

		var cladnetSpec model.CLADNetSpecification
		if err := json.Unmarshal(respCLADNetSpec.Kvs[0].Value, &cladnetSpec); err != nil {
			CBLogger.Error(err)
		}
		CBLogger.Tracef("The CLADNet spec: %v", cladnetSpec)

		var wg sync.WaitGroup

		// Set the networking rule for each peer
		for _, kv := range respPeers.Kvs {
			// Value
			sourcePeerBytes := kv.Value
			var sourcePeer model.Peer
			if err := json.Unmarshal(sourcePeerBytes, &sourcePeer); err != nil {
				CBLogger.Error(err)
			}
			CBLogger.Tracef("The source peer: %v", sourcePeer)

			// Update networking rule for each peer in parallel
			wg.Add(1)
			CBLogger.Tracef("Update the networking rule of peer (ID: %+v, Name: %+v)", sourcePeer.HostID, sourcePeer.HostName)
			go updateNetworkingRuleOfPeer(cladnetSpec.RuleType, sourcePeer, respPeers.Kvs, etcdClient, &wg)
		}
		wg.Wait()
	}
	CBLogger.Debug("End.........")
}

func updateNetworkingRuleOfPeer(ruleType string, sourcePeer model.Peer, peerKvs []*mvccpb.KeyValue, etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Trace("Start.........")
	defer wg.Done()

	var networkingRule model.NetworkingRule

	// Get the networking rule
	keyNetworkingRuleOfPeer := fmt.Sprint(etcdkey.NetworkingRule + "/" + sourcePeer.CladnetID + "/" + sourcePeer.HostID)
	CBLogger.Debugf("Get - %v", keyNetworkingRuleOfPeer)

	respNetworkingRule, etcdErr := etcdClient.Get(context.Background(), keyNetworkingRuleOfPeer)
	if etcdErr != nil {
		CBLogger.Error(etcdErr)
	}
	CBLogger.Tracef("GetResponse: %v", respNetworkingRule)
	CBLogger.Tracef("The number of peers (Count): %v", respNetworkingRule.Count)

	if respNetworkingRule.Count > 0 {
		err := json.Unmarshal(respNetworkingRule.Kvs[0].Value, &networkingRule)
		if err != nil {
			CBLogger.Error(err)
		}
	}

	networkingRule.CladnetID = sourcePeer.CladnetID

	// Create networking rule table for each peer
	for _, peerKv := range peerKvs {
		// Value
		peerBytes := peerKv.Value
		var peer model.Peer
		if err := json.Unmarshal(peerBytes, &peer); err != nil {
			CBLogger.Error(err)
		}
		CBLogger.Tracef("A peer: %v", peer)

		if sourcePeer.HostID != peer.HostID {
			// Select destination IP
			selectedIP, peerScope, err := cbnet.SelectDestinationByRuleType(ruleType, sourcePeer, peer)
			if err != nil {
				CBLogger.Error(err)
			}

			CBLogger.Tracef("Selected IP: %+v", selectedIP)

			networkingRule.UpdateRule(peer.HostID, peer.HostName, peer.IP, selectedIP, peerScope, peer.State)
		}
	}

	// Transaction (compare-and-swap(CAS)) to put networking rule for a peer
	CBLogger.Debugf("Transaction (compare-and-swap(CAS)) - %v", keyNetworkingRuleOfPeer)
	networkingRuleBytes, _ := json.Marshal(networkingRule)
	networkingRuleString := string(networkingRuleBytes)

	// NOTICE: "!=" doesn't work..... It might be a temporal issue.
	txnResp, err := etcdClient.Txn(context.TODO()).
		If(clientv3.Compare(clientv3.Value(keyNetworkingRuleOfPeer), "=", networkingRuleString)).
		Else(clientv3.OpPut(keyNetworkingRuleOfPeer, networkingRuleString)).
		Commit()

	if err != nil {
		CBLogger.Error(err)
	}

	CBLogger.Tracef("TransactionResponse: %#v", txnResp)
	CBLogger.Tracef("ResponseHeader: %#v", txnResp.Header)

	CBLogger.Trace("End.........")
}

/////////////////////////////////////////////
/////////////////////////////////////////////
/////////////////////////////////////////////

func main() {
	CBLogger.Debug("Start.........")

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	// A context for graceful shutdown (It is based on the signal package)
	// NOTE -
	// Use os.Interrupt Ctrl+C or Ctrl+Break on Windows
	// Use syscall.KILL for Kill(can't be caught or ignored) (POSIX)
	// Use syscall.SIGTERM for Termination (ANSI)
	// Use syscall.SIGINT for Terminal interrupt (ANSI)
	// Use syscall.SIGQUIT for Terminal quit (POSIX)
	// Use syscall.SIGHUP for Hangup (POSIX)
	// Use syscall.SIGABRT for Abort (POSIX)
	gracefulShutdownContext, stop := signal.NotifyContext(context.TODO(),
		os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGABRT)
	defer stop()

	// etcd Section
	// Connect to the etcd cluster
	etcdClient, etcdErr := clientv3.New(clientv3.Config{
		Endpoints:   config.ETCD.Endpoints,
		DialTimeout: 5 * time.Second,
	})

	if etcdErr != nil {
		CBLogger.Fatal(etcdErr)
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
	var hostName string
	var networkInterfaceName string
	var tunnelingPort int

	// Set host name
	if config.CBNetwork.Host.Name == "" {
		name, err := os.Hostname()
		if err != nil {
			CBLogger.Error(err)
		}
		hostName = name
	} else {
		hostName = config.CBNetwork.Host.Name
	}

	// Set network interface name
	if config.CBNetwork.Host.NetworkInterfaceName == "" {
		networkInterfaceName = "cbnet0"
	} else {
		networkInterfaceName = config.CBNetwork.Host.NetworkInterfaceName
	}

	// Set tunneling port
	if config.CBNetwork.Host.TunnelingPort == "" {
		tunnelingPort = 8055
	} else {
		tunnelingPort, etcdErr = strconv.Atoi(config.CBNetwork.Host.TunnelingPort)
		if etcdErr != nil {
			CBLogger.Error(etcdErr)
		}
	}

	// Create CBNetwork instance with port, which is a tunneling port
	CBNet = cbnet.New(networkInterfaceName, tunnelingPort)
	CBNet.ConfigureHostID()
	CBNet.CLADNetID = cladnetID
	CBNet.HostName = hostName

	// Enable encryption or not
	CBNet.EnableEncryption(config.CBNetwork.Host.IsEncrypted)

	wg.Add(1)
	// Watch the other agents' secrets (RSA public keys)
	go watchSecret(gracefulShutdownContext, etcdClient, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	wg.Add(1)
	// Watch the control command from the remote
	go watchControlCommand(gracefulShutdownContext, etcdClient, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	wg.Add(1)
	// Watch this peer
	go watchThisPeer(gracefulShutdownContext, etcdClient, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	wg.Add(1)
	// Watch the networking rule to update dynamically
	go watchNetworkingRule(gracefulShutdownContext, etcdClient, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	// Turn up the network interface (TUN) for Cloud Adaptive Network
	handleCommand(cmdtype.Up, etcdClient)

	wg.Add(1)
	// Watch the test request from the remote
	go watchTestRequest(gracefulShutdownContext, etcdClient, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		// Block until a signal is triggered
		<-gracefulShutdownContext.Done()

		// Stop this cb-network agent
		fmt.Println("[Stop] cb-network agent")
		CBNet.CloseCBNetworkInterface()
		// Set this agent state "Released"
		updatePeerState(netstate.Released, etcdClient)

		// Wait for a while
		time.Sleep(1 * time.Second)
	}(&wg)

	// Waiting for all goroutines to finish
	CBLogger.Info("Waiting for all goroutines to finish")
	wg.Wait()

	CBLogger.Debug("End cb-network agent .........")
}
