package main

import (
	"bytes"
	"context"
	"encoding/json"
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

	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	cmd "github.com/cloud-barista/cb-larva/poc-cb-net/internal/command"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/internal/etcd-key"
	file "github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config model.Config

// gRPC client
var cladnetClient pb.CloudAdaptiveNetworkServiceClient

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
		case "cladnet-specification":
			handleCLADNetSpecification(etcdClient, []byte(message.Text))

		case "test-specification":
			handleTestSpecification(etcdClient, []byte(message.Text))

		case "control-command":
			handleControlCommand(etcdClient, message.Text)

		default:

		}

	}
}

func handleCLADNetSpecification(etcdClient *clientv3.Client, responseText []byte) {
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
		Id:               tempSpec.ID,
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

func handleTestSpecification(etcdClient *clientv3.Client, responseText []byte) {
	CBLogger.Debug("Start.........")
	var testSpecification model.TestSpecification
	errUnmarshalEvalSpec := json.Unmarshal(responseText, &testSpecification)
	if errUnmarshalEvalSpec != nil {
		CBLogger.Error(errUnmarshalEvalSpec)
	}

	CBLogger.Tracef("Evaluation specification: %v", testSpecification)

	// Get a networking rule of a cloud adaptive network
	keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + testSpecification.CLADNetID)
	resp, err := etcdClient.Get(context.TODO(), keyNetworkingRule, clientv3.WithPrefix())
	if err != nil {
		CBLogger.Error(err)
	}

	for _, kv := range resp.Kvs {

		var peer model.Peer
		CBLogger.Tracef("Key : %v", kv.Key)
		CBLogger.Tracef("The peer: %v", kv.Value)

		err := json.Unmarshal(kv.Value, &peer)
		if err != nil {
			CBLogger.Error(err)
		}

		// Put the evaluation specification of the CLADNet to the etcd
		keyControlCommand := fmt.Sprint(etcdkey.ControlCommand + "/" + peer.CLADNetID + "/" + peer.HostID)
		CBLogger.Tracef("keyControlCommand: \"%s\"", keyControlCommand)

		strStatusTestSpecification, _ := json.Marshal(testSpecification)

		cmdMessageBody := cmd.BuildCommandMessage(cmd.CheckConnectivity, strings.ReplaceAll(string(strStatusTestSpecification), "\"", "\\\""))
		CBLogger.Tracef("%#v", cmdMessageBody)

		//spec := message.Text
		_, err = etcdClient.Put(context.Background(), keyControlCommand, cmdMessageBody)
		if err != nil {
			CBLogger.Error(err)
		}

	}

	CBLogger.Debug("End.........")
}

func handleControlCommand(etcdClient *clientv3.Client, responseText string) {
	CBLogger.Debug("Start.........")

	cladnetID := gjson.Get(responseText, "CLADNetID").String()
	controlCommand := gjson.Get(responseText, "controlCommand").String()
	controlCommandOption := gjson.Get(responseText, "controlCommandOption").String()

	CBLogger.Tracef("CLADNet ID: %v", cladnetID)
	CBLogger.Tracef("controlCommand: %v", controlCommand)
	CBLogger.Tracef("controlCommandOption: %v", controlCommandOption)

	// Get a networking rule of a cloud adaptive network
	keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + cladnetID)
	resp, err := etcdClient.Get(context.TODO(), keyNetworkingRule, clientv3.WithPrefix())
	if err != nil {
		CBLogger.Error(err)
	}

	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		CBLogger.Tracef("Key : %v", key)
		CBLogger.Tracef("The peer: %v", kv.Value)

		var peer model.Peer
		err := json.Unmarshal(kv.Value, &peer)
		if err != nil {
			CBLogger.Error(err)
		}

		// Put the evaluation specification of the CLADNet to the etcd
		keyControlCommand := fmt.Sprint(etcdkey.ControlCommand + "/" + peer.CLADNetID + "/" + peer.HostID)

		cmdMessageBody := cmd.BuildCommandMessage(controlCommand, controlCommandOption)
		CBLogger.Tracef("%#v", cmdMessageBody)
		//spec := message.Text
		_, err = etcdClient.Put(context.Background(), keyControlCommand, cmdMessageBody)
		if err != nil {
			CBLogger.Error(err)
		}

	}

	CBLogger.Debug("End.........")
}

func getExistingNetworkInfo(etcdClient *clientv3.Client) error {

	// Get the networking rule
	CBLogger.Debugf("Get - %v", etcdkey.NetworkingRule)
	resp, etcdErr := etcdClient.Get(context.Background(), etcdkey.NetworkingRule, clientv3.WithPrefix())
	CBLogger.Tracef("etcdResp: %v", resp)
	if etcdErr != nil {
		CBLogger.Error(etcdErr)
	}

	for _, kv := range resp.Kvs {
		CBLogger.Tracef("CLADNet ID: %v", kv.Key)
		CBLogger.Tracef("The networking rule of the CLADNet: %v", kv.Value)
		CBLogger.Debug("Send a networking rule of CLADNet to AdminWeb frontend")

		// Build the response bytes of a networking rule
		responseBytes := buildResponseBytes("peer", string(kv.Value))

		// Send the networking rule to the front-end
		CBLogger.Debug("Send the networking rule to AdminWeb frontend")
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
		CBLogger.Debug("Send the CLADNet list to AdminWeb frontend")
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

	adminWebURL := fmt.Sprintf("The cb-network admin-web URL => http://%s:%s\n", config.AdminWeb.Host, config.AdminWeb.Port)

	fmt.Println("")
	fmt.Printf("\033[1;36m%s\033[0m", adminWebURL)
	fmt.Println("")

	e.Logger.Fatal(e.Start(":" + config.AdminWeb.Port))
	CBLogger.Debug("End.........")
}

func watchNetworkingRule(wg *sync.WaitGroup, etcdClient *clientv3.Client) {
	defer wg.Done()

	// Watch "/registry/cloud-adaptive-network/networking-rule"
	CBLogger.Debugf("Start to watch \"%v\"", etcdkey.NetworkingRule)
	watchChan1 := etcdClient.Watch(context.Background(), etcdkey.NetworkingRule, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			switch event.Type {
			case mvccpb.PUT: // The watched value has changed.
				CBLogger.Tracef("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

				peer := event.Kv.Value
				CBLogger.Tracef("A peer of CLADNet: %v", peer)

				// Build the response bytes of the networking rule
				responseBytes := buildResponseBytes("peer", string(peer))

				// Send the networking rule to the front-end
				CBLogger.Debug("Send the networking rule to AdminWeb frontend")
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
	CBLogger.Debugf("End to watch \"%v\"", etcdkey.NetworkingRule)
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
				CBLogger.Debug("Send the CLADNet list to AdminWeb frontend")
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

			status := event.Kv.Value
			CBLogger.Tracef("The status: %v", status)

			// Build the response bytes of the networking rule
			responseBytes := buildResponseBytes("NetworkStatus", string(status))

			// Send the networking rule to the front-end
			CBLogger.Debug("Send the status to AdminWeb frontend")
			sendErr := sendMessageToAllPool(responseBytes)
			if sendErr != nil {
				CBLogger.Error(sendErr)
			}
		}
	}
	CBLogger.Debugf("End to watch \"%v\"", etcdkey.Status)
}

func main() {
	CBLogger.Debug("Start cb-network controller .........")

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
	grpcConn, err := grpc.Dial(config.GRPC.ServiceEndpoint, grpc.WithInsecure(), grpc.WithBlock())
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

	// watch section
	wg.Add(1)
	go watchNetworkingRule(&wg, etcdClient)

	wg.Add(1)
	go watchCLADNetSpecification(&wg, etcdClient)

	wg.Add(1)
	go watchStatusInformation(&wg, etcdClient)

	wg.Add(1)
	go RunEchoServer(&wg, config)

	// Waiting for all goroutines to finish
	CBLogger.Info("Waiting for all goroutines to finish")

	wg.Wait()

	CBLogger.Debug("End cb-network controller .........")
}
