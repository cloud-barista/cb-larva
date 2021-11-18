package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/buger/jsonparser"
	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	"github.com/go-resty/resty/v2"
	"google.golang.org/grpc"
)

var config model.Config

func init() {
	fmt.Println("Start......... init() of test-client.go")
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exePath := filepath.Dir(ex)
	fmt.Printf("exePath: %v\n", exePath)

	// Load cb-network config from the current directory (usually for the production)
	configPath := filepath.Join(exePath, "config", "config.yaml")
	fmt.Printf("configPath: %v\n", configPath)
	if !file.Exists(configPath) {
		fmt.Printf("config.yaml doesn't exist at %v\n", configPath)
	}
	config, _ = model.LoadConfig(configPath)
	fmt.Printf("Load %v\n", configPath)
	fmt.Println("End......... init() of test-client.go")
}

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

	start := time.Now()

	// Dummy data to test
	gRPCServiceEndpoint := "localhost:8089"
	// dummyIPNetworks := []string{"10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16", "172.16.1.0/24", "172.16.2.0/24", "172.16.3.0/24", "172.16.4.0/24", "172.16.5.0/24", "172.16.6.0/24", "192.168.1.0/28", "192.168.2.0/28", "192.168.3.0/28", "192.168.4.0/28", "192.168.5.0/28", "192.168.6.0/28", "192.168.7.0/28"}

	cladnetName := "CLADNet01"
	cladnetDescription := "Alvin's CLADNet01"

	nsID := "ns01"
	mcisID := "mcis01"

	client := resty.New()
	client.SetBasicAuth("default", "default")

	// Step 1: Health-check CB-Tumblebug
	fmt.Println("\n\n##### Start ---------- Health-check CB-Tumblebug")
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Get("http://localhost:1323/tumblebug/health")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- Health-check CB-Tumblebug")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Step 2: Create MCIS by (POST ​/ns​/{nsId}​/mcisDynamic Create MCIS Dynamically)
	fmt.Println("\n\n##### Start ---------- Create MCIS")
	reqBody := `{
	"description": "Made in CB-TB",
	"installMonAgent": "no",
	"label": "custom tag",
	"name": "mcis01",
	"vm": [
	{
		"commonImage": "ubuntu18.04",
		"commonSpec": "aws-ap-northeast-2-t2-large"
	},
	{
		"commonImage": "ubuntu18.04",
		"commonSpec": "azure-westus-standard-b2s"
	},
	{
		"commonImage": "ubuntu18.04",
		"commonSpec": "gcp-asia-east1-e2-standard-2"
	},
	{
		"commonImage": "ubuntu18.04",
		"commonSpec": "alibaba-ap-northeast-1-ecs-t5-lc1m2-large"
	}
	]
}`

	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId": nsID,
		}).
		SetBody(reqBody).
		Post("http://localhost:1323/tumblebug/ns/{nsId}/mcisDynamic")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- Create MCIS")
	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

	// Step 3: Get VM address spaces
	fmt.Println("\n\n##### Start ---------- Get VM address spaces")
	data := []byte(resp.String())

	vNetIDs := []string{}

	jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		vNetID, _ := jsonparser.GetString(value, "vNetId")
		vNetIDs = append(vNetIDs, vNetID)
	}, "vm")

	fmt.Printf("vNetIDs: %#v\n", vNetIDs)

	ipNetsInMCIS := []string{}

	for _, v := range vNetIDs {

		// Get VNet
		// curl -X GET "http://localhost:1323/tumblebug/ns/ns01/resources/vNet/ns01-systemdefault-aws-ap-northeast-2" -H "accept: application/json"
		fmt.Printf("\nvNetId: %v\n", v)
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId":   nsID,
				"vNetId": v,
			}).
			Get("http://localhost:1323/tumblebug/ns/{nsId}/resources/vNet/{vNetId}")

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		fmt.Printf("Body: %v\n", resp)

		data := []byte(resp.String())

		ipNet, _ := jsonparser.GetString(data, "subnetInfoList", "[0]", "IPv4_CIDR")
		// trimmedIpNet := strings.Trim(ipNet, "\n")
		ipNetsInMCIS = append(ipNetsInMCIS, ipNet)
	}

	fmt.Printf("IPNetsInMCIS: %#v\n", ipNetsInMCIS)

	fmt.Println("##### End ---------- Get VM address spaces")
	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

	// Step 4: Create a cloud daptive network and get an ID of it.

	cladnetSpec, err := createProperCloudAdaptiveNetwork(gRPCServiceEndpoint, ipNetsInMCIS, cladnetName, cladnetDescription)
	if err != nil {
		log.Printf("Could not create a cloud adaptive network: %v\n", err)
	}

	log.Printf("Struct: %#v\n", cladnetSpec)

	// Step 5: Install the cb-network agent by sending a command to specified MCIS
	fmt.Println("\n\n##### Start ---------- Install the cb-network agent by sending a command to specified MCIS")

	placeHolderCommand := `wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/poc-cb-net/scripts/1.deploy-cb-network-agent.sh -O ~/1.deploy-cb-network-agent.sh; chmod +x ~/1.deploy-cb-network-agent.sh; source ~/1.deploy-cb-network-agent.sh '%s' %s`

	etcdEndpointsJSON, _ := json.Marshal(config.ETCD.Endpoints)
	etcdEndpointsString := string(etcdEndpointsJSON)
	fmt.Printf("etcdEndpointsString: %#v\n", etcdEndpointsString)
	additionalEncodedString := strings.Replace(etcdEndpointsString, "\"", "\\\"", -1)
	fmt.Printf("additionalEncodedString: %#v\n", additionalEncodedString)

	command := fmt.Sprintf(placeHolderCommand, additionalEncodedString, cladnetSpec.ID)
	fmt.Printf("command: %#v\n", command)

	placeHolderBody := `{"command": "%s", "userName": "cb-user"}`
	body := fmt.Sprintf(placeHolderBody, command)
	fmt.Printf("body: %#v\n", body)

	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId":   nsID,
			"mcisId": mcisID,
		}).
		SetBody(body).
		Post("http://localhost:1323/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- Install the cb-network agent by sending a command to specified MCIS")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

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

	elapsed := time.Since(start)
	fmt.Printf("Elapsed time: %s\n", elapsed)
}
