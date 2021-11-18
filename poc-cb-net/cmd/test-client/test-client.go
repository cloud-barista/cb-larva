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

	model "github.com/cloud-barista/cb-larva/poc-cb-net/internal/cb-network/model"
	"github.com/cloud-barista/cb-larva/poc-cb-net/internal/file"
	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
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

func deployKubernetesCluster(nsID string, mcisID string, vmID string) {

	fmt.Println("\n\n###################################")
	fmt.Println("## Welcome to this special stage ##")
	fmt.Println("###################################")

	fmt.Println("\n\nThis stage will try to deploy a single Kubernetes cluster across multi-clouds (simply on MCIS).")
	fmt.Println("\n\nHope you enjoy ;)")
	fmt.Println("Hope you enjoy ;)")
	fmt.Println("Hope you enjoy ;)")

	placeHolderBody := `{"command": "%s", "userName": "cb-user"}`

	// Special stage - Step 1: Setup environment and tools related to Kubernetes
	fmt.Println("\n\n##### Start ---------- Special stage - Step 1: Setup environment and tools related to Kubernetes")

	// Set command
	commandToSetupEnvironmentandTools := `wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/scripts/Kubernetes/1.setup-environment-and-tools.sh -O ~/1.setup-environment-and-tools.sh; chmod +x ~/1.setup-environment-and-tools.sh; ~/1.setup-environment-and-tools.sh`

	// Set request body
	body := fmt.Sprintf(placeHolderBody, commandToSetupEnvironmentandTools)
	fmt.Printf("body: %#v\n", body)

	client := resty.New()
	client.SetBasicAuth("default", "default")

	resp, err := client.R().
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

	fmt.Println("##### End ---------- Special stage - Step 1: Setup environment and tools related to Kubernetes")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage - Step 2: Reboot VMs to apply network configuration
	// curl -X GET "http://localhost:1323/tumblebug/ns/ns01/control/mcis/mcis01?action=refine" -H "accept: application/json"
	fmt.Println("\n\n##### Start ---------- Special stage - Step 2: Reboot VMs to apply network configuration")
	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId":   nsID,
			"mcisId": mcisID,
		}).
		SetQueryParams(map[string]string{
			"action": "reboot",
		}).
		Get("http://localhost:1323/tumblebug/ns/{nsId}/control/mcis/{mcisId}")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- Special stage - Step 2: Reboot VMs to apply network configuration")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage - Step 3: Retrieve the lead/master VM ID and all VM IDs
	// curl -X GET "http://localhost:1323/tumblebug/ns/ns01/mcis/mcis01?option=status" -H "accept: application/json"
	fmt.Println("\n\n##### Start ---------- Special stage - Step 3: Retrieve the lead/master VM ID and all VM IDs")

	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId":   nsID,
			"mcisId": mcisID,
		}).
		SetQueryParams(map[string]string{
			"option": "status",
		}).
		Get("http://localhost:1323/tumblebug/ns/{nsId}/mcis/{mcisId}")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	retMasterVMID := gjson.Get(resp.String(), "status.masterVmId")
	retVMIDs := gjson.Get(resp.String(), "status.vm.#.vNetId")
	fmt.Printf("retMasterVMID: %#v\n", retMasterVMID)
	fmt.Printf("retVMIDs: %#v\n", retVMIDs)

	fmt.Println("##### End ---------- Special stage - Step 3: Retrieve the lead/master VM ID and all VM IDs")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage - Step 4: Setup Kubernetes master
	// curl -X POST "http://localhost:1323/tumblebug/ns/ns01/cmd/mcis/mcis01/vm/vm01" -H "accept: application/json" -H "Content-Type:
	fmt.Println("\n\n##### Start ---------- Special stage - Step 4: Setup Kubernetes master")

	// Set command
	commandToSetupKubernetesMaster := `wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/scripts/Kubernetes/2.setup-K8s-master.sh -O ~/2.setup-K8s-master.sh; chmod +x ~/2.setup-K8s-master.sh; ~/2.setup-K8s-master.sh`

	// Set request body
	body2 := fmt.Sprintf(placeHolderBody, commandToSetupKubernetesMaster)
	fmt.Printf("body2: %#v\n", body2)

	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId":   nsID,
			"mcisId": mcisID,
			"vmId":   retMasterVMID.String(),
		}).
		SetBody(body2).
		Post("http://localhost:1323/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}/vm/{vmId}")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	// Parse Kubernetes join command
	respString := resp.String()
	startIndex := strings.LastIndex(respString, "kubeadm join")
	endIndex := strings.LastIndex(respString, " \"}")

	runes := []rune(respString)
	cmdKubernetesJoin := string(runes[startIndex:endIndex])

	fmt.Println("##### End ---------- Special stage - Step 4: Setup Kubernetes master")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage - Step 5: Join Kubernetes nodes to the Kubernetes master
	fmt.Println("\n\n##### Start ---------- Special stage - Step 5: Join Kubernetes nodes to the Kubernetes master")

	// Set request body
	body3 := fmt.Sprintf(placeHolderBody, cmdKubernetesJoin)
	fmt.Printf("body3: %#v\n", body3)
	for _, vmID := range retVMIDs.Array() {

		if retMasterVMID.String() == vmID.String() {
			continue
		}

		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId":   nsID,
				"mcisId": mcisID,
				"vmId":   vmID.String(),
			}).
			SetBody(body3).
			Post("http://localhost:1323/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}/vm/{vmId}")

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		fmt.Printf("Body: %v\n", resp)

	}

	fmt.Println("##### End ---------- Special stage - Step 5: Join Kubernetes nodes to the Kubernetes master")
	fmt.Println("Sleep 15 sec ( _ _ )zZ")
	time.Sleep(15 * time.Second)

	// Special stage - Step 6: Check status of Kubernetes cluster
	fmt.Println("\n\n##### Start ---------- Special stage - Step 6: Check status of Kubernetes cluster")

	// Set command
	commandToCheckStatusOfKubernetesCluster := `wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/scripts/Kubernetes/3.check-K8s-status.sh -O ~/3.check-K8s-status.sh; chmod +x ~/3.check-K8s-status.sh; ~/3.check-K8s-status.sh`

	// Set request body
	body4 := fmt.Sprintf(placeHolderBody, commandToCheckStatusOfKubernetesCluster)
	fmt.Printf("body4: %#v\n", body4)

	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId":   nsID,
			"mcisId": mcisID,
			"vmId":   retMasterVMID.String(),
		}).
		SetBody(body4).
		Post("http://localhost:1323/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}/vm/{vmId}")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- Special stage - Step 6: Check status of Kubernetes cluster")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

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
	fmt.Println("\n\n##### Start ---------- Step 1: Health-check CB-Tumblebug")
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Get("http://localhost:1323/tumblebug/health")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- Step 1: Health-check CB-Tumblebug")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Step 2: Create MCIS dynamically
	// POST ​/ns​/{nsId}​/mcisDynamic Create MCIS Dynamically
	fmt.Println("\n\n##### Start ---------- Step 2: Create MCIS dynamically")
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

	fmt.Println("##### End ---------- Step 2: Create MCIS dynamically")
	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

	// Step 3: Get VM address spaces
	fmt.Println("\n\n##### Start ----------  Step 3: Get VM address spaces")
	tbMCISInfo := resp

	vNetIDs := []string{}

	retVMID := gjson.Get(tbMCISInfo.String(), "vm.1.id")

	retVNetIDs := gjson.Get(tbMCISInfo.String(), "vm.#.vNetId")
	fmt.Printf("retVNetIDs: %#v\n", retVNetIDs)

	for _, vNetID := range retVNetIDs.Array() {
		vNetIDs = append(vNetIDs, vNetID.String())
	}
	fmt.Printf("vNetIds: %#v\n", vNetIDs)

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

		retIPv4CIDR := gjson.Get(resp.String(), "subnetInfoList.0.IPv4_CIDR")
		fmt.Printf("retIPv4CIDR: %#v\n", retIPv4CIDR)
		ipNetsInMCIS = append(ipNetsInMCIS, retIPv4CIDR.String())
	}

	fmt.Printf("IPNetsInMCIS: %#v\n", ipNetsInMCIS)

	fmt.Println("##### End ----------  Step 3: Get VM address spaces")
	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

	// Step 4: Create a cloud daptive network and get an ID of it
	fmt.Println("\n\n##### Start ---------- Step 4: Create a cloud daptive network and get an ID of it")

	cladnetSpec, err := createProperCloudAdaptiveNetwork(gRPCServiceEndpoint, ipNetsInMCIS, cladnetName, cladnetDescription)
	if err != nil {
		log.Printf("Could not create a cloud adaptive network: %v\n", err)
	}

	log.Printf("Struct: %#v\n", cladnetSpec)

	fmt.Println("##### End ---------- Step 4: Create a cloud daptive network and get an ID of it")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Step 5: Install the cb-network agent by sending a command to specified MCIS
	fmt.Println("\n\n##### Start ---------- Step 5: Install the cb-network agent by sending a command to specified MCIS")

	placeHolderCommand := `wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/poc-cb-net/scripts/1.deploy-cb-network-agent.sh -O ~/1.deploy-cb-network-agent.sh; chmod +x ~/1.deploy-cb-network-agent.sh; source ~/1.deploy-cb-network-agent.sh '%s' %s`

	etcdEndpointsJSON, _ := json.Marshal(config.ETCD.Endpoints)
	etcdEndpointsString := string(etcdEndpointsJSON)
	//fmt.Printf("etcdEndpointsString: %#v\n", etcdEndpointsString)
	additionalEncodedString := strings.Replace(etcdEndpointsString, "\"", "\\\"", -1)
	//fmt.Printf("additionalEncodedString: %#v\n", additionalEncodedString)

	command := fmt.Sprintf(placeHolderCommand, additionalEncodedString, cladnetSpec.ID)
	fmt.Printf("command: %#v\n", command)

	placeHolderBody := `{"command": "%s", "userName": "cb-user"}`
	body := fmt.Sprintf(placeHolderBody, command)
	//fmt.Printf("body: %#v\n", body)

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

	fmt.Println("##### End ---------- Step 5: Install the cb-network agent by sending a command to specified MCIS")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage ;)
	// Deploy a single Kubernetes cluster across multi-clouds (i.e., on MCIS)
	deployKubernetesCluster(nsID, mcisID, retVMID.String())

	elapsed := time.Since(start)
	fmt.Printf("Elapsed time: %s\n", elapsed)
}
