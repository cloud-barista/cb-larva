package main

import (
	"context"
	"encoding/json"
	"errors"
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
	ruletype "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/rule-type"
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
	fmt.Println("\nStart......... init() of cb-network service.go")

	// Set cb-log
	env := os.Getenv("CBLOG_ROOT")
	if env != "" {
		// Load cb-log config from the environment variable path (default)
		fmt.Printf("CBLOG_ROOT: %v\n", env)
		CBLogger = cblog.GetLogger("cb-network")
	} else {

		// Load cb-log config from the current directory (usually for the production)
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exePath := filepath.Dir(ex)
		// fmt.Printf("exe path: %v\n", exePath)

		logConfPath := filepath.Join(exePath, "config", "log_conf.yaml")
		if file.Exists(logConfPath) {
			fmt.Printf("path of log_conf.yaml: %v\n", logConfPath)
			CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)

		} else {
			// Load cb-log config from the project directory (usually for development)
			logConfPath = filepath.Join(exePath, "..", "..", "config", "log_conf.yaml")
			if file.Exists(logConfPath) {
				fmt.Printf("path of log_conf.yaml: %v\n", logConfPath)
				CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)
			} else {
				err := errors.New("fail to load log_conf.yaml")
				panic(err)
			}
		}
		CBLogger.Debugf("Load %v", logConfPath)

	}

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

	CBLogger.Debugf("Load %v", configPath)

	fmt.Println("End......... init() of cb-network service.go")
	fmt.Println("")
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

	// Get all peers in a Cloud Adaptive Network
	keyPeers := fmt.Sprint(etcdkey.Peer + "/" + cladnetID)
	CBLogger.Debugf("Get - %v", keyPeers)
	resp, err := etcdClient.Get(context.TODO(), keyPeers, clientv3.WithPrefix())
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
		keyControlCommand := fmt.Sprint(etcdkey.ControlCommand + "/" + peer.CladnetID + "/" + peer.HostID)

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
			CladnetID:  cladnetID,
			TrialCount: 1,
		})

		if err != nil {
			CBLogger.Error(err)
		}
		testSpec = string(tempSpec)
	}

	// Get all peers in a Cloud Adaptive Network
	keyPeersInCLADNet := fmt.Sprint(etcdkey.Peer + "/" + cladnetID)
	CBLogger.Debugf("Get - %v", keyPeersInCLADNet)
	resp, err := etcdClient.Get(context.TODO(), keyPeersInCLADNet, clientv3.WithPrefix())
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
		keyTestRequest := fmt.Sprint(etcdkey.TestRequest + "/" + peer.CladnetID + "/" + peer.HostID)

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

func (s *serverCloudAdaptiveNetwork) GetCLADNet(ctx context.Context, req *pb.CLADNetRequest) (*pb.CLADNetSpecification, error) {
	log.Printf("Received profile: %v", req)

	// Get a specification of the CLADNet
	keyCLADNetSpecificationOfCLADNet := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + req.CladnetId)
	respSpec, errSpec := etcdClient.Get(context.Background(), keyCLADNetSpecificationOfCLADNet)
	if errSpec != nil {
		CBLogger.Error(errSpec)
		return &pb.CLADNetSpecification{}, status.Errorf(codes.Internal, "error while getting a CLADNetSpecification: %v\n", errSpec)
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

		spec := &pb.CLADNetSpecification{
			CladnetId:        tempCLADNetSpec.CladnetID,
			Name:             tempCLADNetSpec.Name,
			Ipv4AddressSpace: tempCLADNetSpec.Ipv4AddressSpace,
			Description:      tempCLADNetSpec.Description,
			RuleType:         tempCLADNetSpec.RuleType,
		}
		return spec, status.New(codes.OK, "").Err()
	}
	return &pb.CLADNetSpecification{}, status.Errorf(codes.NotFound, "could not find a CLADNet by %v\n", req.CladnetId)
}

func (s *serverCloudAdaptiveNetwork) GetCLADNetList(ctx context.Context, in *empty.Empty) (*pb.CLADNetSpecifications, error) {
	// Get all specification of the CLADNet
	respSpecs, errSpec := etcdClient.Get(context.Background(), etcdkey.CLADNetSpecification, clientv3.WithPrefix())
	if errSpec != nil {
		CBLogger.Error(errSpec)
		return nil, status.Errorf(codes.Internal, "error while getting a list of CLADNetSpecifications: %v\n", errSpec)
	}

	// Unmarshal the specification of the CLADNet if exists
	CBLogger.Tracef("RespRule.Kvs: %v", respSpecs.Kvs)
	if len(respSpecs.Kvs) != 0 {

		specs := &pb.CLADNetSpecifications{}

		for _, specKv := range respSpecs.Kvs {
			var tempSpec model.CLADNetSpecification
			errUnmarshal := json.Unmarshal(specKv.Value, &tempSpec)
			if errUnmarshal != nil {
				CBLogger.Error(errUnmarshal)
			}
			CBLogger.Tracef("TempSpec: %v", tempSpec)
			specs.CladnetSpecifications = append(specs.CladnetSpecifications, &pb.CLADNetSpecification{
				CladnetId:        tempSpec.CladnetID,
				Name:             tempSpec.Name,
				Ipv4AddressSpace: tempSpec.Ipv4AddressSpace,
				Description:      tempSpec.Description,
				RuleType:         tempSpec.RuleType,
			})
		}
		return specs, status.New(codes.OK, "").Err()
	}

	return nil, status.Error(codes.NotFound, "could not find any CLADNetSpecifications")
}

func (s *serverCloudAdaptiveNetwork) CreateCLADNet(ctx context.Context, cladnetSpec *pb.CLADNetSpecification) (*pb.CLADNetSpecification, error) {
	log.Printf("Received profile: %v", cladnetSpec)

	// NOTE - A user can assign the ID of Cloud Adaptive Network. It must be unique one.
	if cladnetSpec.CladnetId != "" {
		// Check if the Cloud Adaptive Network exists or not

		// Request body
		req := &pb.CLADNetRequest{
			CladnetId: cladnetSpec.CladnetId,
		}

		// Get a cloud adaptive network
		_, err := s.GetCLADNet(context.TODO(), req)

		s, ok := status.FromError(err)
		if !ok {
			return &pb.CLADNetSpecification{}, err
		}

		switch s.Code() {
		case codes.OK: // if OK, already exist.
			return &pb.CLADNetSpecification{}, status.Errorf(codes.AlreadyExists, "already exists (CladnetID: %s)", cladnetSpec.CladnetId)
		case codes.Internal:
			return &pb.CLADNetSpecification{}, err
		}
	}

	// Assign a unique CLADNet ID if a user didn't pass it.
	if cladnetSpec.CladnetId == "" {
		// Generate a unique CLADNet ID by the xid package
		guid := xid.New()
		CBLogger.Tracef("A unique CLADNet ID: %v", guid)
		cladnetSpec.CladnetId = guid.String()
	}

	// Currently assign the 1st IP address for Gateway IP (Not used till now)
	ipv4Address, _, errParseCIDR := net.ParseCIDR(cladnetSpec.Ipv4AddressSpace)
	if errParseCIDR != nil {
		CBLogger.Error(errParseCIDR)
		return &pb.CLADNetSpecification{}, status.Errorf(codes.Internal, "error while parsing CIDR: %v\n", errParseCIDR)
	}
	CBLogger.Tracef("IPv4Address: %v", ipv4Address)

	// [Keep] Assign gateway IP address
	// ip := ipv4Address.To4()
	// gatewayIP := nethelper.IncrementIP(ip, 1)
	// cladnetSpec.GatewayIP = gatewayIP.String()
	// CBLogger.Tracef("GatewayIP: %v", cladNetSpec.GatewayIP)

	ruleType := cladnetSpec.RuleType
	if ruleType == "" {
		ruleType = ruletype.Basic
	}

	// Put the specification of the CLADNet to the etcd
	spec := &model.CLADNetSpecification{
		CladnetID:        cladnetSpec.CladnetId,
		Name:             cladnetSpec.Name,
		Ipv4AddressSpace: cladnetSpec.Ipv4AddressSpace,
		Description:      cladnetSpec.Description,
		RuleType:         ruleType,
	}

	bytesCLADNetSpec, _ := json.Marshal(spec)
	CBLogger.Tracef("%#v", spec)

	keyCLADNetSpecification := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + cladnetSpec.CladnetId)
	_, err := etcdClient.Put(context.TODO(), keyCLADNetSpecification, string(bytesCLADNetSpec))
	if err != nil {
		CBLogger.Error(err)
		return &pb.CLADNetSpecification{}, status.Errorf(codes.Internal, "error while putting CLADNetSpecification: %v", err)
	}

	return &pb.CLADNetSpecification{
		CladnetId:        cladnetSpec.CladnetId,
		Name:             cladnetSpec.Name,
		Ipv4AddressSpace: cladnetSpec.Ipv4AddressSpace,
		Description:      cladnetSpec.Description}, status.New(codes.OK, "").Err()
}

func (s *serverCloudAdaptiveNetwork) UpdateCLADNet(ctx context.Context, cladnetSpec *pb.CLADNetSpecification) (*pb.CLADNetSpecification, error) {
	log.Printf("Received: %#v", cladnetSpec)

	// Check if the Cloud Adaptive Network exists or not
	req := &pb.CLADNetRequest{
		CladnetId: cladnetSpec.CladnetId,
	}

	if _, err := s.GetCLADNet(context.TODO(), req); err != nil {
		return &pb.CLADNetSpecification{}, err
	}

	// Update the Cloud Adaptive Network
	tempSpec := model.CLADNetSpecification{
		CladnetID:        cladnetSpec.CladnetId,
		Name:             cladnetSpec.Name,
		Ipv4AddressSpace: cladnetSpec.Ipv4AddressSpace,
		Description:      cladnetSpec.Description,
		RuleType:         cladnetSpec.RuleType,
	}

	specBytes, _ := json.Marshal(tempSpec)
	CBLogger.Tracef("%#v", specBytes)

	keyPeerOfCLADNet := fmt.Sprint(etcdkey.CLADNetSpecification + "/" + cladnetSpec.CladnetId)
	_, err := etcdClient.Put(context.Background(), keyPeerOfCLADNet, string(specBytes))
	if err != nil {
		CBLogger.Error(err)
		return &pb.CLADNetSpecification{}, status.Errorf(codes.Internal, "error while updating the peer: %v", err)
	}

	// Get and return the updated Cloud Adaptive Network
	cladnetSpec, err = s.GetCLADNet(context.TODO(), req)
	if err != nil {
		return &pb.CLADNetSpecification{}, err
	}
	return cladnetSpec, status.New(codes.OK, "").Err()
}

func (s *serverCloudAdaptiveNetwork) RecommendAvailableIPv4PrivateAddressSpaces(ctx context.Context, ipv4CIDRs *pb.IPv4CIDRs) (*pb.AvailableIPv4PrivateAddressSpaces, error) {
	log.Printf("Received: %#v", ipv4CIDRs.Ipv4Cidrs)

	availableSpaces := nethelper.GetAvailableIPv4PrivateAddressSpaces(ipv4CIDRs.Ipv4Cidrs)
	response := &pb.AvailableIPv4PrivateAddressSpaces{
		RecommendedIpv4PrivateAddressSpace: availableSpaces.RecommendedIPv4PrivateAddressSpace,
		AddressSpace10S:                    availableSpaces.AddressSpace10s,
		AddressSpace172S:                   availableSpaces.AddressSpace172s,
		AddressSpace192S:                   availableSpaces.AddressSpace192s}

	return response, status.New(codes.OK, "").Err()
}

func (s *serverCloudAdaptiveNetwork) GetPeer(ctx context.Context, req *pb.PeerRequest) (*pb.Peer, error) {
	log.Printf("Received: %#v", req)

	// Get a peer of the CLADNet
	keyPeer := fmt.Sprint(etcdkey.Peer + "/" + req.CladnetId + "/" + req.HostId)
	respPeer, errEtcd := etcdClient.Get(context.Background(), keyPeer)
	if errEtcd != nil {
		CBLogger.Error(errEtcd)
		return &pb.Peer{}, status.Errorf(codes.Internal, "error while getting a peer: %v", respPeer)
	}

	// Unmarshal the peer of the CLADNet if exists
	CBLogger.Tracef("RespRule.Kvs: %v", respPeer.Kvs)
	if respPeer.Count != 0 {
		tempPeer := model.Peer{}
		errUnmarshal := json.Unmarshal(respPeer.Kvs[0].Value, &tempPeer)
		if errUnmarshal != nil {
			CBLogger.Error(errUnmarshal)
		}
		CBLogger.Tracef("Peer: %v", tempPeer)

		peer := &pb.Peer{
			CladnetId:           tempPeer.CladnetID,
			HostId:              tempPeer.HostID,
			HostName:            tempPeer.HostName,
			HostPrivateIpv4Cidr: tempPeer.HostPrivateIPv4CIDR,
			HostPrivateIp:       tempPeer.HostPrivateIP,
			HostPublicIp:        tempPeer.HostPublicIP,
			Ipv4Cidr:            tempPeer.IPv4CIDR,
			Ip:                  tempPeer.IP,
			State:               tempPeer.State,
			Details: &pb.CloudInformation{
				ProviderName:       tempPeer.Details.ProviderName,
				RegionId:           tempPeer.Details.RegionID,
				AvailabilityZoneId: tempPeer.Details.AvailabilityZoneID,
				VirtualNetworkId:   tempPeer.Details.VirtualNetworkID,
				SubnetId:           tempPeer.Details.SubnetID,
			},
		}

		return peer, status.New(codes.OK, "").Err()
	}

	return &pb.Peer{}, status.Errorf(codes.NotFound, "not found a peer by cladnetId (%+v) and hostId (%+v)", req.CladnetId, req.HostId)
}

func (s *serverCloudAdaptiveNetwork) GetPeerList(ctx context.Context, req *pb.PeerRequest) (*pb.Peers, error) {
	log.Printf("Received: %#v", req)

	// Get peers in a Cloud Adaptive Network
	keyPeersInCLADNet := fmt.Sprint(etcdkey.Peer + "/" + req.CladnetId)
	respPeers, errEtcd := etcdClient.Get(context.TODO(), keyPeersInCLADNet, clientv3.WithPrefix())
	if errEtcd != nil {
		CBLogger.Error(errEtcd)
		return nil, status.Errorf(codes.Internal, "error while getting peers: %v", respPeers)
	}

	// Unmarshal peers of the CLADNet if exists
	CBLogger.Tracef("RespPeers.Kvs: %v", respPeers.Kvs)
	if respPeers.Count != 0 {

		peers := &pb.Peers{}

		for _, peerKv := range respPeers.Kvs {

			tempPeer := model.Peer{}
			errUnmarshal := json.Unmarshal(peerKv.Value, &tempPeer)
			if errUnmarshal != nil {
				CBLogger.Error(errUnmarshal)
			}
			CBLogger.Tracef("Peer: %v", peerKv)

			peers.Peers = append(peers.Peers, &pb.Peer{
				CladnetId:           tempPeer.CladnetID,
				HostId:              tempPeer.HostID,
				HostName:            tempPeer.HostName,
				HostPrivateIpv4Cidr: tempPeer.HostPrivateIPv4CIDR,
				HostPrivateIp:       tempPeer.HostPrivateIP,
				HostPublicIp:        tempPeer.HostPublicIP,
				Ipv4Cidr:            tempPeer.IPv4CIDR,
				Ip:                  tempPeer.IP,
				State:               tempPeer.State,
				Details: &pb.CloudInformation{
					ProviderName:       tempPeer.Details.ProviderName,
					RegionId:           tempPeer.Details.RegionID,
					AvailabilityZoneId: tempPeer.Details.AvailabilityZoneID,
					VirtualNetworkId:   tempPeer.Details.VirtualNetworkID,
					SubnetId:           tempPeer.Details.SubnetID,
				},
			})
		}
		return peers, status.New(codes.OK, "").Err()
	}

	return &pb.Peers{}, status.Errorf(codes.NotFound, "not found any peer by cladnetId (%+v)", req.CladnetId)
}

func (s *serverCloudAdaptiveNetwork) UpdateDetailsOfPeer(ctx context.Context, req *pb.UpdateDetailsRequest) (*pb.Peer, error) {
	log.Printf("Received: %#v", req)

	// Check if the peer exists or not
	peerReq := &pb.PeerRequest{
		CladnetId: req.CladnetId,
		HostId:    req.HostId,
	}

	peer, err := s.GetPeer(context.TODO(), peerReq)
	if err != nil {
		CBLogger.Errorf("%#v", err)
		return &pb.Peer{}, err
	}

	// Update the peer
	tempPeer := model.Peer{
		CladnetID:           peer.CladnetId,
		HostID:              peer.HostId,
		HostName:            peer.HostName,
		HostPrivateIPv4CIDR: peer.HostPrivateIpv4Cidr,
		HostPrivateIP:       peer.HostPrivateIp,
		HostPublicIP:        peer.HostPublicIp,
		IPv4CIDR:            peer.Ipv4Cidr,
		IP:                  peer.Ip,
		State:               peer.State,
		Details: model.CloudInformation{
			ProviderName:       req.CloudInformation.ProviderName,
			RegionID:           req.CloudInformation.RegionId,
			AvailabilityZoneID: req.CloudInformation.AvailabilityZoneId,
			VirtualNetworkID:   req.CloudInformation.VirtualNetworkId,
			SubnetID:           req.CloudInformation.SubnetId,
		},
	}

	peerBytes, _ := json.Marshal(tempPeer)
	doc := string(peerBytes)
	CBLogger.Tracef("%#v", doc)

	keyPeer := fmt.Sprint(etcdkey.Peer + "/" + peer.CladnetId + "/" + peer.HostId)
	_, err = etcdClient.Put(context.Background(), keyPeer, doc)
	if err != nil {
		CBLogger.Error(err)
		return &pb.Peer{}, status.Errorf(codes.Internal, "error while updating the peer: %v", err)
	}

	// Get and return the updated peer
	peer, err = s.GetPeer(context.TODO(), peerReq)
	if err != nil {
		return &pb.Peer{}, err
	}
	return peer, status.New(codes.OK, "").Err()
}

func (s *serverCloudAdaptiveNetwork) GetPeerNetworkingRule(ctx context.Context, req *pb.PeerRequest) (*pb.NetworkingRule, error) {
	log.Printf("Received: %#v", req)

	// Get a peer's networking rule
	keyNetworkingRuleOfPeer := fmt.Sprint(etcdkey.NetworkingRule + "/" + req.CladnetId + "/" + req.HostId)
	respNetworkingRule, errEtcd := etcdClient.Get(context.Background(), keyNetworkingRuleOfPeer)
	if errEtcd != nil {
		CBLogger.Error(errEtcd)
		return &pb.NetworkingRule{}, status.Errorf(codes.Internal, "error while getting a peer's networking rule: %v", errEtcd)
	}

	// Unmarshal the networking rule if exists
	CBLogger.Tracef("RespRule.Kvs: %v", respNetworkingRule.Kvs)
	if respNetworkingRule.Count != 0 {
		tempNetworkingRule := model.NetworkingRule{}
		errUnmarshal := json.Unmarshal(respNetworkingRule.Kvs[0].Value, &tempNetworkingRule)
		if errUnmarshal != nil {
			CBLogger.Error(errUnmarshal)
		}
		CBLogger.Tracef("Peer: %v", tempNetworkingRule)

		networkingRule := &pb.NetworkingRule{
			CladnetId:  tempNetworkingRule.CladnetID,
			HostId:     tempNetworkingRule.HostID,
			HostName:   tempNetworkingRule.HostName,
			PeerIp:     tempNetworkingRule.PeerIP,
			SelectedIp: tempNetworkingRule.SelectedIP,
			PeerScope:  tempNetworkingRule.PeerScope,
			State:      tempNetworkingRule.State,
		}

		return networkingRule, status.New(codes.OK, "").Err()

	}

	return &pb.NetworkingRule{}, status.Errorf(codes.NotFound, "not found the peer's networking rule by cladnetId (%+v) and hostId (%+v)", req.CladnetId, req.HostId)
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
	grpcDocURL := "https://github.com/cloud-barista/cb-larva/blob/main/poc-cb-net/docs/cloud-barista-network-service.md"

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
