package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal"
	dataobjects "github.com/cloud-barista/cb-larva/poc-cb-net/internal/data-objects"
	cblog "github.com/cloud-barista/cb-log"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/labstack/echo"
	"github.com/sirupsen/logrus"
	"html/template"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
)

var dscp *internal.DynamicSubnetConfigurator
var CBLogger *logrus.Logger

func init() {
	// cblog is a global variable.
	configPath := filepath.Join("..", "..", "configs", "log_conf.yaml")
	CBLogger = cblog.GetLoggerWithConfigPath("cb-network", configPath)
}

// Define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	CBLogger.Info("Start.........")
	CBLogger.Debugf("Received TOPIC : %s\n", msg.Topic())
	CBLogger.Debugf("MSG: %s\n", msg.Payload())

	if msg.Topic() == "cb-net/vm-network-information" {

		// Unmarshal the VM network information
		var vmNetworkInfo dataobjects.VMNetworkInformation

		err := json.Unmarshal(msg.Payload(), &vmNetworkInfo)
		if err != nil {
			panic(err)
		}
		CBLogger.Debug("Unmarshalled JSON")
		CBLogger.Debug(vmNetworkInfo)

		prettyJSON, _ := json.MarshalIndent(vmNetworkInfo, "", "\t")
		CBLogger.Debug("Pretty JSON")
		CBLogger.Debug(string(prettyJSON))

		// Update CBNetworking Rule
		dscp.UpdateCBNetworkingRule(vmNetworkInfo)

		doc, _ := json.Marshal(dscp.NetworkingRule)

		CBLogger.Debug("Publish topic, cb-net/networking-rule")
		client.Publish("cb-net/networking-rule", 0, false, doc)

		CBLogger.Info("End.........")
	}
}

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func RunEchoServer(config dataobjects.ConfigMQTTBroker) {
	CBLogger.Info("Start.........")
	e := echo.New()

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("public/*.html")),
	}
	e.Renderer = renderer

	e.Static("/", "assets")
	e.Static("/js", "assets/js")
	e.Static("/css", "assets/css")
	e.Static("/introspect", "assets/introspect")

	e.GET("/", func(c echo.Context) error {
		return c.Render(http.StatusOK, "index.html", map[string]interface{}{
			"host": config.MQTTBrokerHost,
			"port": config.MQTTBrokerPortForWebsocket,
		})
	})

	e.Logger.Fatal(e.Start(":8000"))
	CBLogger.Info("End.........")
}

func main() {
	CBLogger.Info("Start.........")

	// Random number to avoid MQTT client ID duplication
	n, err := rand.Int(rand.Reader, big.NewInt(100))
	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Debugf("Random number: %d\t", n)

	// Create DynamicSubnetConfigurator instance
	dscp = internal.NewDynamicSubnetConfigurator()

	// Load a config of MQTTBroker
	config, _ := dataobjects.LoadConfigMQTTBroker()
	// Create a endpoint link of MQTTBroker
	server := "tcp://" + config.MQTTBrokerHost + ":" + config.MQTTBrokerPort

	// Create a ClientOptions struct setting the broker address, clientID, turn
	// off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker(server)
	opts.SetClientID(fmt.Sprint("cb-net-agent-", n))
	opts.SetDefaultPublishHandler(f)

	// Create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		CBLogger.Error(token.Error())
	}

	// Subscribe to the topic /go-mqtt/sample and request messages to be delivered
	// at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe("cb-net/vm-network-information", 0, nil); token.Wait() && token.Error() != nil {
		CBLogger.Error(token.Error())
		os.Exit(1)
	}

	go RunEchoServer(config)

	// Block to stop this program
	fmt.Println("Press the Enter Key to stop anytime")
	fmt.Scanln()

	//Unsubscribe from /cb-net/vm-network-information"
	if token := c.Unsubscribe("cb-net/vm-network-information"); token.Wait() && token.Error() != nil {
		CBLogger.Error(token.Error())
		os.Exit(1)
	}

	// Disconnect MQTT Client
	c.Disconnect(250)
	CBLogger.Info("End.........")
}
