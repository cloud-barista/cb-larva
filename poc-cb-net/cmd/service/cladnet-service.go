package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/internal/etcd-key"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/rs/xid"
	"github.com/sirupsen/logrus"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	fmt.Println("End......... init() of cladnet-service.go")
}

type server struct {
	pb.UnimplementedCloudAdaptiveNetworkServer
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

	// Assign gateway IP address
	// ip := ipv4Address.To4()
	// gatewayIP := nethelper.IncrementIP(ip, 1)
	// cladnetSpec.GatewayIP = gatewayIP.String()
	// CBLogger.Tracef("GatewayIP: %v", cladNetConfInfo.GatewayIP)

	// Put the configuration information of the CLADNet to the etcd
	bytesCLADNetSpec, _ := json.Marshal(&model.CLADNetSpecification{
		ID:               cladnetSpec.Id,
		Name:             cladnetSpec.Name,
		Ipv4AddressSpace: cladnetSpec.Ipv4AddressSpace,
		Description:      cladnetSpec.Description})
	CBLogger.Tracef("CLADNet specification: %v", bytesCLADNetSpec)

	keyConfigurationInformationOfCLADNet := fmt.Sprint(etcdkey.ConfigurationInformation + "/" + cladnetSpec.Id)
	_, err := etcdClient.Put(context.Background(), keyConfigurationInformationOfCLADNet, string(bytesCLADNetSpec))
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

func (s *server) GetCLADNet(ctx context.Context, in *pb.CLADNetID) (*pb.CLADNetSpecification, error) {
	log.Printf("Received profile: %v", in)
	// cladnetResponse := &pb.CLADNetResponse{
	// 	IsSucceeded: true,
	// 	Message:     "",
	// 	CladnetSpecification: &pb.CLADNetSpecification{
	// 		Id:               "0",
	// 		Name:             "AAA",
	// 		Ipv4AddressSpace: "192.168.77.0/28",
	// 		Description:      "Described"}}
	var cladnetSpec pb.CLADNetSpecification

	return &cladnetSpec, status.New(codes.OK, "").Err()
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
	listener, errListen := net.Listen("tcp", config.GRPC.ListenPort)
	if errListen != nil {
		log.Fatalf("failed to listen: %v", errListen)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterCloudAdaptiveNetworkServer(grpcServer, &server{})
	if errGRPCServer := grpcServer.Serve(listener); errGRPCServer != nil {
		log.Fatalf("failed to serve: %v", errGRPCServer)
	}
}
