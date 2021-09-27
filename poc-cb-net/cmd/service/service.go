package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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

type server struct {
	pb.UnimplementedCloudAdaptiveNetworkServer
}

func (s *server) CreateCLADNet(ctx context.Context, in *pb.CreateCLADNetRequest) (*pb.CLADNetResponse, error) {
	log.Printf("Received profile: %v", in.CladnetSpecification)
	in.CladnetSpecification.Id = "0"
	cladnetResponse := &pb.CLADNetResponse{
		IsSucceeded:          true,
		Message:              "",
		CladnetSpecification: in.CladnetSpecification}

	return cladnetResponse, nil
}

func (s *server) GetCLADNet(ctx context.Context, in *pb.CLADNetID) (*pb.CLADNetResponse, error) {
	log.Printf("Received profile: %v", in.Value)
	cladnetResponse := &pb.CLADNetResponse{
		IsSucceeded: true,
		Message:     "",
		CladnetSpecification: &pb.CLADNetSpecification{
			Id:               "0",
			Name:             "AAA",
			Ipv4AddressSpace: "192.168.77.0/28",
			Description:      "Described"}}

	return cladnetResponse, nil
}

func main() {
	lis, err := net.Listen("tcp", config.GRPCServer.ListenPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterCloudAdaptiveNetworkServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
