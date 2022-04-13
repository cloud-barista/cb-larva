package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go"
	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
	cmdtype "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/command-type"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/etcd-key"
	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	nethelper "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/network-helper"
	testtype "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/test-type"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	echoSwagger "github.com/swaggo/echo-swagger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger
var config model.Config
var etcdClient *clientv3.Client

func init() {
	fmt.Println("Start......... init() of cb-network service.go")
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
		fmt.Printf("not exist - %v\n", logConfPath)
		// Load cb-log config from the project directory (usually for development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		fmt.Printf("projectRoot: %v\n", string(path))
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
	fmt.Println("End......... init() of cb-network service.go")
}

type serverSystemManagement struct {
	pb.UnimplementedSystemManagementServiceServer
}

func (s *serverSystemManagement) Health(ctx context.Context, in *emptypb.Empty) (*wrapperspb.StringValue, error) {
	return &wrapperspb.StringValue{Value: "healthy"}, status.New(codes.OK, "").Err()
}

func (s *serverSystemManagement) ControlCloudAdaptiveNetwork(ctx context.Context, req *pb.ControlRequest) (*pb.ControlResponse, error) {
	CBLogger.Debug("Start.........")

	CBLogger.Tracef("Received profile: %v", req)

	CBLogger.Debugf("CLADNet ID: %#v", req.CladnetId)
	CBLogger.Debugf("Command: %#v", req.CommandType)

	cladnetID := req.CladnetId
	commandType := req.CommandType.String()

	controlResponse := &pb.ControlResponse{
		IsSucceeded: false,
		Message:     "",
	}

	// Get a networking rule of a cloud adaptive network
	keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + cladnetID)
	CBLogger.Debugf("Get - %v", keyNetworkingRule)
	resp, err := etcdClient.Get(context.TODO(), keyNetworkingRule, clientv3.WithPrefix())
	if err != nil {
		CBLogger.Error(err)
		controlResponse.Message = fmt.Sprintf("error while getting the networking rule: %v\n", err)
		return controlResponse, status.Errorf(codes.Internal, err.Error())
	}

	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		CBLogger.Tracef("Key : %v", key)
		CBLogger.Tracef("The peer: %v", string(kv.Value))

		var peer model.Peer
		err := json.Unmarshal(kv.Value, &peer)
		if err != nil {
			CBLogger.Error(err)
		}

		// Put the evaluation specification of the CLADNet to the etcd
		keyControlCommand := fmt.Sprint(etcdkey.ControlCommand + "/" + peer.CLADNetID + "/" + peer.HostID)

		CBLogger.Debugf("Put - %v", keyControlCommand)
		controlCommandBody := cmdtype.BuildCommandMessage(commandType)
		CBLogger.Tracef("%#v", controlCommandBody)
		//spec := message.Text
		_, err = etcdClient.Put(context.Background(), keyControlCommand, controlCommandBody)
		if err != nil {
			CBLogger.Error(err)
			controlResponse.Message = fmt.Sprintf("error while putting the command: %v\n", err)
			return controlResponse, status.Errorf(codes.Internal, err.Error())
		}
	}

	controlResponse.IsSucceeded = true
	controlResponse.Message = "The command successfully transferred."

	CBLogger.Debug("End.........")

	return controlResponse, status.New(codes.OK, "").Err()
}

func (s *serverSystemManagement) TestCloudAdaptiveNetwork(ctx context.Context, req *pb.TestRequest) (*pb.TestResponse, error) {
	CBLogger.Debug("Start.........")

	CBLogger.Tracef("Received profile: %v", req)

	CBLogger.Debugf("CLADNet ID: %#v", req.CladnetId)
	CBLogger.Debugf("TestType: %#v", req.TestType)
	CBLogger.Debugf("TestSpec: %#v", req.TestSpec)

	cladnetID := req.CladnetId
	testType := req.TestType.String()
	testSpec := req.TestSpec

	testResponse := &pb.TestResponse{
		IsSucceeded: false,
		Message:     "",
	}

	if testSpec == "" || testSpec == "string" {
		tempSpec, err := json.Marshal(model.TestSpecification{
			CLADNetID:  cladnetID,
			TrialCount: 1,
		})

		if err != nil {
			CBLogger.Error(err)
		}
		testSpec = string(tempSpec)
	}

	// Get a networking rule of a cloud adaptive network
	keyNetworkingRule := fmt.Sprint(etcdkey.NetworkingRule + "/" + cladnetID)
	CBLogger.Debugf("Get - %v", keyNetworkingRule)
	resp, err := etcdClient.Get(context.TODO(), keyNetworkingRule, clientv3.WithPrefix())
	if err != nil {
		CBLogger.Error(err)
		testResponse.Message = fmt.Sprintf("error while getting the networking rule: %v\n", err)
		return testResponse, status.Errorf(codes.Internal, err.Error())
	}

	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		CBLogger.Tracef("Key : %v", key)
		CBLogger.Tracef("The peer: %v", string(kv.Value))

		var peer model.Peer
		err := json.Unmarshal(kv.Value, &peer)
		if err != nil {
			CBLogger.Error(err)
		}

		// Put the evaluation specification of the CLADNet to the etcd
		keyTestRequest := fmt.Sprint(etcdkey.TestRequest + "/" + peer.CLADNetID + "/" + peer.HostID)

		CBLogger.Debugf("Put - %v", keyTestRequest)
		testRequestBody := testtype.BuildTestMessage(testType, testSpec)
		CBLogger.Tracef("%#v", testRequestBody)
		//spec := message.Text
		_, err = etcdClient.Put(context.Background(), keyTestRequest, testRequestBody)
		if err != nil {
			CBLogger.Error(err)
			testResponse.Message = fmt.Sprintf("error while putting the command: %v\n", err)
			return testResponse, status.Errorf(codes.Internal, err.Error())
		}
	}

	testResponse.IsSucceeded = true
	testResponse.Message = "The command successfully transferred."

	CBLogger.Debug("End.........")

	return testResponse, status.New(codes.OK, "").Err()
}

type serverCloudAdaptiveNetwork struct {
	pb.UnimplementedCloudAdaptiveNetworkServiceServer
}

func (s *serverCloudAdaptiveNetwork) GetCLADNet(ctx context.Context, cladnetID *pb.CLADNetID) (*pb.CLADNetSpecification, error) {
	log.Printf("Received profile: %v", cladnetID)

	// Get a specification of the CLADNet
	keyCLADNetSpecificationOfCLADNet := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + cladnetID.CladnetId)
	respSpec, errSpec := etcdClient.Get(context.Background(), keyCLADNetSpecificationOfCLADNet)
	if errSpec != nil {
		CBLogger.Error(errSpec)
		return nil, status.Errorf(codes.Internal, "error while getting a CLADNetSpecification: %v\n", errSpec)
	}

	var tempCLADNetSpec model.CLADNetSpecification

	// Unmarshal the specification of the CLADNet if exists
	CBLogger.Tracef("RespRule.Kvs: %v", respSpec.Kvs)
	if len(respSpec.Kvs) != 0 {
		errUnmarshal := json.Unmarshal(respSpec.Kvs[0].Value, &tempCLADNetSpec)
		if errUnmarshal != nil {
			CBLogger.Error(errUnmarshal)
		}
		CBLogger.Tracef("TempSpec: %v", tempCLADNetSpec)

		return &pb.CLADNetSpecification{
			Id:               tempCLADNetSpec.ID,
			Name:             tempCLADNetSpec.Name,
			Ipv4AddressSpace: tempCLADNetSpec.Ipv4AddressSpace,
			Description:      tempCLADNetSpec.Description}, status.New(codes.OK, "").Err()
	}
	return nil, status.Errorf(codes.NotFound, "Cannot find a CLADNet by %v\n", cladnetID.CladnetId)
}

func (s *serverCloudAdaptiveNetwork) GetCLADNetList(ctx context.Context, in *empty.Empty) (*pb.CLADNetSpecifications, error) {
	// Get all specification of the CLADNet
	respSpecs, errSpec := etcdClient.Get(context.Background(), etcdkey.CLADNetSpecification, clientv3.WithPrefix())
	if errSpec != nil {
		CBLogger.Error(errSpec)
		return nil, status.Errorf(codes.Internal, "Error while putting CLADNetSpecification: %v\n", errSpec)
	}

	// Unmarshal the specification of the CLADNet if exists
	CBLogger.Tracef("RespRule.Kvs: %v", respSpecs.Kvs)
	if len(respSpecs.Kvs) != 0 {

		specs := &pb.CLADNetSpecifications{}

		for _, spec := range respSpecs.Kvs {
			var tempCLADNetSpec model.CLADNetSpecification
			errUnmarshal := json.Unmarshal(spec.Value, &tempCLADNetSpec)
			if errUnmarshal != nil {
				CBLogger.Error(errUnmarshal)
			}
			CBLogger.Tracef("TempSpec: %v", tempCLADNetSpec)
			specs.CladnetSpecifications = append(specs.CladnetSpecifications, &pb.CLADNetSpecification{
				Id:               tempCLADNetSpec.ID,
				Name:             tempCLADNetSpec.Name,
				Ipv4AddressSpace: tempCLADNetSpec.Ipv4AddressSpace,
				Description:      tempCLADNetSpec.Description})
		}
		return specs, status.New(codes.OK, "").Err()
	}

	return nil, status.Error(codes.NotFound, "No CLADNet exists\n")
}

func (s *serverCloudAdaptiveNetwork) CreateCLADNet(ctx context.Context, cladnetSpec *pb.CLADNetSpecification) (*pb.CLADNetSpecification, error) {
	log.Printf("Received profile: %v", cladnetSpec)

	// Generate a unique CLADNet ID by the xid package
	guid := xid.New()
	CBLogger.Tracef("A unique CLADNet ID: %v", guid)
	cladnetSpec.Id = guid.String()

	// Currently assign the 1st IP address for Gateway IP (Not used till now)
	ipv4Address, _, errParseCIDR := net.ParseCIDR(cladnetSpec.Ipv4AddressSpace)
	if errParseCIDR != nil {
		CBLogger.Error(errParseCIDR)
		return nil, status.Errorf(codes.Internal, "Error while parsing CIDR: %v\n", errParseCIDR)
	}
	CBLogger.Tracef("IPv4Address: %v", ipv4Address)

	// [Keep] Assign gateway IP address
	// ip := ipv4Address.To4()
	// gatewayIP := nethelper.IncrementIP(ip, 1)
	// cladnetSpec.GatewayIP = gatewayIP.String()
	// CBLogger.Tracef("GatewayIP: %v", cladNetSpec.GatewayIP)

	// Put the specification of the CLADNet to the etcd
	spec := &model.CLADNetSpecification{
		ID:               cladnetSpec.Id,
		Name:             cladnetSpec.Name,
		Ipv4AddressSpace: cladnetSpec.Ipv4AddressSpace,
		Description:      cladnetSpec.Description}

	bytesCLADNetSpec, _ := json.Marshal(spec)
	CBLogger.Tracef("%#v", spec)

	keyCLADNetSpecificationOfCLADNet := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + cladnetSpec.Id)
	_, err := etcdClient.Put(context.Background(), keyCLADNetSpecificationOfCLADNet, string(bytesCLADNetSpec))
	if err != nil {
		CBLogger.Error(err)
		return nil, status.Errorf(codes.Internal, "Error while putting CLADNetSpecification: %v\n", err)
	}

	return &pb.CLADNetSpecification{
		Id:               cladnetSpec.Id,
		Name:             cladnetSpec.Name,
		Ipv4AddressSpace: cladnetSpec.Ipv4AddressSpace,
		Description:      cladnetSpec.Description}, status.New(codes.OK, "").Err()
}

func (s *serverCloudAdaptiveNetwork) RecommendAvailableIPv4PrivateAddressSpaces(ctx context.Context, ipnets *pb.IPNetworks) (*pb.AvailableIPv4PrivateAddressSpaces, error) {
	log.Printf("Received: %#v", ipnets.IpNetworks)

	availableSpaces := nethelper.GetAvailableIPv4PrivateAddressSpaces(ipnets.IpNetworks)
	response := &pb.AvailableIPv4PrivateAddressSpaces{
		RecommendedIpv4PrivateAddressSpace: availableSpaces.RecommendedIPv4PrivateAddressSpace,
		AddressSpace10S:                    availableSpaces.AddressSpace10s,
		AddressSpace172S:                   availableSpaces.AddressSpace172s,
		AddressSpace192S:                   availableSpaces.AddressSpace192s}

	return response, status.New(codes.OK, "").Err()
}

// If "Content-Type: application/grpc", use gRPC server handler,
// Otherwise, use gRPC Gateway handler (for REST API)
func grpcHandler(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}

func main() {

	// Set web assets path to the current directory (usually for the production)
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	CBLogger.Tracef("exePath: %v", exePath)
	docsPath := filepath.Join(exePath, "docs")

	swaggerJSONPath := filepath.Join(docsPath, "cloud_barista_network.swagger.json")
	CBLogger.Tracef("swaggerJsonPath: %v", swaggerJSONPath)
	if !file.Exists(swaggerJSONPath) {
		// Set web assets path to the project directory (usually for the development)
		var path []byte
		path, err = exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		docsPath = filepath.Join(projectPath, "poc-cb-net", "docs")
	}

	//// etcd section
	etcdClient, err = clientv3.New(clientv3.Config{
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

	//// gRPC and REST service section

	// Multiplexer (mux) to handle requests for gRPC, REST, and Swagger dashboards respectively
	mux := http.NewServeMux()
	// NOTE - ServeMux (mux) is an HTTP request multiplexer. It matches the URL of each
	// incoming request against a list of registered patterns and calls the
	// handler for the pattern that most closely matches the URL.

	// Create a gRPC server object
	grpcServer := grpc.NewServer()
	// Attach the CloudAdaptiveNetwork service to the server
	pb.RegisterSystemManagementServiceServer(grpcServer, &serverSystemManagement{})
	pb.RegisterCloudAdaptiveNetworkServiceServer(grpcServer, &serverCloudAdaptiveNetwork{})

	// Create echo server to provide Swagger dashboard
	e := echo.New()

	// Middleware
	// e.Use(middleware.Logger())
	// e.Use(middleware.Recover())

	e.HideBanner = true
	//e.colorer.Printf(banner, e.colorer.Red("v"+Version), e.colorer.Blue(website))

	// Read swagger.json
	swaggerJSONPath = filepath.Join(docsPath, "cloud_barista_network.swagger.json")
	swaggerJSON, err := ioutil.ReadFile(swaggerJSONPath)
	if err != nil {
		CBLogger.Error(err)
	}
	swaggerStr := string(swaggerJSON)

	// Match "/cloud_barista_network.swagger.json" request to the handler function, which provides "swagger.json"
	mux.HandleFunc("/cloud_barista_network.swagger.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swaggerStr))
	})

	// On Swagger dashboard, the url pointing to API definition
	url := echoSwagger.URL("/cloud_barista_network.swagger.json")
	// Route "/*" request to echoSwagger, which includes Swagger UI
	e.GET("/*", echoSwagger.EchoWrapHandler(url))

	// Match "/swagger" to echo server
	mux.Handle("/swagger/", e)

	// gRPC Gateway section
	// Create a gRPC Gateway mux for gRPC service and REST service
	gwmux := runtime.NewServeMux()

	// Register CloudAdaptiveNetwork handler to gwmux
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	addr := fmt.Sprintf(":%s", config.Service.Port)
	err = pb.RegisterSystemManagementServiceHandlerFromEndpoint(context.Background(), gwmux, addr, options)
	if err != nil {
		CBLogger.Fatalf("Failed to register gateway: %v", err)
	}

	err = pb.RegisterCloudAdaptiveNetworkServiceHandlerFromEndpoint(context.Background(), gwmux, addr, options)
	if err != nil {
		CBLogger.Fatalf("Failed to register gateway: %v", err)
	}

	// Match "/" request to gRPC gateway mux
	mux.Handle("/", gwmux)

	// Display API documents (gRPC protocol documentation, REST API documentation by Swagger)
	swaggerURL := fmt.Sprintf("http://%s/swagger/index.html", config.Service.Endpoint)
	grpcDocURL := "https://github.com/cloud-barista/cb-larva/blob/main/poc-cb-net/docs/cloud-adaptive-network-service.md"

	CBLogger.Infof("Serving gRPC-Gateway(gRPC, REST), Swagger dashboard on %v", addr)

	fmt.Println("")
	fmt.Printf("\033[1;36m%s\033[0m\n", "[The cb-network service]")
	fmt.Printf("\033[1;36m * gRPC protocol document\033[0m\n")
	fmt.Printf("\033[1;36m   ==> %s\033[0m\n", grpcDocURL)
	fmt.Println("")
	fmt.Printf("\033[1;36m * Swagger dashboard(set in 'config.yaml')\033[0m\n")
	fmt.Printf("\033[1;36m   ==> %s\033[0m\n", swaggerURL)
	fmt.Println("")

	// Serve gRPC server and gRPC Gateway by "grpcHandler"
	err = http.ListenAndServe(addr, grpcHandler(grpcServer, mux))
	if err != nil {
		CBLogger.Fatalf("Failed to listen and serve: %v", err)
	}
}
