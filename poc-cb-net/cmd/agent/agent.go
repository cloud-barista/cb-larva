package main

import (
	"context"
	"encoding/binary"
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
var loggerNamePrefix = "agent"

func init() {
	fmt.Println("\nStart......... init() of agent.go")

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
	fmt.Printf("Load %v", configPath)

	// Cloud Adaptive Network section
	// Config
	cladnetID := config.CBNetwork.CLADNetID
	tunnelingPort := config.CBNetwork.Host.TunnelingPort
	networkInterfaceName := config.CBNetwork.Host.NetworkInterfaceName
	var hostName string

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

	// Create CBNetwork instance with port, which is a tunneling port
	CBNet = cbnet.New(networkInterfaceName, tunnelingPort)
	CBNet.ConfigureHostID()
	CBNet.CLADNetID = cladnetID
	CBNet.HostName = hostName

	loggerName := fmt.Sprintf("%s-%s", loggerNamePrefix, CBNet.HostID)

	// Set cb-log
	logConfPath := ""
	env := os.Getenv("CBLOG_ROOT")
	if env != "" {
		// Load cb-log config from the environment variable path (default)
		fmt.Printf("CBLOG_ROOT: %v\n", env)
		CBLogger = cblog.GetLogger(loggerName)
	} else {

		// Load cb-log config from the current directory (usually for the production)
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exePath := filepath.Dir(ex)
		// fmt.Printf("exe path: %v\n", exePath)

		logConfPath = filepath.Join(exePath, "config", "log_conf.yaml")
		if file.Exists(logConfPath) {
			fmt.Printf("path of log_conf.yaml: %v\n", logConfPath)
			CBLogger = cblog.GetLoggerWithConfigPath(loggerName, logConfPath)

		} else {
			// Load cb-log config from the project directory (usually for development)
			logConfPath = filepath.Join(exePath, "..", "..", "config", "log_conf.yaml")
			if file.Exists(logConfPath) {
				fmt.Printf("path of log_conf.yaml: %v\n", logConfPath)
				CBLogger = cblog.GetLoggerWithConfigPath(loggerName, logConfPath)
			} else {
				err := errors.New("fail to load log_conf.yaml")
				panic(err)
			}
		}
		fmt.Printf("Load %v", configPath)
	}

	CBLogger.Debugf("Load %v", configPath)
	CBLogger.Debugf("Load %v", logConfPath)

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
	CBLogger.Debugf("Watch - %v", keyControlCommand)
	watchChan1 := etcdClient.Watch(ctx, keyControlCommand)
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Pushed - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

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

		state := CBNet.ThisPeerState()
		CBLogger.Debugf("current state: %+v", state)
		if state == netstate.Tunneling {
			updatePeerState(netstate.Closing, etcdClient)
			CBNet.CloseCBNetworkInterface()
			updatePeerState(netstate.Released, etcdClient)
		}

	case cmdtype.Up:
		CBLogger.Debug("configure the cb-network interface in 'released' or '' state")

		state := CBNet.ThisPeerState()
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
	CBLogger.Debugf("Watch - %v", keyTestRequest)
	watchChan1 := etcdClient.Watch(ctx, keyTestRequest)
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Pushed - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

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

		state := CBNet.ThisPeerState()
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
	CBLogger.Debugf("Put - %v", keyStatusInformation)

	strNetworkStatus, _ := json.Marshal(networkStatus)
	size := binary.Size(strNetworkStatus)
	CBLogger.WithField("total size", size).Tracef("PutRequest size (bytes)")

	putResp, err := etcdClient.Put(context.Background(), keyStatusInformation, string(strNetworkStatus))
	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("PutResponse: %#v", putResp)

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

func initializeAgent(etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	cladnetID := CBNet.CLADNetID
	hostID := CBNet.HostID

	// Get this host's network information
	CBLogger.Debug("Get the host network information")
	CBNet.UpdateHostNetworkInformation()
	temp := CBNet.GetHostNetworkInformation()
	currentHostNetworkInformationBytes, _ := json.Marshal(temp)
	currentHostNetworkInformation := string(currentHostNetworkInformationBytes)
	CBLogger.Trace(currentHostNetworkInformation)

	keyHostNetworkInformation := fmt.Sprint(etcdkey.HostNetworkInformation + "/" + cladnetID + "/" + hostID)
	CBLogger.Debugf("Put - %v", keyHostNetworkInformation)

	size := binary.Size(currentHostNetworkInformationBytes)
	CBLogger.WithField("total size", size).Tracef("PutRequest size (bytes)")

	putResp, err := etcdClient.Put(context.TODO(), keyHostNetworkInformation, currentHostNetworkInformation)
	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("PutResponse: %#v", putResp)

	CBLogger.Debug("End.........")
}

func updatePeerState(state string, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	tempPeer := CBNet.ThisPeer
	tempPeer.State = state

	keyPeer := fmt.Sprint(etcdkey.Peer + "/" + CBNet.CLADNetID + "/" + CBNet.HostID)

	CBLogger.Debugf("Put - %v", keyPeer)
	doc, _ := json.Marshal(tempPeer)

	size := binary.Size(doc)
	CBLogger.WithField("total size", size).Tracef("PutRequest size (bytes)")

	putResp, err := etcdClient.Put(context.TODO(), keyPeer, string(doc))
	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("PutResponse: %#v", putResp)

	CBLogger.Debug("End.........")
}

func watchSecret(ctx context.Context, etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")

	defer wg.Done()

	// Watch "/registry/cloud-adaptive-network/secret/{cladnet-id}"
	keySecretGroup := fmt.Sprint(etcdkey.Secret + "/" + CBNet.CLADNetID)
	CBLogger.Debugf("Watch with prefix - %v", keySecretGroup)
	watchChan1 := etcdClient.Watch(ctx, keySecretGroup, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Pushed - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
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
	// Time trying to acquire a lock
	start := time.Now()
	if err := lock.Lock(ctx); err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Lock acquired for '%s'", keyPrefix)

	// Get the secret
	keySecret := fmt.Sprint(etcdkey.Secret + "/" + cladnetID)
	CBLogger.Debugf("Get with prefix - %v", keySecret)
	getResp, etcdErr := etcdClient.Get(context.TODO(), keySecret, clientv3.WithPrefix())
	if etcdErr != nil {
		CBLogger.Error(etcdErr)
	}
	CBLogger.Tracef("GetResponse: %#v", getResp)

	fields := createFieldsForResponseSizes(*getResp)
	CBLogger.WithFields(fields).Tracef("GetResponse size (bytes)")

	// Set the other hosts' secrets
	for _, kv := range getResp.Kvs {
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
	CBLogger.Tracef("Lock released for '%s'", keyPrefix)
	// Elapsed time from the time trying to acquire a lock
	elapsed := time.Since(start)
	CBLogger.Tracef("Elapsed time for locking: %s", elapsed)

	CBLogger.Debug("End.........")
}

// Watch all peers related to the same Cloud Adaptive Network
func watchPeers(ctx context.Context, etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")
	defer wg.Done()

	// Watch "/registry/cloud-adaptive-network/peer/{cladnet-id}"
	keyPeersInCLADNet := fmt.Sprint(etcdkey.Peer + "/" + CBNet.CLADNetID)
	CBLogger.Debugf("Watch with prefix - %v", keyPeersInCLADNet)

	watchChan := etcdClient.Watch(ctx, keyPeersInCLADNet, clientv3.WithPrefix())
	for watchResponse := range watchChan {
		for _, event := range watchResponse.Events {
			switch event.Type {
			case mvccpb.PUT:
				CBLogger.Tracef("Pushed - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
				// key := string(event.Kv.Key)

				var peer model.Peer
				err := json.Unmarshal(event.Kv.Value, &peer)
				if err != nil {
					CBLogger.Error(err)
				}

				// Store peers to synchronize and use it in local
				// prevThisPeer := CBNet.ThisPeer
				CBNet.StorePeer(peer)

				// Initialize or update networking rule
				if peer.HostID == CBNet.HostID { // for this peer

					ruleType, err := getRuleType(etcdClient)
					if err != nil {
						CBLogger.Error(err)
						continue
					}

					// Configure a virtual network interface for Cloud Adaptive Network, if it is the configuring state
					if peer.State == netstate.Configuring {
						CBLogger.Debug("Configure a virtual network interface (i.e., TUN device)")
						err := CBNet.ConfigureCBNetworkInterface()
						if err != nil {
							CBLogger.Error(err)
						}

						// Set initially the networking rule for this peer
						CBLogger.Debug("Initially set the networking rule for this peer")
						updateNetworkingRule(CBNet.ThisPeer, CBNet.OtherPeers, ruleType, etcdClient)

						// Update this peer's state to "tunneling"
						CBLogger.Debug("Change this peer's state to 'tunneling'")
						updatePeerState(netstate.Tunneling, etcdClient)

					} else if peer.State == netstate.Tunneling {

						// // Update networking rule if it's not a simple state chanage of this peer
						// if prevThisPeer.State == peer.State {
						// Update the networking rule for this peer
						CBLogger.Debug("Update the networking rule for this peer")
						updateNetworkingRule(CBNet.ThisPeer, CBNet.OtherPeers, ruleType, etcdClient)
						// }

					} else {
						CBLogger.Debugf("Skip to update the networking rule (this peer's state: %v)", peer.State)
					}

				} else { // for the other peers
					ruleType, err := getRuleType(etcdClient)
					if err != nil {
						CBLogger.Error(err)
						continue
					}

					// Keep updating networking rules if it is the tunneling state
					if CBNet.ThisPeerState() == netstate.Tunneling {
						updatePeerInNetworkingRule(CBNet.ThisPeer, peer, ruleType, etcdClient)
					}
				}

			case mvccpb.DELETE: // The watched key has been deleted.
				CBLogger.Tracef("Pushed - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			default:
				CBLogger.Errorf("Known event (%s), Key(%q), Value(%q)", event.Type, event.Kv.Key, event.Kv.Value)
			}

		}
	}
	CBLogger.Debug("End.........")
}

func getRuleType(etcdClient *clientv3.Client) (string, error) {
	CBLogger.Debug("Start.........")

	// Get a ruleType from the specification of Cloud Adaptive Network
	keyCLADNetSpec := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + CBNet.CLADNetID)
	CBLogger.Debugf("Get - %v", keyCLADNetSpec)

	respCLADNetSpec, etcdErr := etcdClient.Get(context.Background(), keyCLADNetSpec)
	if etcdErr != nil {
		CBLogger.Error(etcdErr)
		return "", etcdErr
	}
	CBLogger.Tracef("GetResponse: %#v", respCLADNetSpec)

	fields := createFieldsForResponseSizes(*respCLADNetSpec)
	CBLogger.WithFields(fields).Tracef("GetResponse size (bytes)")

	var cladnetSpec model.CLADNetSpecification
	if err := json.Unmarshal(respCLADNetSpec.Kvs[0].Value, &cladnetSpec); err != nil {
		CBLogger.Error(err)
		return "", err
	}
	CBLogger.Tracef("The CLADNet spec: %v", cladnetSpec)

	ruleType := cladnetSpec.RuleType
	CBLogger.Tracef("Rule type: %v", ruleType)

	CBLogger.Debug("End.........")
	return ruleType, nil
}

func updateNetworkingRule(thisPeer model.Peer, otherPeers map[string]model.Peer, ruleType string, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	countOtherPeers := len(otherPeers)
	if countOtherPeers > 0 {

		// Set the networking rule for this peer
		var networkingRule model.NetworkingRule
		networkingRule.CladnetID = thisPeer.CladnetID

		// Create networking rule table for each peer
		for _, peer := range otherPeers {

			CBLogger.Tracef("A peer: %v", peer)

			if thisPeer.HostID != peer.HostID {
				// Select destination IP
				selectedIP, peerScope, err := cbnet.SelectDestinationByRuleType(ruleType, thisPeer, peer)
				if err != nil {
					CBLogger.Error(err)
				}

				CBLogger.Tracef("Selected IP: %+v", selectedIP)
				networkingRule.UpdateRule(peer.HostID, peer.HostName, peer.IP, selectedIP, peerScope, peer.State)
			}
		}

		// Assign the networking rule
		CBNet.UpdateNetworkingRule(networkingRule)

		// Transaction (compare-and-swap(CAS)) to put networking rule for a peer
		keyNetworkingRuleOfThisPeer := fmt.Sprint(etcdkey.NetworkingRule + "/" + thisPeer.CladnetID + "/" + thisPeer.HostID)
		CBLogger.Debugf("Transaction (compare-and-swap(CAS)) - %v", keyNetworkingRuleOfThisPeer)
		networkingRuleBytes, _ := json.Marshal(networkingRule)
		networkingRuleString := string(networkingRuleBytes)

		size := binary.Size(networkingRuleBytes)
		CBLogger.WithField("total size", size).Tracef("TransactionRequest size (bytes)")

		// NOTICE: "!=" doesn't work..... It might be a temporal issue.
		txnResp, err := etcdClient.Txn(context.TODO()).
			If(clientv3.Compare(clientv3.Value(keyNetworkingRuleOfThisPeer), "=", networkingRuleString)).
			Else(clientv3.OpPut(keyNetworkingRuleOfThisPeer, networkingRuleString)).
			Commit()

		if err != nil {
			CBLogger.Error(err)
		}

		CBLogger.Tracef("TransactionResponse: %#v", txnResp)
		CBLogger.Tracef("ResponseHeader: %#v", txnResp.Header)
	}

	CBLogger.Debug("End.........")
}

func updatePeerInNetworkingRule(thisPeer model.Peer, otherPeer model.Peer, ruleType string, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")

	networkingRule := CBNet.NetworkingRule

	// Update networking rule for the peer

	// Select destination IP
	selectedIP, peerScope, err := cbnet.SelectDestinationByRuleType(ruleType, thisPeer, otherPeer)
	if err != nil {
		CBLogger.Error(err)
	}

	CBLogger.Tracef("Selected IP: %+v", selectedIP)

	networkingRule.UpdateRule(otherPeer.HostID, otherPeer.HostName, otherPeer.IP, selectedIP, peerScope, otherPeer.State)

	// Assign the networking rule
	CBNet.UpdateNetworkingRule(networkingRule)

	// Transaction (compare-and-swap(CAS)) to put networking rule for a peer
	keyNetworkingRuleOfThisPeer := fmt.Sprint(etcdkey.NetworkingRule + "/" + thisPeer.CladnetID + "/" + thisPeer.HostID)
	CBLogger.Debugf("Transaction (compare-and-swap(CAS)) - %v", keyNetworkingRuleOfThisPeer)
	networkingRuleBytes, _ := json.Marshal(networkingRule)
	networkingRuleString := string(networkingRuleBytes)

	size := binary.Size(networkingRuleBytes)
	CBLogger.WithField("total size", size).Tracef("TransactionRequest size (bytes)")

	// NOTICE: "!=" doesn't work..... It might be a temporal issue.
	txnResp, err := etcdClient.Txn(context.TODO()).
		If(clientv3.Compare(clientv3.Value(keyNetworkingRuleOfThisPeer), "=", networkingRuleString)).
		Else(clientv3.OpPut(keyNetworkingRuleOfThisPeer, networkingRuleString)).
		Commit()

	if err != nil {
		CBLogger.Error(err)
	}

	CBLogger.Tracef("TransactionResponse: %#v", txnResp)
	CBLogger.Tracef("ResponseHeader: %#v", txnResp.Header)

	CBLogger.Debug("End.........")
}

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

	// Create a session to acquire a lock
	session, _ := concurrency.NewSession(etcdClient)
	defer session.Close()

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

	// Synchronize all peers in local
	// Lock is required for secure synchronization.
	// Without the lock, it could be missing a part of peer updates while synchronizing and watching peers respectively.

	// Prepare lock
	keyPrefix := fmt.Sprint(etcdkey.LockPeer + "/" + CBNet.CLADNetID)

	lock := concurrency.NewMutex(session, keyPrefix)
	ctx := context.TODO()

	// Acquire lock (or wait to have it)
	CBLogger.Debug("Acquire a lock")
	// Time trying to acquire a lock
	start := time.Now()
	if err := lock.Lock(ctx); err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Lock acquired for '%s'", keyPrefix)

	wg.Add(1)
	// Watch all peers
	go watchPeers(gracefulShutdownContext, etcdClient, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	// Get all peers
	// Create a key of host in the specific CLADNet's networking rule
	keyPeersInCLADNet := fmt.Sprint(etcdkey.Peer + "/" + CBNet.CLADNetID)
	CBLogger.Debugf("Get with prefix - %v", keyPeersInCLADNet)

	getResp, respErr := etcdClient.Get(context.TODO(), keyPeersInCLADNet, clientv3.WithPrefix())
	if respErr != nil {
		CBLogger.Error(respErr)
	}
	CBLogger.Tracef("GetResponse: %#v", getResp)

	fields := createFieldsForResponseSizes(*getResp)
	CBLogger.WithFields(fields).Tracef("GetResponse size (bytes)")

	for _, kv := range getResp.Kvs {
		key := string(kv.Key)
		CBLogger.Tracef("Key : %v", key)
		CBLogger.Tracef("The peer: %v", string(kv.Value))

		var peer model.Peer
		err := json.Unmarshal(kv.Value, &peer)
		if err != nil {
			CBLogger.Error(err)
		}
		// Store peers to synchronize and use it in local
		CBNet.StorePeer(peer)
	}

	// Release lock
	CBLogger.Debug("Release a lock")
	if err := lock.Unlock(ctx); err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Lock released for '%s'", keyPrefix)
	// Elapsed time from the time trying to acquire a lock
	elapsed := time.Since(start)
	CBLogger.Tracef("Elapsed time for locking: %s", elapsed)

	// Turn up the virtual network interface (i.e., TUN device) for Cloud Adaptive Network
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

	CBLogger.Debug("End.........")
}

func createFieldsForResponseSizes(res clientv3.GetResponse) logrus.Fields {

	// lenKvs := res.Count
	fields := logrus.Fields{}

	headerSize := res.Header.Size()
	kvSize := 0
	for _, kv := range res.Kvs {
		kvSize += kv.Size()
	}

	totalSize := headerSize + kvSize

	fields["total size"] = totalSize
	fields["header size"] = headerSize
	fields["kvs size"] = kvSize

	for i, kv := range res.Kvs {
		tempKey := "kv " + strconv.Itoa(i)
		fields[tempKey] = kv.Size()
		// kvSize += kv.Size()
	}

	return fields
}
