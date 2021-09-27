package main

import (
	"context"
	"log"
	"net"

	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	"google.golang.org/grpc"
)

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
	lis, err := net.Listen("tcp", ":8088")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterCloudAdaptiveNetworkServer(s, &server{})
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
