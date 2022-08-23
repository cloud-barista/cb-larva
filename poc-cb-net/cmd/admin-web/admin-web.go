package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go"
	cbnet "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network"
	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/etcd-key"
	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config model.Config
var loggerNamePrefix = "admin-web"
var adminWebID string

// gRPC client
var cladnetClient pb.CloudAdaptiveNetworkServiceClient
var systemManagementClient pb.SystemManagementServiceClient

func init() {
	fmt.Println("\nStart......... init() of admin-web.go")

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

	// Generate a temporary ID for cb-network admin-web (only one admin-web works)
	guid := xid.New()
	adminWebID = guid.String()

	loggerName := fmt.Sprintf("%s-%s", loggerNamePrefix, adminWebID)

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

	fmt.Println("End......... init() of admin-web.go")
	fmt.Println("")
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

// WebsocketHandler represents a handler to watch and send networking rules to admin-web frontend.
func WebsocketHandler(c echo.Context) error {
	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	connectionPool.Lock()
	connectionPool.connections[ws] = struct{}{}

	defer func(connection *websocket.Conn) {
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

	defer func() {
		errClose := etcdClient.Close()
		if errClose != nil {
			CBLogger.Fatal("Can't close the etcd client", errClose)
		}
	}()

	CBLogger.Infoln("The etcdClient is connected.")

	// Get the existing both the networking rule and the specification of the CLADNet
	errInitData := getExistingNetworkInfo(etcdClient)
	if errInitData != nil {
		CBLogger.Errorf("getExistingNetworkInfo() error: %v", errInitData)
	}

	for {
		// Read
		_, msgRead, err := ws.ReadMessage()
		if err != nil {
			CBLogger.Error(err)
			return err
		}
		CBLogger.Tracef("Message Read: %s", msgRead)

		var message model.WebsocketMessageFrame
		errUnmarshalDataFrame := json.Unmarshal(msgRead, &message)
		if errUnmarshalDataFrame != nil {
			CBLogger.Error(errUnmarshalDataFrame)
		}

		switch message.Type {
		case "create-cladnet":
			handleCreateCLADNet(etcdClient, []byte(message.Text))

		case "test-cladnet":
			handleTestCLADNet(etcdClient, message.Text)

		case "control-cladnet":
			handleControlCLADNet(etcdClient, message.Text)

		default:

		}

	}
}

func handleCreateCLADNet(etcdClient *clientv3.Client, responseText []byte) {
	CBLogger.Debug("Start.........")

	// Unmarshal the specification of Cloud Adaptive Network (CLADNet)
	// :IPv4 Network, Description

	var tempSpec model.CLADNetSpecification
	errUnmarshal := json.Unmarshal(responseText, &tempSpec)
	if errUnmarshal != nil {
		CBLogger.Errorln("Failed to parse CLADNetSpecification:", errUnmarshal)
	}
	CBLogger.Trace("TempSpec:", tempSpec)

	cladnetSpec := &pb.CLADNetSpecification{
		CladnetId:        tempSpec.CladnetID,
		Name:             tempSpec.Name,
		Ipv4AddressSpace: tempSpec.Ipv4AddressSpace,
		Description:      tempSpec.Description}

	CBLogger.Tracef("The requested CLADNet specification: %v", cladnetSpec.String())

	// Request to create CLADNet
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := cladnetClient.CreateCLADNet(ctx, cladnetSpec)

	if err != nil {
		log.Fatalf("could not request: %v", err)
	}

	// Response for the request
	CBLogger.Tracef("Response: %v", r)

	CBLogger.Debug("End.........")
}

func handleTestCLADNet(etcdClient *clientv3.Client, responseText string) {
	CBLogger.Debug("Start.........")

	cladnetID := gjson.Get(responseText, "cladnetId").String()
	testType := gjson.Get(responseText, "testType").String()
	testSpec := gjson.Get(responseText, "testSpec").Raw

	CBLogger.Tracef("CLADNet ID: %#v", cladnetID)
	CBLogger.Tracef("testType: %#v", testType)
	CBLogger.Tracef("testSpec: %#v", testSpec)

	// Request to create CLADNet
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	testRequest := &pb.TestRequest{
		CladnetId: cladnetID,
		TestType:  pb.TestType(pb.TestType_value[testType]),
		TestSpec:  testSpec,
	}

	testResponse, err := systemManagementClient.TestCloudAdaptiveNetwork(ctx, testRequest)
	if err != nil {
		CBLogger.Error(err)
	}

	CBLogger.Debugf("Command result: %#v", testResponse)

	CBLogger.Debug("End.........")
}

func handleControlCLADNet(etcdClient *clientv3.Client, responseText string) {
	CBLogger.Debug("Start.........")

	cladnetID := gjson.Get(responseText, "cladnetId").String()
	commandType := gjson.Get(responseText, "commandType").String()

	CBLogger.Tracef("CLADNet ID: %#v", cladnetID)
	CBLogger.Tracef("commandType: %#v", commandType)

	// Request to create CLADNet
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	controlRequest := &pb.ControlRequest{
		CladnetId:   cladnetID,
		CommandType: pb.CommandType(pb.CommandType_value[commandType]),
	}

	controlResponse, err := systemManagementClient.ControlCloudAdaptiveNetwork(ctx, controlRequest)
	if err != nil {
		CBLogger.Error(err)
	}

	CBLogger.Debugf("Command result: %#v", controlResponse)

	CBLogger.Debug("End.........")
}

func getExistingNetworkInfo(etcdClient *clientv3.Client) error {

	// Get all peers
	CBLogger.Debugf("Get - %v", etcdkey.Peer)
	resp, etcdErr := etcdClient.Get(context.Background(), etcdkey.Peer, clientv3.WithPrefix())
	CBLogger.Tracef("etcdResp: %v", resp)
	if etcdErr != nil {
		CBLogger.Error(etcdErr)
	}

	for _, kv := range resp.Kvs {
		CBLogger.Tracef("CLADNet ID: %v", kv.Key)
		CBLogger.Tracef("A peer of the CLADNet: %v", kv.Value)
		CBLogger.Debug("Send a peer of CLADNet to admin-web frontend")

		// Build the response bytes of a networking rule
		responseBytes := buildResponseBytes("peer", string(kv.Value))

		// Send the networking rule to the front-end
		CBLogger.Debug("Send the networking rule to admin-web frontend")
		sendErr := sendMessageToAllPool(responseBytes)
		if sendErr != nil {
			CBLogger.Error(sendErr)
		}

	}

	if resp.Count == 0 {
		CBLogger.Debug("no networking rule of CLADNet exists")
	}

	// Get the specification of the CLADNet
	CBLogger.Debugf("Get - %v", etcdkey.CLADNetSpecification)
	respMultiSpec, err := etcdClient.Get(context.Background(), etcdkey.CLADNetSpecification, clientv3.WithPrefix())
	if err != nil {
		CBLogger.Error(err)
		return err
	}

	if len(respMultiSpec.Kvs) != 0 {
		var cladnetSpecificationList []string
		for _, Spec := range respMultiSpec.Kvs {
			cladnetSpecificationList = append(cladnetSpecificationList, string(Spec.Value))
		}

		CBLogger.Tracef("CladnetSpecificationList: %v", cladnetSpecificationList)

		// Build response JSON
		var buf bytes.Buffer
		text := strings.Join(cladnetSpecificationList, ",")
		buf.WriteString("[")
		buf.WriteString(text)
		buf.WriteString("]")

		// Build the response bytes of a CLADNet list
		responseBytes := buildResponseBytes("CLADNetList", buf.String())

		// Send the CLADNet list to the front-end
		CBLogger.Debug("Send the CLADNet list to admin-web frontend")
		sendErr := sendMessageToAllPool(responseBytes)
		if sendErr != nil {
			CBLogger.Error(sendErr)
		}
	}
	return nil
}

func buildResponseBytes(responseType string, responseText string) []byte {
	CBLogger.Debug("Start.........")
	var response model.WebsocketMessageFrame
	response.Type = responseType
	response.Text = responseText

	CBLogger.Tracef("ResponseStr: %#v", response)
	responseBytes, _ := json.Marshal(response)
	CBLogger.Debug("End.........")
	return responseBytes
}

// func sendResponseText(ws *websocket.Conn, responseType string, responseText string) error {
// 	CBLogger.Debug("Start.........")
// 	responseBytes := buildResponseBytes(responseType, responseText)

// 	// Response to the front-end
// 	errWriteJSON := ws.WriteMessage(websocket.TextMessage, responseBytes)
// 	if errWriteJSON != nil {
// 		CBLogger.Error(errWriteJSON)
// 		return errWriteJSON
// 	}
// 	CBLogger.Debug("End.........")
// 	return nil
// }

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
func RunEchoServer(wg *sync.WaitGroup, config model.Config) {
	defer wg.Done()

	// Set web assets path to the current directory (usually for the production)
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	CBLogger.Tracef("exePath: %v", exePath)
	webPath := filepath.Join(exePath, "web")

	indexPath := filepath.Join(webPath, "public", "index.html")
	CBLogger.Tracef("indexPath: %v", indexPath)
	if !file.Exists(indexPath) {
		// Set web assets path to the project directory (usually for the development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		webPath = filepath.Join(projectPath, "poc-cb-net", "web")
	}

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
			"websocket_host": "http://" + config.AdminWeb.Host + ":" + config.AdminWeb.Port,
		})
	})

	// Render
	e.GET("/ws", WebsocketHandler)

	CBNet := cbnet.New("temp", "")

	adminWebURL := fmt.Sprintf("http://%s:%s", config.AdminWeb.Host, config.AdminWeb.Port)
	localhostURL := fmt.Sprintf("http://%s:%s", "localhost", config.AdminWeb.Port)
	publicAccessURL := fmt.Sprintf("http://%s:%s", CBNet.HostPublicIP, config.AdminWeb.Port)

	fmt.Println("")
	fmt.Printf("\033[1;36m%s\033[0m\n", "[The cb-network admin-web URLs]")
	fmt.Printf("\033[1;36m ==> %s (set in 'config.yaml')\033[0m\n", adminWebURL)
	fmt.Printf("\033[1;36m ==> %s (may need to check firewall rule)\033[0m\n", publicAccessURL)
	if config.AdminWeb.Host != "localhost" {
		fmt.Printf("\033[1;36m ==> %s\033[0m\n", localhostURL)
	}

	fmt.Println("")

	e.Logger.Fatal(e.Start(":" + config.AdminWeb.Port))
	CBLogger.Debug("End.........")
}

func watchPeer(wg *sync.WaitGroup, etcdClient *clientv3.Client) {
	defer wg.Done()

	// Watch "/registry/cloud-adaptive-network/peer"
	CBLogger.Debugf("Start to watch \"%v\"", etcdkey.Peer)
	watchChan1 := etcdClient.Watch(context.Background(), etcdkey.Peer, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			switch event.Type {
			case mvccpb.PUT: // The watched value has changed.
				CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

				peer := event.Kv.Value
				CBLogger.Tracef("A peer of CLADNet: %v", string(peer))

				// Build the response bytes of the networking rule
				responseBytes := buildResponseBytes("peer", string(peer))

				// Send the networking rule to the front-end
				CBLogger.Debug("Send the networking rule to admin-web frontend")
				sendErr := sendMessageToAllPool(responseBytes)
				if sendErr != nil {
					CBLogger.Error(sendErr)
				}

			case mvccpb.DELETE: // The watched key has been deleted.
				CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			default:
				CBLogger.Errorf("Known event (%s), Key(%q), Value(%q)", event.Type, event.Kv.Key, event.Kv.Value)
			}
		}
	}
	CBLogger.Debugf("End to watch \"%v\"", etcdkey.Peer)
}

func watchCLADNetSpecification(wg *sync.WaitGroup, etcdClient *clientv3.Client) {
	defer wg.Done()

	// It doesn't work for the time being
	// Watch "/registry/cloud-adaptive-network/cladnet-specification"
	CBLogger.Debugf("Start to watch \"%v\"", etcdkey.CLADNetSpecification)
	watchChan1 := etcdClient.Watch(context.Background(), etcdkey.CLADNetSpecification, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			CBLogger.Tracef("Updated CLADNet: %v", string(event.Kv.Value))

			// Get the specification of the CLADNet
			CBLogger.Debugf("Get - %v", etcdkey.CLADNetSpecification)
			respMultiSpec, err := etcdClient.Get(context.Background(), etcdkey.CLADNetSpecification, clientv3.WithPrefix())
			if err != nil {
				CBLogger.Error(err)
			}

			if len(respMultiSpec.Kvs) != 0 {
				var cladnetSpecificationList []string
				for _, Spec := range respMultiSpec.Kvs {
					cladnetSpecificationList = append(cladnetSpecificationList, string(Spec.Value))
				}

				CBLogger.Tracef("cladnetSpecificationList: %v", cladnetSpecificationList)

				// Build response JSON
				var buf bytes.Buffer
				text := strings.Join(cladnetSpecificationList, ",")
				buf.WriteString("[")
				buf.WriteString(text)
				buf.WriteString("]")

				// Build the response bytes of a CLADNet list
				responseBytes := buildResponseBytes("CLADNetList", buf.String())

				// Send the CLADNet list to the front-end
				CBLogger.Debug("Send the CLADNet list to admin-web frontend")
				sendErr := sendMessageToAllPool(responseBytes)
				if sendErr != nil {
					CBLogger.Error(sendErr)
				}
			}
		}
	}
	CBLogger.Debugf("End to watch \"%v\"", etcdkey.CLADNetSpecification)
}

func watchStatusInformation(wg *sync.WaitGroup, etcdClient *clientv3.Client) {
	defer wg.Done()

	// Watch "/registry/cloud-adaptive-network/status/information/{cladnet-id}/{host-id}"
	CBLogger.Debugf("Start to watch \"%v\"", etcdkey.StatusInformation)
	watchChan1 := etcdClient.Watch(context.Background(), etcdkey.StatusInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			slicedKeys := strings.Split(string(event.Kv.Key), "/")
			parsedHostID := slicedKeys[len(slicedKeys)-1]
			CBLogger.Tracef("ParsedHostID: %v", parsedHostID)

			status := string(event.Kv.Value)
			CBLogger.Tracef("The status: %v", status)

			// Build the response bytes of the networking rule
			responseBytes := buildResponseBytes("NetworkStatus", status)

			// Send the networking rule to the front-end
			CBLogger.Debug("Send the status to admin-web frontend")
			sendErr := sendMessageToAllPool(responseBytes)
			if sendErr != nil {
				CBLogger.Error(sendErr)
			}
		}
	}
	CBLogger.Debugf("End to watch \"%v\"", etcdkey.Status)
}

func main() {
	CBLogger.Debug("Start cb-network admin-web .........")

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	// etcd section
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

	// gRPC client section
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	grpcConn, err := grpc.Dial(config.Service.Endpoint, options...)
	if err != nil {
		log.Fatalf("Cannot connect to gRPC Server: %v", err)
	}
	defer func() {
		grpcConnErr := grpcConn.Close()
		if grpcConnErr != nil {
			CBLogger.Fatal("Can't close the gRPC conn", grpcConnErr)
		}
	}()

	CBLogger.Infoln("The gRPC client is connected.")

	cladnetClient = pb.NewCloudAdaptiveNetworkServiceClient(grpcConn)
	systemManagementClient = pb.NewSystemManagementServiceClient(grpcConn)

	// watch section
	wg.Add(1)
	go watchPeer(&wg, etcdClient)

	wg.Add(1)
	go watchCLADNetSpecification(&wg, etcdClient)

	wg.Add(1)
	go watchStatusInformation(&wg, etcdClient)

	wg.Add(1)
	go RunEchoServer(&wg, config)

	// Waiting for all goroutines to finish
	CBLogger.Info("Waiting for all goroutines to finish")

	wg.Wait()

	CBLogger.Debug("End cb-network admin-web .........")
}
