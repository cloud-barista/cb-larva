package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	cbnet "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/data-objects"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/internal/etcd-key"
	file "github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var dscp *cbnet.DynamicSubnetConfigurator

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config dataobjects.Config

func init() {
	fmt.Println("Start......... init() of controller.go")
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
	fmt.Println("End......... init() of controller.go")
}

// func watchConfigurationInformation(wg *sync.WaitGroup, etcdClient *clientv3.Client) {
// 	defer wg.Done()

// 	// It doesn't work for the time being
// 	// Watch "/registry/cloud-adaptive-network/configuration-information"
// 	CBLogger.Debugf("Start to watch \"%v\"", etcdkey.ConfigurationInformation)
// 	watchChan1 := etcdClient.Watch(context.Background(), etcdkey.ConfigurationInformation, clientv3.WithPrefix())
// 	for watchResponse := range watchChan1 {
// 		for _, event := range watchResponse.Events {
// 			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
// 			//slicedKeys := strings.Split(string(event.Kv.Key), "/")
// 			//for _, value := range slicedKeys {
// 			//	fmt.Println(value)
// 			//}
// 		}
// 	}
// 	CBLogger.Debugf("End to watch \"%v\"", etcdkey.ConfigurationInformation)
// }

func watchHostNetworkInformation(wg *sync.WaitGroup, etcdClient *clientv3.Client) {
	defer wg.Done()
	// Watch "/registry/cloud-adaptive-network/host-network-information"
	CBLogger.Debugf("Start to watch \"%v\"", etcdkey.HostNetworkInformation)
	watchChan2 := etcdClient.Watch(context.Background(), etcdkey.HostNetworkInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan2 {
		for _, event := range watchResponse.Events {
			switch event.Type {
			case mvccpb.PUT: // The watched value has changed.
				CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
				var hostNetworkInformation dataobjects.HostNetworkInformation
				err := json.Unmarshal(event.Kv.Value, &hostNetworkInformation)
				if err != nil {
					CBLogger.Error(err)
				}

				// Parse HostID and CLADNetID from the Key
				slicedKeys := strings.Split(string(event.Kv.Key), "/")
				parsedHostID := slicedKeys[len(slicedKeys)-1]
				CBLogger.Tracef("ParsedHostId: %v", parsedHostID)
				parsedCLADNetID := slicedKeys[len(slicedKeys)-2]
				CBLogger.Tracef("ParsedCLADNetId: %v", parsedCLADNetID)

				// Get the configuration information of the CLADNet
				keyConfigurationInformationOfCLADNet := fmt.Sprint(etcdkey.ConfigurationInformation + "/" + parsedCLADNetID)
				respConfInfo, errConfInfo := etcdClient.Get(context.Background(), keyConfigurationInformationOfCLADNet)
				if errConfInfo != nil {
					CBLogger.Error(errConfInfo)
				}

				var tempConfInfo dataobjects.CLADNetConfigurationInformation
				var cladNetCIDRBlock string

				// Unmarshal the configuration information of the CLADNet if exists
				CBLogger.Tracef("RespRule.Kvs: %v", respConfInfo.Kvs)
				if len(respConfInfo.Kvs) != 0 {
					errUnmarshal := json.Unmarshal(respConfInfo.Kvs[0].Value, &tempConfInfo)
					if errUnmarshal != nil {
						CBLogger.Error(errUnmarshal)
					}
					CBLogger.Tracef("TempConfInfo: %v", tempConfInfo)
					// Get a network CIDR block of CLADNet
					cladNetCIDRBlock = tempConfInfo.CIDRBlock
				} else {
					// [To be updated] Update the assignment logic of the default network CIDR block
					cladNetCIDRBlock = "192.168.119.0/24"
				}

				// Get Networking rule of the CLADNet
				keyNetworkingRuleOfCLADNet := fmt.Sprint(etcdkey.NetworkingRule + "/" + parsedCLADNetID)
				CBLogger.Tracef("Key: %v", keyNetworkingRuleOfCLADNet)
				respRule, respRuleErr := etcdClient.Get(context.Background(), keyNetworkingRuleOfCLADNet)
				if respRuleErr != nil {
					CBLogger.Error(respRuleErr)
				}

				var tempRule dataobjects.NetworkingRule

				// Unmarshal the existing networking rule of the CLADNet if exists
				CBLogger.Tracef("RespRule.Kvs: %v", respRule.Kvs)
				if len(respRule.Kvs) != 0 {
					errUnmarshal := json.Unmarshal(respRule.Kvs[0].Value, &tempRule)
					if errUnmarshal != nil {
						CBLogger.Error(errUnmarshal)
					}
				} else {
					tempRule.CLADNetID = parsedCLADNetID
				}

				CBLogger.Tracef("TempRule: %v", tempRule)

				// Update the existing networking rule
				// If not, assign an IP address to a host and append it to networking rule
				if tempRule.Contain(parsedHostID) {
					// [To be updated] All values should be compared on purpose.
					tempRule.UpdateRule(parsedHostID, "", "", hostNetworkInformation.PublicIP)
				} else {

					// Assign a candidate of IP Address in serial order to a host
					// Exclude Network Address, Broadcast Address, Gateway Address
					hostIPCIDRBlock, hostIPAddress := assignIPAddressToHost(cladNetCIDRBlock, uint32(len(tempRule.HostID)+2))

					// Append {HostID, HostIPCIDRBlock, HostIPAddress, PublicIP} to a CLADNet's Networking Rule
					tempRule.AppendRule(parsedHostID, hostIPCIDRBlock, hostIPAddress, hostNetworkInformation.PublicIP)
				}

				CBLogger.Debugf("Put \"%v\"", keyNetworkingRuleOfCLADNet)

				doc, _ := json.Marshal(tempRule)

				//requestTimeout := 10 * time.Second
				//ctx, _ := context.WithTimeout(context.Background(), requestTimeout)
				_, err = etcdClient.Put(context.Background(), keyNetworkingRuleOfCLADNet, string(doc))
				if err != nil {
					CBLogger.Error(err)
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

func assignIPAddressToHost(cidrBlock string, numberOfIPsAssigned uint32) (string, string) {
	// Get IPNet struct from string
	_, ipv4Net, errParseCIDR := net.ParseCIDR(cidrBlock)
	if errParseCIDR != nil {
		CBLogger.Fatal(errParseCIDR)
	}

	// Get NetworkAddress(uint32) (The first IP address of this CLADNet)
	firstIP := binary.BigEndian.Uint32(ipv4Net.IP)
	CBLogger.Trace(firstIP)

	// Get Subnet Mask(uint32) from IPNet struct
	subnetMask := binary.BigEndian.Uint32(ipv4Net.Mask)
	CBLogger.Trace(subnetMask)

	// Get BroadcastAddress(uint32) (The last IP address of this CLADNet)
	lastIP := (firstIP & subnetMask) | (subnetMask ^ 0xffffffff)
	CBLogger.Trace(lastIP)

	// Get a candidate of IP Address in serial order to assign IP Address to a client
	// Exclude Network Address, Broadcast Address, Gateway Address
	ipCandidate := firstIP + numberOfIPsAssigned

	// Create IP address of type net.IP. IPv4 is 4 bytes, IPv6 is 16 bytes.
	var ip = make(net.IP, 4)
	if ipCandidate < lastIP-1 {
		binary.BigEndian.PutUint32(ip, ipCandidate)
	} else {
		CBLogger.Error("This IP is out of range of the CLADNet")
	}

	// Get CIDR Prefix
	cidrPrefix, _ := ipv4Net.Mask.Size()
	// Create Host IP CIDR Block
	hostIPCIDRBlock := fmt.Sprint(ip, "/", cidrPrefix)
	// To string IP Address
	hostIPAddress := fmt.Sprint(ip)

	return hostIPCIDRBlock, hostIPAddress
}

func main() {
	CBLogger.Debug("Start cb-network controller .........")

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	// Create DynamicSubnetConfigurator instance
	dscp = cbnet.NewDynamicSubnetConfigurator()
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

	// Waiting for all goroutines to finish
	CBLogger.Info("Waiting for all goroutines to finish")
	wg.Wait()

	CBLogger.Debug("End cb-network controller .........")
}
