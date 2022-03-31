package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	nethelper "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/network-helper"
	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
)

var config model.Config
var endpointTB = "http://localhost:1323"
var placeHolderBody = `{"command": "%s", "userName": "cb-user"}`

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
)

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

func main() {

	start := time.Now()

	// Dummy data to test
	gRPCServiceEndpoint := "localhost:8053"
	// dummyIPNetworks := []string{"10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16", "172.16.1.0/24", "172.16.2.0/24", "172.16.3.0/24", "172.16.4.0/24", "172.16.5.0/24", "172.16.6.0/24", "192.168.1.0/28", "192.168.2.0/28", "192.168.3.0/28", "192.168.4.0/28", "192.168.5.0/28", "192.168.6.0/28", "192.168.7.0/28"}

	cladnetName := "CLADNet01"
	cladnetDescription := "Alvin's CLADNet01"

	nsID := "ns01"
	mcisID := "cbnet01"

	client := resty.New()
	client.SetBasicAuth("default", "default")

	isOn := true

	// Step 1: Health-check CB-Tumblebug
	fmt.Println("\n\n##### Start ---------- Step 1: Health-check CB-Tumblebug")
	pressEnterKeyToContinue(isOn)

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Get(fmt.Sprintf("%s/tumblebug/health", endpointTB))

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
	pressEnterKeyToContinue(isOn)

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
		Post(fmt.Sprintf("%s/tumblebug/ns/{nsId}/mcisDynamic", endpointTB))

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	mcisStatus := gjson.Get(resp.String(), "status")
	fmt.Printf("=====> status: %v \n", mcisStatus)

	fmt.Println("##### End ---------- Step 2: Create MCIS dynamically")
	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

	// Step 3: Get VM address spaces
	fmt.Println("\n\n##### Start ----------  Step 3: Get VM address spaces")
	pressEnterKeyToContinue(isOn)

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
			Get(fmt.Sprintf("%s/tumblebug/ns/{nsId}/resources/vNet/{vNetId}", endpointTB))

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
	fmt.Println("Sleep 3 sec ( _ _ )zZ")
	time.Sleep(3 * time.Second)

	// Step 4: Create a cloud adaptive network and get an ID of it
	fmt.Println("\n\n##### Start ---------- Step 4: Create a cloud adaptive network and get an ID of it")
	pressEnterKeyToContinue(isOn)

	cladnetSpec, err := createProperCloudAdaptiveNetwork(gRPCServiceEndpoint, ipNetsInMCIS, cladnetName, cladnetDescription)
	if err != nil {
		log.Printf("Could not create a cloud adaptive network: %v\n", err)
	}

	log.Printf("Struct: %#v\n", cladnetSpec)

	fmt.Println("##### End ---------- Step 4: Create a cloud adaptive network and get an ID of it")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Step 5: Install the cb-network agent by sending a command to specified MCIS
	fmt.Println("\n\n##### Start ---------- Step 5: Install the cb-network agent by sending a command to specified MCIS")
	pressEnterKeyToContinue(isOn)

	placeHolderCommand := `wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/poc-cb-net/scripts/1.deploy-cb-network-agent.sh -O ~/1.deploy-cb-network-agent.sh; chmod +x ~/1.deploy-cb-network-agent.sh; source ~/1.deploy-cb-network-agent.sh '%s' %s`

	etcdEndpointsJSON, _ := json.Marshal(config.ETCD.Endpoints)
	etcdEndpointsString := string(etcdEndpointsJSON)
	//fmt.Printf("etcdEndpointsString: %#v\n", etcdEndpointsString)
	additionalEncodedString := strings.Replace(etcdEndpointsString, "\"", "\\\"", -1)
	//fmt.Printf("additionalEncodedString: %#v\n", additionalEncodedString)

	command := fmt.Sprintf(placeHolderCommand, additionalEncodedString, cladnetSpec.ID)
	fmt.Printf("command: %#v\n", command)

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
		Post(fmt.Sprintf("%s/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}", endpointTB))

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- Step 5: Install the cb-network agent by sending a command to specified MCIS")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage ;)
	char := askYesOrNoQuestion(isOn, "Do you want to deploy a single Kubernetes cluster across multi-clouds? (y/n)")
	if char == "y" {
		// Deploy a single Kubernetes cluster across multi-clouds (i.e., on MCIS)
		deployKubernetesCluster(nsID, mcisID, retVMID.String())
	}

	char = askYesOrNoQuestion(isOn, "Do you want to clean MCIS? (y/n)")
	if char == "y" {
		// Step 6: Delete MCIS
		// curl -X DELETE "http://localhost:1323/tumblebug/ns/ns01/mcis/mcis01?option=terminate" -H "accept: application/json"
		fmt.Println("\n\n##### Start ---------- Step 6: Delete MCIS")
		pressEnterKeyToContinue(isOn)

		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId":   nsID,
				"mcisId": mcisID,
			}).
			SetQueryParams(map[string]string{
				"option": "terminate",
			}).
			Delete(fmt.Sprintf("%s/tumblebug/ns/{nsId}/mcis/{mcisId}", endpointTB))

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		fmt.Printf("Body: %v\n", resp)

		fmt.Println("##### End ---------- Step 6: Delete MCIS")
		fmt.Println("Sleep 5 sec ( _ _ )zZ")
		time.Sleep(5 * time.Second)

		// Step 7: Delete defaultResources
		// curl -X DELETE "http://localhost:1323/tumblebug/ns/ns01/defaultResources" -H "accept: application/json"
		fmt.Println("\n\n##### Start ---------- Step 7: Delete defaultResources")
		pressEnterKeyToContinue(isOn)

		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId": nsID,
			}).
			Delete(fmt.Sprintf("%s/tumblebug/ns/{nsId}/defaultResources", endpointTB))

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		fmt.Printf("Body: %v\n", resp)

		fmt.Println("##### End ---------- Step 7: Delete defaultResources")
	}

	elapsed := time.Since(start)
	fmt.Printf("Elapsed time: %s\n", elapsed)
}

func pressEnterKeyToContinue(isOn bool) {

	if isOn {
		fmt.Printf("%s##### Press the 'ENTER' key to continue...%s\n", string(colorYellow), string(colorReset))
		var input string
		fmt.Scanln(&input)
	}
}

func insertK8sCommands(isOn bool) string {
	if isOn {
		fmt.Printf("\n\n%s[Usage] Select a shortcut command below or Enter manual 'kubectl' commands%s\n", string(colorYellow), string(colorReset))
		fmt.Println("    - Insert '1' for 'kubectl get nodes'")
		fmt.Println("    - Insert '2' for 'kubectl get pods --namespace kube-system'")
		fmt.Println("    - Insert '3' for 'kubectl get pods'")
		fmt.Println("    - Insert '4' for 'kubectl get services'")
		fmt.Println("    - Insert '5' for 'deploying Kubernetes dashboard'")
		fmt.Println("    - Insert '6' for 'deploying Guestbook application'")
		fmt.Println("    - Insert '7' for 'deploying Weave Scope'")
		fmt.Println("    - Insert 'q'(Q) to quit")
		fmt.Println("    - Insert 'your kubectl commands'")
		fmt.Print(">> ")

		var line string
		// fmt.Scanln(&input)

		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			line = scanner.Text()
		}
		return line
	}
	return ""
}

func askYesOrNoQuestion(isOn bool, question string) string {
	if isOn {
		for {
			var input string

			fmt.Printf("\n\n%s%s%s\n", string(colorYellow), question, string(colorReset))
			fmt.Print(">> ")
			fmt.Scanln(&input)

			lowerString := strings.ToLower(input)

			if lowerString == "y" || lowerString == "n" {
				return lowerString
			}
		}
	}
	return ""
}

func createProperCloudAdaptiveNetwork(gRPCServiceEndpoint string, ipNetworks []string, cladnetName string, cladnetDescription string) (model.CLADNetSpecification, error) {

	var cladnetSpec *pb.CLADNetSpecification
	ipNets := &pb.IPNetworks{IpNetworks: ipNetworks}
	var spec model.CLADNetSpecification

	log.Printf("Service Call Method: %s", config.ServiceCallMethod)
	switch config.ServiceCallMethod {
	case "grpc":

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

		cladnetSpec, err = cladnetClient.CreateCLADNet(ctx, reqCladnetSpec)
		if err != nil {
			log.Printf("Could not request: %v", err)
			return model.CLADNetSpecification{}, err
		}

		log.Printf("Struct: %#v", cladnetSpec)

		spec = model.CLADNetSpecification{
			ID:               cladnetSpec.Id,
			Name:             cladnetSpec.Name,
			Ipv4AddressSpace: cladnetSpec.Ipv4AddressSpace,
			Description:      cladnetSpec.Description,
		}
	case "rest":

		ipNetworksJson, _ := json.Marshal(ipNets)
		fmt.Println(string(ipNetworksJson))

		client := resty.New()
		// client.SetBasicAuth("default", "default")

		// Request a recommendation of available IPv4 private address spaces.
		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetBody(ipNetworksJson).
			Post(fmt.Sprintf("http://%s/v1/cladnet/available-ipv4-address-spaces", gRPCServiceEndpoint))
		// Output print
		log.Printf("\nError: %v\n", err)
		log.Printf("Time: %v\n", resp.Time())
		log.Printf("Body: %v\n", resp)

		if err != nil {
			log.Printf("Could not request: %v\n", err)
			return model.CLADNetSpecification{}, err
		}

		var availableIPv4PrivateAddressSpaces nethelper.AvailableIPv4PrivateAddressSpaces

		json.Unmarshal(resp.Body(), &availableIPv4PrivateAddressSpaces)
		log.Printf("%+v\n", availableIPv4PrivateAddressSpaces)
		log.Printf("RecommendedIpv4PrivateAddressSpace: %#v", availableIPv4PrivateAddressSpaces.RecommendedIPv4PrivateAddressSpace)

		reqCladnetSpec := &pb.CLADNetSpecification{
			Id:               "",
			Name:             cladnetName,
			Ipv4AddressSpace: availableIPv4PrivateAddressSpaces.RecommendedIPv4PrivateAddressSpace,
			Description:      cladnetDescription}

		reqCladnetSpecJson, _ := json.Marshal(reqCladnetSpec)
		fmt.Println(string(reqCladnetSpecJson))

		// Request to create a Cloud Adaptive Network
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetBody(reqCladnetSpecJson).
			Post(fmt.Sprintf("http://%s/v1/cladnet", gRPCServiceEndpoint))
		// Output print
		log.Printf("\nError: %v\n", err)
		log.Printf("Time: %v\n", resp.Time())
		log.Printf("Body: %v\n", resp)

		if err != nil {
			log.Printf("Could not request: %v\n", err)
			return model.CLADNetSpecification{}, err
		}

		json.Unmarshal(resp.Body(), &spec)
		log.Printf("%#v\n", spec)

	default:
		log.Printf("Unknown service call method: %v\n", config.ServiceCallMethod)
	}

	return spec, nil
}

func interactWithMasterVM(nsID string, mcisID string, masterVMID string) {
	isOn := true
	client := resty.New()
	client.SetBasicAuth("default", "default")

StatusCheckK8s:
	for {
		inputCommand := insertK8sCommands(isOn)

		fmt.Println(inputCommand)

		cmd := ""
		switch inputCommand {
		case "1":
			cmd = "kubectl get nodes"
		case "2":
			cmd = "kubectl get pods --namespace kube-system"
		case "3":
			cmd = "kubectl get pods"
		case "4":
			cmd = "kubectl get services"
		case "5":
			cmd = "wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/scripts/Kubernetes/4.deploy-and-access-Kubernetes-dashboard.sh -O ~/4.deploy-and-access-Kubernetes-dashboard.sh; chmod +x ~/4.deploy-and-access-Kubernetes-dashboard.sh; ~/4.deploy-and-access-Kubernetes-dashboard.sh"
		case "6":
			cmd = "wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/scripts/Kubernetes/5.deploy-and-access-Guestbook.sh -O ~/5.deploy-and-access-Guestbook.sh; chmod +x ~/5.deploy-and-access-Guestbook.sh; ~/5.deploy-and-access-Guestbook.sh"
		case "7":
			cmd = "wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/scripts/Kubernetes/6.deploy-and-access-Weave-Scope.sh -O ~/6.deploy-and-access-Weave-Scope.sh; chmod +x ~/6.deploy-and-access-Weave-Scope.sh; ~/6.deploy-and-access-Weave-Scope.sh"
		case "q":
			break StatusCheckK8s
		case "":
			continue
		default:
			cmd = inputCommand
		}

		// Set request body
		body := fmt.Sprintf(placeHolderBody, cmd)
		fmt.Printf("body4: %#v\n", body)

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId":   nsID,
				"mcisId": mcisID,
				"vmId":   masterVMID,
			}).
			SetBody(body).
			Post(fmt.Sprintf("%s/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}/vm/{vmId}", endpointTB))

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		// fmt.Printf("Body: %v\n", resp)

		ret := gjson.Get(resp.String(), "result")
		fmt.Println("[Result]")
		fmt.Println(ret)
	}
}

func deployKubernetesCluster(nsID string, mcisID string, vmID string) {

	fmt.Println("\n\n###################################")
	fmt.Println("## Welcome to this special stage ##")
	fmt.Println("###################################")

	fmt.Println("\n\nThis stage will try to deploy a single Kubernetes cluster across multi-clouds (simply on MCIS).")
	fmt.Println("\n\nHope you enjoy ;)")
	fmt.Println("Hope you enjoy ;)")
	fmt.Println("Hope you enjoy ;)")

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	// Enable 'Press the ENTER key to continue...'
	isOn := true

	client := resty.New()
	client.SetBasicAuth("default", "default")

	// Special stage - Step 1: Retrieve the lead/master VM ID and all VM IDs
	// curl -X GET "http://localhost:1323/tumblebug/ns/ns01/mcis/mcis01?option=status" -H "accept: application/json"
	fmt.Println("\n\n##### Start ---------- Special stage - Step 1: Retrieve the lead/master VM ID and all VM IDs")
	pressEnterKeyToContinue(isOn)

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId":   nsID,
			"mcisId": mcisID,
		}).
		SetQueryParams(map[string]string{
			"option": "status",
		}).
		Get(fmt.Sprintf("%s/tumblebug/ns/{nsId}/mcis/{mcisId}", endpointTB))

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	retMasterVMID := gjson.Get(resp.String(), "status.masterVmId")
	retVMIDs := gjson.Get(resp.String(), "status.vm.#.id")
	fmt.Printf("retMasterVMID: %#v\n", retMasterVMID.String())
	fmt.Printf("retVMIDs: %#v\n", retVMIDs.String())

	fmt.Println("##### End ---------- Special stage - Step 1: Retrieve the lead/master VM ID and all VM IDs")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage - Step 2: Setup environment and tools related to Kubernetes
	fmt.Println("\n\n##### Start ---------- Special stage - Step 2: Setup environment and tools related to Kubernetes")
	pressEnterKeyToContinue(isOn)

	// Set command
	commandToSetupEnvironmentandTools := `wget https://raw.githubusercontent.com/cloud-barista/cb-larva/develop/scripts/Kubernetes/1.setup-environment-and-tools.sh -O ~/1.setup-environment-and-tools.sh; chmod +x ~/1.setup-environment-and-tools.sh; ~/1.setup-environment-and-tools.sh`

	// Set request body
	body := fmt.Sprintf(placeHolderBody, commandToSetupEnvironmentandTools)
	fmt.Printf("body: %#v\n", body)

	// Setup in parallel
	for _, vmID := range retVMIDs.Array() {
		wg.Add(1)
		go func(wg *sync.WaitGroup, vmID string) {
			defer wg.Done()

			respEach, errEach := client.R().
				SetHeader("Content-Type", "application/json").
				SetHeader("Accept", "application/json").
				SetPathParams(map[string]string{
					"nsId":   nsID,
					"mcisId": mcisID,
					"vmId":   vmID,
				}).
				SetBody(body).
				Post(fmt.Sprintf("%s/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}/vm/{vmId}", endpointTB))

			// Output print
			fmt.Printf("\nError: %v\n", errEach)
			fmt.Printf("Time: %v\n", respEach.Time())
			// fmt.Printf("Body: %v\n", respEach)
			ret := gjson.Get(respEach.String(), "result")
			fmt.Println("[Result]")
			fmt.Println(ret)
			fmt.Printf("Done to setup on VM - '%s'\n", vmID)

		}(&wg, vmID.String())
		time.Sleep(10 * time.Millisecond)
	}
	wg.Wait()

	fmt.Println("##### End ---------- Special stage - Step 2: Setup environment and tools related to Kubernetes")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage - Step 3: Reboot VMs to apply network configuration
	// curl -X GET "http://localhost:1323/tumblebug/ns/ns01/control/mcis/mcis01?action=reboot" -H "accept: application/json"
	fmt.Println("\n\n##### Start ---------- Special stage - Step 3: Reboot VMs to apply network configuration")
	pressEnterKeyToContinue(isOn)

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
		Get(fmt.Sprintf("%s/tumblebug/ns/{nsId}/control/mcis/{mcisId}", endpointTB))

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

	// after rebooting, check MCIS status
	for {
		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId":   nsID,
				"mcisId": mcisID,
			}).
			SetQueryParams(map[string]string{
				"action": "resume",
			}).
			Get(fmt.Sprintf("%s/tumblebug/ns/{nsId}/control/mcis/{mcisId}", endpointTB))

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		fmt.Printf("Body: %v\n", resp)

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
			Get(fmt.Sprintf("%s/tumblebug/ns/{nsId}/mcis/{mcisId}", endpointTB))

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		fmt.Printf("Body: %v\n", resp)

		mcisStatus := gjson.Get(resp.String(), "status.status")
		fmt.Printf("=====> status: %#v\n", mcisStatus.String())

		char := askYesOrNoQuestion(isOn, "Do you want to check MCIS status again? (y/n)")
		if char == "n" {
			break
		}
	}

	fmt.Println("##### End ---------- Special stage - Step 3: Reboot VMs to apply network configuration")
	fmt.Println("Sleep 3 sec ( _ _ )zZ")
	time.Sleep(3 * time.Second)

	// Special stage - Step 4: Setup Kubernetes master
	// curl -X POST "http://localhost:1323/tumblebug/ns/ns01/cmd/mcis/mcis01/vm/vm01" -H "accept: application/json" -H "Content-Type:
	fmt.Println("\n\n##### Start ---------- Special stage - Step 4: Setup Kubernetes master")
	pressEnterKeyToContinue(isOn)

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
		Post(fmt.Sprintf("%s/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}/vm/{vmId}", endpointTB))

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	// fmt.Printf("Body: %v\n", resp)

	ret := gjson.Get(resp.String(), "result")
	fmt.Println("[Result]")
	fmt.Println(ret.String())

	// Parse Kubernetes join command
	respString := resp.String()
	startIndex := strings.LastIndex(respString, "kubeadm join")
	endIndex := strings.LastIndex(respString, " \"}")

	runes := []rune(respString)
	cmdKubeadmJoin := string(runes[startIndex:endIndex])
	fmt.Printf("cmdKubeadmJoin: %#v\n", cmdKubeadmJoin)

	fmt.Println("##### End ---------- Special stage - Step 4: Setup Kubernetes master")
	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

	// Special stage - Step 5: Join Kubernetes nodes to the Kubernetes master
	fmt.Println("\n\n##### Start ---------- Special stage - Step 5: Join Kubernetes nodes to the Kubernetes master")
	pressEnterKeyToContinue(isOn)

	// Concatenate "sudo" to run the kubeadm join command as the root user
	cmdSudoKubeadmJoin := fmt.Sprintf("%s %s", "sudo", cmdKubeadmJoin)
	fmt.Printf("cmdSudoKubeadmJoin: %#v\n", cmdSudoKubeadmJoin)

	// Set request body
	body3 := fmt.Sprintf(placeHolderBody, cmdSudoKubeadmJoin)
	fmt.Printf("body3: %#v\n", body3)

	// Join in parallel
	for _, vmID := range retVMIDs.Array() {

		if retMasterVMID.String() == vmID.String() {
			continue
		}

		wg.Add(1)
		go func(wg *sync.WaitGroup, vmID string) {
			defer wg.Done()

			resp, err = client.R().
				SetHeader("Content-Type", "application/json").
				SetHeader("Accept", "application/json").
				SetPathParams(map[string]string{
					"nsId":   nsID,
					"mcisId": mcisID,
					"vmId":   vmID,
				}).
				SetBody(body3).
				Post(fmt.Sprintf("%s/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}/vm/{vmId}", endpointTB))

			fmt.Println("\nTarget Kubernetes node (VM) to join: ", vmID)
			// Output print
			fmt.Printf("Error: %v\n", err)
			fmt.Printf("Time: %v\n", resp.Time())
			fmt.Printf("Body: %v\n", resp)

		}(&wg, vmID.String())
		time.Sleep(1 * time.Second)
	}
	wg.Wait()

	fmt.Println("##### End ---------- Special stage - Step 5: Join Kubernetes nodes to the Kubernetes master")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage - Step 6: Interact with Kubernetes master
	fmt.Println("\n\n##### Start ---------- Special stage - Step 6: Interact with Kubernetes master")
	pressEnterKeyToContinue(isOn)

	fmt.Printf("retMasterVMID: %s\n", retMasterVMID.String())

	interactWithMasterVM(nsID, mcisID, retMasterVMID.String())

	fmt.Println("##### End ---------- Special stage - Step 6: Interact with Kubernetes master")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Special stage - Step 7: Randomly pick and reboot a Kubernetes node
	fmt.Println("\n\n##### Start ---------- Special stage - Step 7: Randomly pick and reboot a Kubernetes node")

	for {
		char := askYesOrNoQuestion(isOn, "A randomly picked Kubernetes node will be suspended, waited, and resumed. Do you want to proceed ? (y/n)")

		if char == "n" {
			break
		}

		var randomlyPickedVM string
		rand.Seed(time.Now().UnixNano())
		for {
			index := rand.Intn(len(retVMIDs.Array()))
			randomlyPickedVM = retVMIDs.Array()[index].String()
			if retMasterVMID.String() == randomlyPickedVM {
				continue
			} else {
				break
			}
		}
		fmt.Printf("MasterVM: %s\n", retMasterVMID.String())
		fmt.Printf("VM (%s) is selected.\n", randomlyPickedVM)

		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId":   nsID,
				"mcisId": mcisID,
				"vmId":   randomlyPickedVM,
			}).
			SetQueryParams(map[string]string{
				"action": "suspend",
			}).
			Get(fmt.Sprintf("%s/tumblebug/ns/{nsId}/control/mcis/{mcisId}/vm/{vmId}", endpointTB))

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		fmt.Printf("Body: %v\n", resp)

		fmt.Println("Sleep 20 sec ( _ _ )zZ")
		time.Sleep(20 * time.Second)

		resp, err = client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId":   nsID,
				"mcisId": mcisID,
				"vmId":   randomlyPickedVM,
			}).
			SetQueryParams(map[string]string{
				"action": "resume",
			}).
			Get(fmt.Sprintf("%s/tumblebug/ns/{nsId}/control/mcis/{mcisId}/vm/{vmId}", endpointTB))

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		fmt.Printf("Body: %v\n", resp)
	}

	fmt.Println("##### End ---------- Special stage - Step 7: Randomly pick and reboot a Kubernetes node")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

}
