package main

import (
	"context"
	"log"
	"time"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	"google.golang.org/grpc"
)

func createProperCloudAdaptiveNetwork(gRPCServiceEndpoint string, ipNetworks []string, cladnetName string, cladnetDescription string) (model.CLADNetSpecification, error) {

	ipNets := &pb.IPNetworks{IpNetworks: ipNetworks}

	// Connect to the gRPC server
	grpcConn, err := grpc.Dial(gRPCServiceEndpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Printf("Cannot connect: %v\n", err)
		return model.CLADNetSpecification{}, err
	}
	defer grpcConn.Close()

	// Create a stub of Cloud AdaptiveNetwork
	cladnetClient := pb.NewCloudAdaptiveNetworkServiceClient(grpcConn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// Call rpc, RecommendAvailableIPv4PrivateAddressSpaces(ctx, ipNets)
	availableIPv4PrivateAddressSpaces, err := cladnetClient.RecommendAvailableIPv4PrivateAddressSpaces(ctx, ipNets)
	if err != nil {
		log.Printf("Could not request: %v\n", err)
		return model.CLADNetSpecification{}, err
	}
	log.Printf("RecommendedIpv4PrivateAddressSpace: %#v", availableIPv4PrivateAddressSpaces.RecommendedIpv4PrivateAddressSpace)

	// Call rpc, CreateCLADNet(ctx, cladnetSpec)
	reqCladnetSpec := &pb.CLADNetSpecification{
		Id:               "",
		Name:             cladnetName,
		Ipv4AddressSpace: availableIPv4PrivateAddressSpaces.RecommendedIpv4PrivateAddressSpace,
		Description:      cladnetDescription}

	cladnetSpec, err := cladnetClient.CreateCLADNet(ctx, reqCladnetSpec)
	if err != nil {
		log.Printf("Could not request: %v", err)
		return model.CLADNetSpecification{}, err
	}

	log.Printf("Struct: %#v", cladnetSpec)

	spec := model.CLADNetSpecification{
		ID:               cladnetSpec.Id,
		Name:             cladnetSpec.Name,
		Ipv4AddressSpace: cladnetSpec.Ipv4AddressSpace,
		Description:      cladnetSpec.Description,
	}
	return spec, nil
}

func main() {

	gRPCServiceEndpoint := "localhost:8089"
	dummyIPNetworks := []string{"10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16", "172.16.1.0/24", "172.16.2.0/24", "172.16.3.0/24", "172.16.4.0/24", "172.16.5.0/24", "172.16.6.0/24", "192.168.1.0/28", "192.168.2.0/28", "192.168.3.0/28", "192.168.4.0/28", "192.168.5.0/28", "192.168.6.0/28", "192.168.7.0/28"}

	cladnetName := "CLADNet01"
	cladnetDescription := "Alvin's CLADNet01"

	// Step 1: Create a cloud daptive network and get an ID of it.

	cladnetSpec, err := createProperCloudAdaptiveNetwork(gRPCServiceEndpoint, dummyIPNetworks, cladnetName, cladnetDescription)
	if err != nil {
		log.Printf("Could not create a cloud adaptive network: %v\n", err)
	}

	log.Printf("Struct: %#v\n", cladnetSpec)

	// Step 2: Deploy cb-network agent to hosts with ${ETCD_HOSTS}, ${CLADNet_ID}, and ${VMID}
	// An example step from a CB-Tumblebug script
	// 	BuildAndRunCBNetworkAgentCMD="wget https://raw.githubusercontent.com/cloud-barista/cb-larva/main/poc-cb-net/scripts/build-and-run-agent-in-the-background.sh -O ~/build-and-run-agent-in-the-background.sh;
	//		chmod +x ~/build-and-run-agent-in-the-background.sh;
	// 		~/build-and-run-agent-in-the-background.sh '${ETCD_HOSTS}' ${CLADNET_ID} ${VMID}"
	//
	//     echo "CMD: ${BuildAndRunCBNetworkAgentCMD}"
	//
	//     VAR1=$(curl -H "${AUTH}" -sX POST http://$TumblebugServer/tumblebug/ns/$NSID/cmd/mcis/$MCISID/vm/$VMID -H 'Content-Type: application/json' -d @- <<EOF
	//         {
	//         "command"        : "${BuildAndRunCBNetworkAgentCMD}"
	//         }
	// EOF
	//     )

}
