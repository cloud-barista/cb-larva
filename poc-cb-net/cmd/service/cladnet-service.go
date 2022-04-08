package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/etcd-key"
	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	nethelper "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/network-helper"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
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
	fmt.Println("Start......... init() of cladnet-service.go")
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
	fmt.Println("End......... init() of cladnet-service.go")
}

type server struct {
	pb.UnimplementedCloudAdaptiveNetworkServiceServer
}

// // NewServer represents the default constructor for server
// func NewServer() *server {
// 	return &server{}
// }

// If "Content-Type: application/grpc", use gRPC server handler,
// Otherwise, use gRPC Gateway handler (for REST API)
func grpcHandler(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	})
}

func serveSwagger(mux *http.ServeMux) {
	mime.AddExtensionType(".svg", "image/svg+xml")

	// Set web assets path to the current directory (usually for the production)
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	CBLogger.Tracef("exePath: %v", exePath)
	swaggerUIPath := filepath.Join(exePath, "third_party", "swagger-ui")

	indexPath := filepath.Join(swaggerUIPath, "public", "index.html")
	CBLogger.Tracef("indexPath: %v", indexPath)
	if !file.Exists(indexPath) {
		// Set web assets path to the project directory (usually for the development)
		path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			panic(err)
		}
		projectPath := strings.TrimSpace(string(path))
		swaggerUIPath = filepath.Join(projectPath, "poc-cb-net", "third_party", "swagger-ui")
	}

	// Expose files in third_party/swagger-ui/ on <host>/swagger-ui
	fileServer := http.FileServer(http.Dir(swaggerUIPath))
	prefix := "/swagger-ui/"
	mux.Handle(prefix, http.StripPrefix(prefix, fileServer))
}

func (s *server) SayHello(ctx context.Context, in *emptypb.Empty) (*wrapperspb.StringValue, error) {
	return &wrapperspb.StringValue{Value: "Hi, welcome to CB-Larva"}, status.New(codes.OK, "").Err()
}

func (s *server) GetCLADNet(ctx context.Context, cladnetID *pb.CLADNetID) (*pb.CLADNetSpecification, error) {
	log.Printf("Received profile: %v", cladnetID)

	// Get a specification of the CLADNet
	keyCLADNetSpecificationOfCLADNet := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + cladnetID.Value)
	respSpec, errSpec := etcdClient.Get(context.Background(), keyCLADNetSpecificationOfCLADNet)
	if errSpec != nil {
		CBLogger.Error(errSpec)
		return nil, status.Errorf(codes.Internal, "Error while putting CLADNetSpecification: %v\n", errSpec)
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
	return nil, status.Errorf(codes.NotFound, "Cannot find a CLADNet by %v\n", cladnetID.Value)
}

func (s *server) GetCLADNetList(ctx context.Context, in *empty.Empty) (*pb.CLADNetSpecifications, error) {
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

func (s *server) CreateCLADNet(ctx context.Context, cladnetSpec *pb.CLADNetSpecification) (*pb.CLADNetSpecification, error) {
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

func (s *server) RecommendAvailableIPv4PrivateAddressSpaces(ctx context.Context, ipnets *pb.IPNetworks) (*pb.AvailableIPv4PrivateAddressSpaces, error) {
	log.Printf("Received: %#v", ipnets.IpNetworks)

	availableSpaces := nethelper.GetAvailableIPv4PrivateAddressSpaces(ipnets.IpNetworks)
	response := &pb.AvailableIPv4PrivateAddressSpaces{
		RecommendedIpv4PrivateAddressSpace: availableSpaces.RecommendedIPv4PrivateAddressSpace,
		AddressSpace10S:                    availableSpaces.AddressSpace10s,
		AddressSpace172S:                   availableSpaces.AddressSpace172s,
		AddressSpace192S:                   availableSpaces.AddressSpace192s}

	return response, status.New(codes.OK, "").Err()
}

func main() {

	var err error

	// etcd section
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

	// gRPC section

	// Create a gRPC server object
	grpcServer := grpc.NewServer()
	// Attach the CloudAdaptiveNetwork service to the server
	pb.RegisterCloudAdaptiveNetworkServiceServer(grpcServer, &server{})

	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	mux := http.NewServeMux()

	swaggerJSON, err := ioutil.ReadFile("../../pkg/api/gen/openapiv2/cbnetwork/cloud_adaptive_network.swagger.json")
	if err != nil {
		CBLogger.Error(err)
	}
	swaggerStr := string(swaggerJSON)

	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, strings.NewReader(swaggerStr))
	})

	// gRPC Gateway section
	// Create a gRPC Gateway mux object
	gwmux := runtime.NewServeMux()

	// Register CloudAdaptiveNetwork handler to gwmux
	addr := fmt.Sprintf(":%s", config.GRPC.ServerPort)
	err = pb.RegisterCloudAdaptiveNetworkServiceHandlerFromEndpoint(context.Background(), gwmux, addr, options)
	if err != nil {
		CBLogger.Fatalf("Failed to register gateway: %v", err)
	}

	mux.Handle("/", gwmux)
	serveSwagger(mux)

	CBLogger.Infof("Serving gRPC-Gateway on %v", addr)
	// Serve gRPC server and gRPC Gateway by "allHandler"
	err = http.ListenAndServe(addr, grpcHandler(grpcServer, mux))
	if err != nil {
		CBLogger.Fatalf("Failed to listen and serve: %v", err)
	}
}
