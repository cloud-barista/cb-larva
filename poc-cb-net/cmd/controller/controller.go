package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/internal/etcd-key"
	file "github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config model.Config

func init() {
	fmt.Println("Start......... init() of controller.go")
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

	// // Create a session to acruie a lock
	// session, _ := concurrency.NewSession(etcdClient)
	// defer session.Close()

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

					// Parse HostID and CLADNetID from the Key
					slicedKeys := strings.Split(string(event.Kv.Key), "/")
					parsedHostID := slicedKeys[len(slicedKeys)-1]
					CBLogger.Tracef("ParsedHostId: %v", parsedHostID)
					parsedCLADNetID := slicedKeys[len(slicedKeys)-2]
					CBLogger.Tracef("ParsedCLADNetId: %v", parsedCLADNetID)

					// Create a key of the CLADNet specification
					keyCLADNetSpecificationOfCLADNet := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + parsedCLADNetID)

					// Get the CLADNet specification to check IPv4AddressSpace
					cladnetIpv4AddressSpace, err := getIpv4AddressSpace(etcdClient, keyCLADNetSpecificationOfCLADNet)
					if err != nil {
						CBLogger.Error(err)
					}

					// Create a key of host in the specific CLADNet's networking rule
					keyNetworkingRuleOfPeer := fmt.Sprint(etcdkey.NetworkingRule + "/" + parsedCLADNetID + "/" + parsedHostID)

					// // Needed?? (not sure yet)
					// lock := concurrency.NewMutex(session, keyNetworkingRuleOfHost)
					// ctx := context.TODO()

					// // Acquire a lock to protect a networking rule
					// if err := lock.Lock(ctx); err != nil {
					// 	CBLogger.Errorf("Could NOT acquire lock for '%v', error: %v", keyNetworkingRuleOfHost, err)
					// }
					// CBLogger.Debugf("Acquired lock for '%v'", keyNetworkingRuleOfHost)

					// Get a host's networking rule
					CBLogger.Tracef("Key: %v", keyNetworkingRuleOfPeer)
					respRule, respRuleErr := etcdClient.Get(context.TODO(), keyNetworkingRuleOfPeer)
					if respRuleErr != nil {
						CBLogger.Error(respRuleErr)
					}

					var peer model.Peer

					// Newly allocate the host's configuration
					if respRule.Count == 0 {
						// Create a key of host in the specific CLADNet's networking rule
						keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + parsedCLADNetID)

						// Get the count of networking rule
						CBLogger.Tracef("Key: %v", keyNetworkingRule)
						resp, respErr := etcdClient.Get(context.TODO(), keyNetworkingRule, clientv3.WithPrefix(), clientv3.WithCountOnly())
						if respErr != nil {
							CBLogger.Error(respErr)
						}

						hostIPv4Network, hostIPAddress := assignIPAddressToHost(cladnetIpv4AddressSpace, uint32(resp.Count+2))
						peer = model.Peer{
							CLADNetID:          parsedCLADNetID,
							HostID:             parsedHostID,
							PrivateIPv4Network: hostIPv4Network,
							PrivateIPv4Address: hostIPAddress,
							PublicIPv4Address:  hostNetworkInformation.PublicIP,
							State:              model.Suspended,
						}

					} else { // Update the host's configuration

						if err := json.Unmarshal(respRule.Kvs[0].Value, &peer); err != nil {
							CBLogger.Error(err)
						}

						peer.PublicIPv4Address = hostNetworkInformation.PublicIP
					}

					CBLogger.Debugf("Put \"%v\"", keyNetworkingRuleOfPeer)
					doc, _ := json.Marshal(peer)

					if _, err := etcdClient.Put(context.TODO(), keyNetworkingRuleOfPeer, string(doc)); err != nil {
						CBLogger.Error(err)
					}

					// // Release a lock to protect a networking rule
					// if err := lock.Unlock(ctx); err != nil {
					// 	CBLogger.Errorf("Cannot release lock for '%v', error: %v", keyNetworkingRuleOfHost, err)
					// }
					// CBLogger.Debugf("Released lock for '%v'", keyNetworkingRuleOfHost)
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
	keyToLease := fmt.Sprintf("lease/%s-%d", key, revision)
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

func assignIPAddressToHost(ipNetwork string, numberOfIPsAssigned uint32) (string, string) {
	// Get IPNet struct from string
	_, ipv4Net, errParseCIDR := net.ParseCIDR(ipNetwork)
	if errParseCIDR != nil {
		CBLogger.Error(errParseCIDR)
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
	if ipCandidate < lastIP {
		binary.BigEndian.PutUint32(ip, ipCandidate)
	} else {
		CBLogger.Error("This IP is out of range of the CLADNet")
	}

	// Get CIDR Prefix
	cidrPrefix, _ := ipv4Net.Mask.Size()
	// Create Host IP CIDR Block
	hostIPv4Network := fmt.Sprint(ip, "/", cidrPrefix)
	// To string IP Address
	hostIPAddress := fmt.Sprint(ip)

	return hostIPv4Network, hostIPAddress
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

	// Waiting for all goroutines to finish
	CBLogger.Info("Waiting for all goroutines to finish")
	wg.Wait()

	CBLogger.Debugf("End cb-network controller (%s) .........", controllerID)
}
