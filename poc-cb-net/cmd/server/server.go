package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/data-objects"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/internal/etcd-key"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/client/v3"
	"html/template"
	"io"
	"math/big"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var dscp *cbnet.DynamicSubnetConfigurator

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config dataobjects.Config

func init() {
	// cblog is a global variable.
	logConfPath := filepath.Join("..", "..", "configs", "log_conf.yaml")
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)

	// Load config
	configPath := filepath.Join("..", "..", "configs", "config.yaml")
	config, _ = dataobjects.LoadConfig(configPath)

}

var (
	upgrader = websocket.Upgrader{}
)

var connectionPool = struct {
	sync.RWMutex
	connections map[*websocket.Conn]struct{}
}{
	connections: make(map[*websocket.Conn]struct{}),
}

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// WebsocketHandler represents a handler to watch and send networking rules to AdminWeb frontend.
func WebsocketHandler(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	connectionPool.Lock()
	connectionPool.connections[ws] = struct{}{}

	defer func(connection *websocket.Conn){
		connectionPool.Lock()
		delete(connectionPool.connections, connection)
		connectionPool.Unlock()
	}(ws)

	connectionPool.Unlock()

	// etcd Section
	etcdClient, err := clientv3.New(clientv3.Config{
		Endpoints:   config.ETCD.Endpoints,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		CBLogger.Fatal(err)
	}

	CBLogger.Infoln("The etcdClient is connected.")

	CBLogger.Infof("Get - %v", etcdkey.NetworkingRule)
	resp, etcdErr := etcdClient.Get(context.Background(), etcdkey.NetworkingRule, clientv3.WithPrefix())
	if etcdErr != nil {
		CBLogger.Error(etcdErr)
	}
	CBLogger.Tracef("etcdResp: %v\n", resp)

	CBLogger.Debug("Send CLADNet information to AdminWeb frontend")
	cladnet := resp.Kvs[0].Value
	sendErr := sendMessageToAllPool(cladnet)
	if sendErr != nil {
		CBLogger.Error(sendErr)
	}

	errClose := etcdClient.Close()
	if errClose != nil {
		CBLogger.Fatal("Can't close the etcd client", errClose)
	}

	for {
		// Read
		_, msgRead, err := ws.ReadMessage()
		if err != nil {
			CBLogger.Error(err)
			return err
		}
		CBLogger.Tracef("Message Read: %s\n", msgRead)

		//// Write
		//msgToBeWritten := []byte("Hello, Client")
		//err = sendMessageToAllPool(msgToBeWritten)
		//if err != nil {
		//	return err
		//}
		//fmt.Printf("Message Written: %s\n", msgToBeWritten)
	}
}

func sendMessageToAllPool(message []byte) error {
	connectionPool.RLock()
	defer connectionPool.RUnlock()
	for connection := range connectionPool.connections {
		err := connection.WriteMessage(websocket.TextMessage, message)
		if err != nil {
			return err
		}
	}
	return nil
}

// RunEchoServer represents a function to run echo server.
func RunEchoServer(config dataobjects.Config) {

	webPath := "../../web"
	CBLogger.Debug("Start.........")
	e := echo.New()

	e.Static("/", webPath+"/assets")
	e.Static("/js", webPath+"/assets/js")
	e.Static("/css", webPath+"/assets/css")
	e.Static("/introspect", webPath+"/assets/introspect")

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob(webPath + "/public/*.html")),
	}
	e.Renderer = renderer

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", map[string]interface{}{
			"websocket_host":  "http://" + config.AdminWeb.Host + ":" + config.AdminWeb.Port,
		})
	})

	// Render
	e.GET("/ws", WebsocketHandler)

	e.Logger.Fatal(e.Start(":" + config.AdminWeb.Port))
	CBLogger.Debug("End.........")
}

func watchNetworkingRule(etcdClient *clientv3.Client){
	CBLogger.Infof("The etcdClient is watching \"%v\"\n", etcdkey.NetworkingRule)
	watchChan1 := etcdClient.Watch(context.Background(), etcdkey.NetworkingRule, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q\n", event.Type, event.Kv.Key, event.Kv.Value)
			slicedKeys := strings.Split(string(event.Kv.Key), "/")
			parsedHostID := slicedKeys[len(slicedKeys)-1]
			CBLogger.Tracef("ParsedHostID: %v\n", parsedHostID)

			CBLogger.Info("Send the networking rule to AdminWeb frontend")
			sendErr := sendMessageToAllPool(event.Kv.Value)
			if sendErr != nil {
				CBLogger.Error(sendErr)
			}
			CBLogger.Info("Done to send")
			CBLogger.Tracef("Message Written: %s\n", event.Kv.Value)
		}
	}
}

func watchConfigurationInformation(etcdClient *clientv3.Client) {
	// It doesn't work for the time being
	CBLogger.Infof("The etcdClient is watching \"%v\"\n", etcdkey.ConfigurationInformation)
	watchChan1 := etcdClient.Watch(context.Background(), etcdkey.ConfigurationInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q\n", event.Type, event.Kv.Key, event.Kv.Value)
			//slicedKeys := strings.Split(string(event.Kv.Key), "/")
			//for _, value := range slicedKeys {
			//	fmt.Println(value)
			//}
		}
	}
}

func watchHostNetworkInformation(etcdClient *clientv3.Client) {
	CBLogger.Infof("The etcdClient is watching \"%v\"\n", etcdkey.HostNetworkInformation)
	watchChan2 := etcdClient.Watch(context.Background(), etcdkey.HostNetworkInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan2 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q\n", event.Type, event.Kv.Key, event.Kv.Value)

			var hostNetworkInformation dataobjects.HostNetworkInformation
			err := json.Unmarshal(event.Kv.Value, &hostNetworkInformation)
			if err != nil {
				CBLogger.Panic(err)
			}

			// Parse groupId from the Key
			slicedKeys := strings.Split(string(event.Kv.Key), "/")
			parsedHostID := slicedKeys[len(slicedKeys)-1]
			CBLogger.Tracef("ParsedHostId: %v\n", parsedHostID)
			parsedGroupID := slicedKeys[len(slicedKeys)-2]
			CBLogger.Tracef("ParsedGroupId: %v\n", parsedGroupID)

			// [TBD] Get CLADNet configuration information of a group
			// [TBD] Get CIDRBlock

			// The below CIDRBlock is used temporally.
			cladNetCIDRBlock := "192.168.10.0/23"

			// Get Networking rule of the group
			keyNetworkingRuleOfGroup := fmt.Sprint(etcdkey.NetworkingRule + "/" + parsedGroupID)
			CBLogger.Tracef("Key: %v\n", keyNetworkingRuleOfGroup)
			respRule, respRuleErr := etcdClient.Get(context.Background(), keyNetworkingRuleOfGroup)
			if respRuleErr != nil {
				CBLogger.Error(respRuleErr)
			}

			var tempRule dataobjects.NetworkingRule

			CBLogger.Tracef("RespRule.Kvs: %v\n", respRule.Kvs)
			if len(respRule.Kvs) != 0 {
				errUnmarshal := json.Unmarshal(respRule.Kvs[0].Value, &tempRule)
				if errUnmarshal != nil {
					CBLogger.Panic(errUnmarshal)
				}
			}

			CBLogger.Tracef("TempRule: %v\n", tempRule)

			// !!! Should compare all value
			if tempRule.Contain(parsedHostID) {
				tempRule.UpdateRule(parsedHostID, "", "", hostNetworkInformation.PublicIP)
			} else {

				// Get IPNet struct from string
				_, ipv4Net, errParseCIDR := net.ParseCIDR(cladNetCIDRBlock)
				if errParseCIDR != nil {
					CBLogger.Fatal(errParseCIDR)
				}

				// Get NetworkAddress(uint32) (The first IP address of this network)
				firstIP := binary.BigEndian.Uint32(ipv4Net.IP)
				CBLogger.Trace(firstIP)

				// Get Subnet Mask(uint32) from IPNet struct
				subnetMask := binary.BigEndian.Uint32(ipv4Net.Mask)
				CBLogger.Trace(subnetMask)

				// Get BroadcastAddress(uint32) (The last IP address of this network)
				lastIP := (firstIP & subnetMask) | (subnetMask ^ 0xffffffff)
				CBLogger.Trace(lastIP)

				// Get a candidate of IP Address in serial order to assign IP Address to a client
				// Exclude Network Address, Broadcast Address, Gateway Address
				ipCandidate := firstIP + uint32(len(tempRule.HostID)+2)

				// Create IP address of type net.IP. IPv4 is 4 bytes, IPv6 is 16 bytes.
				var ip = make(net.IP, 4)
				if ipCandidate < lastIP-1 {
					binary.BigEndian.PutUint32(ip, ipCandidate)
				} else {
					CBLogger.Panic("This IP is out of range of the network")
				}

				// Get CIDR Prefix
				cidrPrefix, _ := ipv4Net.Mask.Size()
				// Create Host IP CIDR Block
				hostIPCIDRBlock := fmt.Sprint(ip, "/", cidrPrefix)
				// To string IP Address
				hostIPAddress := fmt.Sprint(ip)

				// Append {HostID, HostIPCIDRBlock, HostIPAddress, PublicIP} to a group's Networking Rule
				tempRule.AppendRule(parsedHostID, hostIPCIDRBlock, hostIPAddress, hostNetworkInformation.PublicIP)
			}

			CBLogger.Debugf("Put \"%v\"\n", keyNetworkingRuleOfGroup)

			doc, _ := json.Marshal(tempRule)

			//requestTimeout := 10 * time.Second
			//ctx, _ := context.WithTimeout(context.Background(), requestTimeout)
			_, err = etcdClient.Put(context.Background(), keyNetworkingRuleOfGroup, string(doc))
			if err != nil {
				CBLogger.Panic(err)
			}
		}
	}

}

func main() {
	CBLogger.Debug("Start.........")

	// Random number to avoid MQTT client HostID duplication
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Random number: %d\t", n)

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

	go watchNetworkingRule(etcdClient)

	go watchConfigurationInformation(etcdClient)

	go watchHostNetworkInformation(etcdClient)

	go RunEchoServer(config)

	// Block to stop this program
	CBLogger.Info("Press the Enter Key to stop anytime")
	fmt.Scanln()

	CBLogger.Debug("End.........")
}
