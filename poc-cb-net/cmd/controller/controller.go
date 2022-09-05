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
	"strconv"
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
var loggerNamePrefix = "controller"
var controllerID string

func init() {
	fmt.Println("\nStart......... init() of controller.go")

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

	// Generate a temporary ID for cb-network controller (it's not managed)
	guid := xid.New()
	controllerID = guid.String()

	loggerName := fmt.Sprintf("%s-%s", loggerNamePrefix, controllerID)

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
		fmt.Printf("Load %v", logConfPath)
	}

	CBLogger.Debugf("Load %v", configPath)
	CBLogger.Debugf("Load %v", logConfPath)

	fmt.Println("End......... init() of controller.go")
	fmt.Println("")
}

func watchHostNetworkInformation(wg *sync.WaitGroup, etcdClient *clientv3.Client) {
	CBLogger.Debug("Start.........")
	defer wg.Done()

	// Create a session to acquire a lock
	session, _ := concurrency.NewSession(etcdClient)
	defer session.Close()

	// Watch "/registry/cloud-adaptive-network/host-network-information"
	CBLogger.Debugf("Watch with prefix - %v", etcdkey.HostNetworkInformation)

	watchChan2 := etcdClient.Watch(context.Background(), etcdkey.HostNetworkInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan2 {
		for _, event := range watchResponse.Events {
			switch event.Type {
			case mvccpb.PUT: // The watched value has changed.
				CBLogger.Tracef("Pushed - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

				// Try to acquire a workload by multiple cb-network controllers
				isAcquired := tryToAcquireWorkload(etcdClient, string(event.Kv.Key), watchResponse.Header.GetRevision())

				// Proceed the following by a cb-network controller acquiring the workload
				if isAcquired {

					var hostNetworkInformation model.HostNetworkInformation
					if err := json.Unmarshal(event.Kv.Value, &hostNetworkInformation); err != nil {
						CBLogger.Error(err)
					}
					hostName := hostNetworkInformation.HostName
					hostPublicIP := hostNetworkInformation.PublicIP

					// Find default host network interface and set IP and IPv4CIDR
					hostIP, hostIPv4CIDR, err := getDefaultInterfaceInfo(hostNetworkInformation.NetworkInterfaces)
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

					// Acquire a lock  (or wait to have it) to update peer
					CBLogger.Debug("Acquire a lock")
					// Time trying to acquire a lock
					start := time.Now()
					err = lock.Lock(ctx)
					if err != nil {
						CBLogger.Errorf("Could NOT acquire lock for '%v', error: %v", keyPrefix, err)
					}
					CBLogger.Tracef("Lock acquired for '%s'", keyPrefix)

					// Create a key of peer
					keyPeer := fmt.Sprint(etcdkey.Peer + "/" + parsedCLADNetID + "/" + parsedHostID)

					// Get a peer
					CBLogger.Debugf("Get - %v", keyPeer)
					respRule, respRuleErr := etcdClient.Get(context.TODO(), keyPeer)
					if respRuleErr != nil {
						CBLogger.Error(respRuleErr)
					}
					CBLogger.Tracef("GetResponse: %#v", respRule)

					fields := createFieldsForResponseSizes(*respRule)
					CBLogger.WithFields(fields).Tracef("GetResponse size (bytes)")

					var peer model.Peer

					// Newly allocate the host's configuration
					if respRule.Count == 0 {

						peer = allocatePeer(parsedCLADNetID, parsedHostID, hostName, hostIPv4CIDR, hostIP, hostPublicIP, etcdClient)

					} else { // Update the host's configuration

						err = json.Unmarshal(respRule.Kvs[0].Value, &peer)
						if err != nil {
							CBLogger.Error(err)
						}

						peer.HostPrivateIPv4CIDR = hostIPv4CIDR
						peer.HostPrivateIP = hostIP
						peer.HostPublicIP = hostNetworkInformation.PublicIP
						peer.State = netstate.Configuring
					}

					peerBytes, _ := json.Marshal(peer)
					peerStr := string(peerBytes)

					CBLogger.Debugf("Put - %v", keyPeer)
					CBLogger.Tracef("Value: %#v", peer)

					size := binary.Size(peerBytes)
					CBLogger.WithField("total size", size).Tracef("PutRequest size (bytes)")

					putResp, err := etcdClient.Put(context.TODO(), keyPeer, peerStr)
					if err != nil {
						CBLogger.Error(err)
					}
					CBLogger.Tracef("PutResponse: %#v", putResp)

					// Release a lock to update peer
					CBLogger.Debug("Release a lock")
					if err := lock.Unlock(ctx); err != nil {
						CBLogger.Error(err)
					}
					CBLogger.Tracef("Lock released for '%s'", keyPrefix)
					// Elapsed time from the time trying to acquire a lock
					elapsed := time.Since(start)
					CBLogger.Tracef("Elapsed time for locking: %s", elapsed)
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

func tryToAcquireWorkload(etcdClient *clientv3.Client, key string, revision int64) bool {
	CBLogger.Debug("Start.........")
	// Key to lease temporally by which each cb-network controller can distinguish each updated value
	keyToLease := fmt.Sprintf("lease%s-%d", key, revision)
	// fmt.Printf("%#v\n", keyPrefix)

	// Self-assign a workload by Compare-and-Swap (CAS) and Lease
	lease := clientv3.NewLease(etcdClient)
	ttl := int64(15)
	grantResp, grantErr := lease.Grant(context.TODO(), ttl)
	if grantErr != nil {
		CBLogger.Errorf("'lease.Grant' error: %#v", grantErr)
	}

	CBLogger.Debugf("Transaction (compare-and-swap(CAS)) -  %v", keyToLease)

	messageToCheck := fmt.Sprintf("Vanished in %d sec", ttl)
	txResp, err2 := etcdClient.Txn(context.TODO()).
		If(clientv3.Compare(clientv3.Value(keyToLease), "=", messageToCheck)).
		Then(clientv3.OpGet(keyToLease)).
		Else(clientv3.OpPut(keyToLease, messageToCheck, clientv3.WithLease(grantResp.ID))).
		Commit()

	if err2 != nil {
		CBLogger.Errorf("transaction error: %#v", err2)
	}

	CBLogger.Tracef("TransactionResponse: %#v", txResp)
	isAcquired := !txResp.Succeeded

	if isAcquired {
		CBLogger.Debugf("acquires by '%s'", keyToLease)
	} else {
		CBLogger.Debugf("'%s' already occupied by the other cb-network controlller", keyToLease)
	}

	CBLogger.Debug("End.........")
	return isAcquired
}

func getDefaultInterfaceInfo(networkInterfaces []model.NetworkInterface) (ipAddr string, ipNet string, err error) {
	// Find default host network interface and set IP and IPv4CIDR

	for _, networkInterface := range networkInterfaces {
		if networkInterface.Name == "eth0" || networkInterface.Name == "ens4" || networkInterface.Name == "ens5" {
			return networkInterface.IPv4, networkInterface.IPv4CIDR, nil
		}
	}
	return "", "", errors.New("could not find default network interface")
}

func getIpv4AddressSpace(etcdClient *clientv3.Client, key string) (string, error) {
	CBLogger.Debug("Start.........")

	CBLogger.Debugf("Get - %v", key)
	respSpec, errSpec := etcdClient.Get(context.Background(), key)
	if errSpec != nil {
		CBLogger.Error(errSpec)
	}
	CBLogger.Tracef("GetResponse: %#v", respSpec)

	fields := createFieldsForResponseSizes(*respSpec)
	CBLogger.WithFields(fields).Tracef("GetResponse size (bytes)")

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
		CBLogger.Debug("End.........")
		return tempSpec.Ipv4AddressSpace, nil
	}
	CBLogger.Debug("End.........")
	return "", errors.New("no cloud adaptive network exists")
}

func assignIPAddressToPeer(ipCIDR string, numberOfIPsAssigned uint32) (string, string, error) {
	CBLogger.Debug("Start.........")

	// Get IPNet struct from string
	_, ipv4Net, errParseCIDR := net.ParseCIDR(ipCIDR)
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
	peerIPv4CIDR := fmt.Sprint(ip, "/", cidrPrefix)
	// To string IP Address
	peerIPAddress := fmt.Sprint(ip)

	if ipCandidate >= lastIP {
		errStr := fmt.Sprintf("IP (%v) is out of ipv4Net's range (%v)", ip.String(), ipv4Net.IP.String())
		CBLogger.Errorf(errStr)
		CBLogger.Debug("End.........")
		return peerIPv4CIDR, peerIPAddress, errors.New(errStr)
	}
	CBLogger.Debug("End.........")

	return peerIPv4CIDR, peerIPAddress, nil
}

func allocatePeer(cladnetID string, hostID string, hostName string, hostIPv4CIDR string, hostIP string, hostPublicIP string, etcdClient *clientv3.Client) model.Peer {
	CBLogger.Debug("Start.........")

	// Create a key of the CLADNet specification
	keyCLADNetSpecificationOfCLADNet := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + cladnetID)

	// Get the CLADNet specification to check IPv4AddressSpace
	cladnetIpv4AddressSpace, err := getIpv4AddressSpace(etcdClient, keyCLADNetSpecificationOfCLADNet)
	if err != nil {
		CBLogger.Error(err)
	}

	// Create a key of peers in a Cloud Adaptive Network
	keyPeersInCLADNet := fmt.Sprint(etcdkey.Peer + "/" + cladnetID)

	// Get the number of peers
	CBLogger.Debugf("Get with prefix - %v", keyPeersInCLADNet)
	resp, respErr := etcdClient.Get(context.TODO(), keyPeersInCLADNet, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if respErr != nil {
		CBLogger.Error(respErr)
	}
	CBLogger.Tracef("GetResponse: %#v", resp)

	fields := createFieldsForResponseSizes(*resp)
	CBLogger.WithFields(fields).Tracef("GetResponse size (bytes)")

	state := netstate.Configuring
	peerIPv4CIDR, peerIPAddress, err := assignIPAddressToPeer(cladnetIpv4AddressSpace, uint32(resp.Count+2))
	if err != nil {
		CBLogger.Error(err)
		state = netstate.Failed

	}

	// "0.0.0.0" will be assigned in error case
	peer := model.Peer{
		CladnetID:           cladnetID,
		HostID:              hostID,
		HostName:            hostName,
		HostPrivateIPv4CIDR: hostIPv4CIDR,
		HostPrivateIP:       hostIP,
		HostPublicIP:        hostPublicIP,
		IPv4CIDR:            peerIPv4CIDR,
		IP:                  peerIPAddress,
		State:               state,
	}

	CBLogger.Debug("End.........")
	return peer
}

func main() {

	CBLogger.Debug("Start.........")

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
	go watchHostNetworkInformation(&wg, etcdClient)

	// wg.Add(1)
	// go watchPeer(&wg, etcdClient)

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
