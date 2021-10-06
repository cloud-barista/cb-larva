package main

import (
	"context"
	"log"
	"time"

	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	"google.golang.org/grpc"
)

func main() {

	// gRPC section
	grpcConn, err := grpc.Dial("localhost:8089", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Cannot connect: %v", err)
	}
	defer grpcConn.Close()

	cladnetClient := pb.NewCloudAdaptiveNetworkClient(grpcConn)

	// Request/call CreateCLADNnet()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ipNetworks := &pb.IPNetworks{IpNetworks: []string{"10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16", "172.16.1.0/24", "172.16.2.0/24", "172.16.3.0/24", "172.16.4.0/24", "172.16.5.0/24", "172.16.6.0/24", "192.168.1.0/28", "192.168.2.0/28", "192.168.3.0/28", "192.168.4.0/28", "192.168.5.0/28", "192.168.6.0/28", "192.168.7.0/28"}}

	ret1, err := cladnetClient.RecommendAvailableIPv4PrivateAddressSpaces(ctx, ipNetworks)
	if err != nil {
		log.Fatalf("could not request: %v", err)
	}
	log.Printf("RecommendedIpv4PrivateAddressSpace: %v", ret1.RecommendedIpv4PrivateAddressSpace)

	cladnetSpec := &pb.CLADNetSpecification{
		Id:               "",
		Name:             "CLADNet01",
		Ipv4AddressSpace: ret1.RecommendedIpv4PrivateAddressSpace,
		Description:      "Alvin's CLADNet01"}

	ret2, err := cladnetClient.CreateCLADNet(ctx, cladnetSpec)

	if err != nil {
		log.Fatalf("could not request: %v", err)
	}

	log.Printf("Config: %v", ret2)
}
