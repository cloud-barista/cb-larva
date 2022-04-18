package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/etcd-key"
	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	netstate "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/network-state"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config model.Config

func init() {
	fmt.Println("Start......... init() of controller.go")

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
		fmt.Printf("exe path: %v\n", exePath)

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
	fmt.Printf("exe path: %v\n", exePath)

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

	fmt.Println("End......... init() of controller.go")
}

// func watchCLADNetSpecification(wg *sync.WaitGroup, etcdClient *clientv3.Client) {
// 	defer wg.Done()

// 	// Watch "/registry/cloud-adaptive-network/cladnet-specification"
// 	CBLogger.Debugf("Start to watch \"%v\"", etcdkey.CLADNetSpecification)
// 	watchChan1 := etcdClient.Watch(context.Background(), etcdkey.CLADNetSpecification, clientv3.WithPrefix(), clientv3.WithRev(1))
// 	for watchResponse := range watchChan1 {
// 		for _, event := range watchResponse.Events {
// 			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
// 			slicedKeys := strings.Split(string(event.Kv.Key), "/")
// 			for _, value := range slicedKeys {
// 				fmt.Println(value)
// 			}

// 			var cladnetSpec pb.CLADNetSpecification
// 			errUnmarshal := json.Unmarshal(event.Kv.Value, &cladnetSpec)
// 			if errUnmarshal != nil {
// 				CBLogger.Error(errUnmarshal)
// 			}

// 			CBLogger.Tracef("The requested CLADNet specification: %v", cladnetSpec.String())

// 			// Generate a unique CLADNet ID by the xid package
// 			guid := xid.New()
// 			CBLogger.Tracef("A unique CLADNet ID: %v", guid)
// 			cladnetSpec.Id = guid.String()

// 			// Currently assign the 1st IP address for Gateway IP (Not used till now)
// 			ipv4Address, _, errParseCIDR := net.ParseCIDR(cladnetSpec.Ipv4AddressSpace)
// 			if errParseCIDR != nil {
// 				CBLogger.Fatal(errParseCIDR)
// 			}
// 			CBLogger.Tracef("IPv4Address: %v", ipv4Address)

// 			// Assign gateway IP address
// 			// ip := ipv4Address.To4()
// 			// gatewayIP := nethelper.IncrementIP(ip, 1)
// 			// cladnetSpec.GatewayIP = gatewayIP.String()
// 			// CBLogger.Tracef("GatewayIP: %v", cladNetSpec.GatewayIP)

// 			// Put the specification of the CLADNet to the etcd
// 			keyCLADNetSpecificationOfCLADNet := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + cladnetSpec.Id)
// 			strCLADNetSpec, _ := json.Marshal(cladnetSpec.String())
// 			_, err := etcdClient.Put(context.Background(), keyCLADNetSpecificationOfCLADNet, string(strCLADNetSpec))
// 			if err != nil {
// 				CBLogger.Fatal(err)
// 			}
// 		}
// 	}
// 	CBLogger.Debugf("End to watch \"%v\"", etcdkey.CLADNetSpecification)
// }

func watchHostNetworkInformation(wg *sync.WaitGroup, etcdClient *clientv3.Client, controllerID string) {
	defer wg.Done()
	// Watch "/registry/cloud-adaptive-network/host-network-information"
	CBLogger.Debugf("Start to watch \"%v\"", etcdkey.HostNetworkInformation)

	// Create a session to acquire a lock
	session, _ := concurrency.NewSession(etcdClient)
	defer session.Close()

	watchChan2 := etcdClient.Watch(context.Background(), etcdkey.HostNetworkInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan2 {
		for _, event := range watchResponse.Events {
			switch event.Type {
			case mvccpb.PUT: // The watched value has changed.
				CBLogger.Tracef("\n[cb-network controller (%s)]\nWatch - %s %q : %q",
					controllerID, event.Type, event.Kv.Key, event.Kv.Value)

				// Try to acquire a workload by multiple cb-network controllers
				isAcquired := tryToAcquireWorkload(etcdClient, controllerID, string(event.Kv.Key), watchResponse.Header.GetRevision())

				// Proceed the following by a cb-network controller acquiring the workload
				if isAcquired {

					var hostNetworkInformation model.HostNetworkInformation
					if err := json.Unmarshal(event.Kv.Value, &hostNetworkInformation); err != nil {
						CBLogger.Error(err)
					}
					hostName := hostNetworkInformation.HostName
					hostPublicIP := hostNetworkInformation.PublicIP

					// Find default host network interface and set IP and IPNetwork
					hostIP, hostIPNetwork, err := getDefaultInterfaceInfo(hostNetworkInformation.NetworkInterfaces)
					if err != nil {
						CBLogger.Error(err)
						continue
					}

					// Parse HostID and CLADNetID from the Key
					slicedKeys := strings.Split(string(event.Kv.Key), "/")
					parsedHostID := slicedKeys[len(slicedKeys)-1]
					CBLogger.Tracef("ParsedHostId: %v", parsedHostID)
					parsedCLADNetID := slicedKeys[len(slicedKeys)-2]
					CBLogger.Tracef("ParsedCLADNetId: %v", parsedCLADNetID)

					// Prepare lock
					keyPrefix := fmt.Sprint(etcdkey.LockPeer + "/" + parsedCLADNetID)

					lock := concurrency.NewMutex(session, keyPrefix)
					ctx := context.TODO()

					// Acquire a lock to protect a networking rule
					CBLogger.Debug("Acquire a lock")
					if err := lock.Lock(ctx); err != nil {
						CBLogger.Errorf("Could NOT acquire lock for '%v', error: %v", keyPrefix, err)
					}
					CBLogger.Tracef("Acquired lock for '%s'", keyPrefix)

					// Create a key of host in the specific CLADNet's networking rule
					keyPeer := fmt.Sprint(etcdkey.Peer + "/" + parsedCLADNetID + "/" + parsedHostID)

					// Get a host's networking rule
					CBLogger.Tracef("Key: %v", keyPeer)
					respRule, respRuleErr := etcdClient.Get(context.TODO(), keyPeer)
					if respRuleErr != nil {
						CBLogger.Error(respRuleErr)
					}

					var peer model.Peer

					// Newly allocate the host's configuration
					if respRule.Count == 0 {

						peer = allocatePeer(parsedCLADNetID, parsedHostID, hostName, hostIPNetwork, hostIP, hostPublicIP, etcdClient)

					} else { // Update the host's configuration

						if err := json.Unmarshal(respRule.Kvs[0].Value, &peer); err != nil {
							CBLogger.Error(err)
						}

						peer.HostPrivateIPNetwork = hostIPNetwork
						peer.HostPrivateIP = hostIP
						peer.HostPublicIP = hostNetworkInformation.PublicIP
						peer.State = netstate.Configuring
					}

					CBLogger.Debugf("Put - %v", keyPeer)
					CBLogger.Tracef("Value: %#v", peer)

					doc, _ := json.Marshal(peer)
					if _, err := etcdClient.Put(context.TODO(), keyPeer, string(doc)); err != nil {
						CBLogger.Error(err)
					}

					// Release a lock to protect a networking rule
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
	CBLogger.Debugf("End to watch \"%v\"", etcdkey.HostNetworkInformation)
}

func tryToAcquireWorkload(etcdClient *clientv3.Client, controllerID string, key string, revision int64) bool {
	CBLogger.Debugf("Start (%s) .........", controllerID)
	// Key to lease temporally by which each cb-network controller can distinguish each updated value
	keyToLease := fmt.Sprintf("lease%s-%d", key, revision)
	// fmt.Printf("%#v\n", keyPrefix)

	// Self-assign a workload by Compare-and-Swap (CAS) and Lease
	lease := clientv3.NewLease(etcdClient)
	ttl := int64(15)
	grantResp, grantErr := lease.Grant(context.TODO(), ttl)
	if grantErr != nil {
		CBLogger.Errorf("\n[cb-network controller (%s)]\n'lease.Grant' error: %#v",
			controllerID, grantErr)
	}

	messageToCheck := fmt.Sprintf("Vanished in %d sec", ttl)
	txResp, err2 := etcdClient.Txn(context.TODO()).
		If(clientv3.Compare(clientv3.Value(keyToLease), "=", messageToCheck)).
		Then(clientv3.OpGet(keyToLease)).
		Else(clientv3.OpPut(keyToLease, messageToCheck, clientv3.WithLease(grantResp.ID))).
		Commit()

	if err2 != nil {
		CBLogger.Errorf("\n[cb-network controller (%s)]\ntransaction error: %#v",
			controllerID, err2)
	}

	CBLogger.Tracef("[%s] txResp: %v\n", controllerID, txResp)
	isAcquired := !txResp.Succeeded

	if isAcquired {
		CBLogger.Debugf("[%s] acquires by '%s'", controllerID, keyToLease)
	} else {
		CBLogger.Debugf("[%s] '%s' already occupied by the other cb-network controlller", controllerID, keyToLease)
	}

	CBLogger.Debugf("End (%s) .........", controllerID)
	return isAcquired
}

func getDefaultInterfaceInfo(networkInterfaces []model.NetworkInterface) (ipAddr string, ipNet string, err error) {
	// Find default host network interface and set IP and IPNetwork

	for _, networkInterface := range networkInterfaces {
		if networkInterface.Name == "eth0" || networkInterface.Name == "ens4" || networkInterface.Name == "ens5" {
			return networkInterface.IPv4, networkInterface.IPv4Network, nil
		}
	}
	return "", "", errors.New("could not find default network interface")
}

func getIpv4AddressSpace(etcdClient *clientv3.Client, key string) (string, error) {

	respSpec, errSpec := etcdClient.Get(context.Background(), key)
	if errSpec != nil {
		CBLogger.Error(errSpec)
	}

	var tempSpec model.CLADNetSpecification

	// Unmarshal the specification of the CLADNet if exists
	CBLogger.Tracef("RespRule.Kvs: %v", respSpec.Kvs)
	if len(respSpec.Kvs) != 0 {
		errUnmarshal := json.Unmarshal(respSpec.Kvs[0].Value, &tempSpec)
		if errUnmarshal != nil {
			CBLogger.Error(errUnmarshal)
		}
		CBLogger.Tracef("TempSpec: %v", tempSpec)
		// Get an IPv4 address space of CLADNet
		return tempSpec.Ipv4AddressSpace, nil
	}
	return "", errors.New("no cloud adaptive network exists")
}

func assignIPAddressToPeer(ipNetwork string, numberOfIPsAssigned uint32) (string, string, error) {
	// Get IPNet struct from string
	_, ipv4Net, errParseCIDR := net.ParseCIDR(ipNetwork)
	if errParseCIDR != nil {
		CBLogger.Error(errParseCIDR)
		return "", "", errParseCIDR
	}

	// Get NetworkAddress(uint32) (The first IP address of this CLADNet)
	firstIP := binary.BigEndian.Uint32(ipv4Net.IP)
	CBLogger.Tracef("Network address: %s(%d)", ipv4Net.IP.String(), firstIP)

	// Get Subnet Mask(uint32) from IPNet struct
	subnetMask := binary.BigEndian.Uint32(ipv4Net.Mask)
	CBLogger.Tracef("Subnet mask: %s(%d)", ipv4Net.Mask.String(), subnetMask)

	// Get BroadcastAddress(uint32) (The last IP address of this CLADNet)
	lastIP := (firstIP & subnetMask) | (subnetMask ^ 0xffffffff)

	var broadcastAddress = make(net.IP, 4)
	binary.BigEndian.PutUint32(broadcastAddress, lastIP)
	CBLogger.Tracef("Broadcast address: %s(%d)", fmt.Sprint(broadcastAddress), lastIP)

	// Get a candidate of IP Address in serial order to assign IP Address to a client
	// Exclude Network Address, Broadcast Address, Gateway Address
	ipCandidate := firstIP + numberOfIPsAssigned

	// Create IP address of type net.IP. IPv4 is 4 bytes, IPv6 is 16 bytes.
	var ip = make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, ipCandidate)

	// Get CIDR Prefix
	cidrPrefix, _ := ipv4Net.Mask.Size()
	// Create Host IP CIDR Block
	hostIPv4Network := fmt.Sprint(ip, "/", cidrPrefix)
	// To string IP Address
	hostIPAddress := fmt.Sprint(ip)

	if ipCandidate >= lastIP {
		errStr := fmt.Sprintf("IP (%v) is out of ipv4Net's range (%v)", ip.String(), ipv4Net.IP.String())
		CBLogger.Errorf(errStr)
		return hostIPv4Network, hostIPAddress, errors.New(errStr)
	}

	return hostIPv4Network, hostIPAddress, nil
}

func allocatePeer(cladnetID string, hostID string, hostName string, hostIPNetwork string, hostIP string, hostPublicIP string, etcdClient *clientv3.Client) model.Peer {

	// Create a key of the CLADNet specification
	keyCLADNetSpecificationOfCLADNet := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + cladnetID)

	// Get the CLADNet specification to check IPv4AddressSpace
	cladnetIpv4AddressSpace, err := getIpv4AddressSpace(etcdClient, keyCLADNetSpecificationOfCLADNet)
	if err != nil {
		CBLogger.Error(err)
	}

	// Create a key of host in the specific CLADNet's networking rule
	keyPeer := fmt.Sprint(etcdkey.Peer + "/" + cladnetID)

	// Get the count of networking rule
	CBLogger.Tracef("Key: %v", keyPeer)
	resp, respErr := etcdClient.Get(context.TODO(), keyPeer, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if respErr != nil {
		CBLogger.Error(respErr)
	}

	state := netstate.Configuring
	peerIPv4Network, peerIPAddress, err := assignIPAddressToPeer(cladnetIpv4AddressSpace, uint32(resp.Count+2))
	if err != nil {
		CBLogger.Error(err)
		state = netstate.Failed

	}

	// "0.0.0.0" will be assigned in error case
	peer := model.Peer{
		CLADNetID:            cladnetID,
		HostID:               hostID,
		HostName:             hostName,
		HostPrivateIPNetwork: hostIPNetwork,
		HostPrivateIP:        hostIP,
		HostPublicIP:         hostPublicIP,
		IPNetwork:            peerIPv4Network,
		IP:                   peerIPAddress,
		State:                state,
	}

	return peer

}

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
					parsedHostID := slicedKeys[len(slicedKeys)-1]
					CBLogger.Tracef("ParsedHostId: %v", parsedHostID)
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

					// Get peers
					keyPeersInCLADNet := fmt.Sprint(etcdkey.Peer + "/" + parsedCLADNetID)
					CBLogger.Debugf("Get - %v", keyPeersInCLADNet)

					respPeers, etcdErr := etcdClient.Get(context.Background(), keyPeersInCLADNet, clientv3.WithPrefix())
					if etcdErr != nil {
						CBLogger.Error(etcdErr)
					}
					CBLogger.Tracef("GetResponse: %v", respPeers)
					CBLogger.Tracef("The number of peers (Count): %v", respPeers.Count)

					if respPeers.Count >= 2 {

						var wg sync.WaitGroup

						// Set the networking rule for each peer
						for _, kv := range respPeers.Kvs {

							// // Key
							// key := string(kv.Key)
							// CBLogger.Tracef("Key: %v", key)

							// Value
							sourcePeerBytes := kv.Value
							var sourcePeer model.Peer
							if err := json.Unmarshal(sourcePeerBytes, &sourcePeer); err != nil {
								CBLogger.Error(err)
							}
							CBLogger.Tracef("The source peer: %v", sourcePeer)

							// Update networking rule for each peer in parallel
							wg.Add(1)
							go updateNetworkingRuleOfPeer(sourcePeer, respPeers.Kvs, etcdClient, &wg)
						}

						wg.Wait()

					}

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

func updateNetworkingRuleOfPeer(sourcePeer model.Peer, peerKvs []*mvccpb.KeyValue, etcdClient *clientv3.Client, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")
	defer wg.Done()

	var networkingRule model.NetworkingRule

	// Get the networking rule
	keyNetworkingRuleOfPeer := fmt.Sprint(etcdkey.NetworkingRule + "/" + sourcePeer.CLADNetID + "/" + sourcePeer.HostID)
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
			// A function will be added soon.
			selectedIP := peer.HostPublicIP

			networkingRule.UpdateRule(peer.CLADNetID, peer.HostName, peer.IP, selectedIP, peer.State)
		}
	}

	// Put networking rule for a peer
	CBLogger.Debugf("Put - %v", keyNetworkingRuleOfPeer)
	doc, _ := json.Marshal(networkingRule)
	if _, err := etcdClient.Put(context.TODO(), keyNetworkingRuleOfPeer, string(doc)); err != nil {
		CBLogger.Error(err)
	}

	CBLogger.Debug("End.........")
}

func main() {

	guid := xid.New()
	controllerID := guid.String()
	CBLogger.Debugf("Start cb-network controller (%s) .........", controllerID)

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	// etcd Section
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

	wg.Add(1)
	go watchHostNetworkInformation(&wg, etcdClient, controllerID)

	wg.Add(1)
	go watchPeer(&wg, etcdClient, controllerID)

	// Waiting for all goroutines to finish
	CBLogger.Info("Waiting for all goroutines to finish")
	wg.Wait()

	CBLogger.Debugf("End cb-network controller (%s) .........", controllerID)
}
