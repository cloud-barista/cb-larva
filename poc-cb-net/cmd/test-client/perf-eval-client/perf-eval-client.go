package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go"
	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
	cmdtype "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/command-type"
	etcdkey "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/etcd-key"
	"github.com/cloud-barista/cb-larva/poc-cb-net/pkg/file"
	ruletype "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/rule-type"
	cblog "github.com/cloud-barista/cb-log"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

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

var CBLogger *logrus.Logger
var config model.Config

func init() {
	fmt.Println("Start......... init() of admin-web.go")

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
		fmt.Printf("exe path: %v\n", exePath)

		logConfPath := filepath.Join(exePath, "config", "log_conf.yaml")
		if file.Exists(logConfPath) {
			fmt.Printf("path of log_conf.yaml: %v\n", logConfPath)
			CBLogger = cblog.GetLoggerWithConfigPath("cb-network", logConfPath)

		} else {
			// Load cb-log config from the project directory (usually for development)
			logConfPath = filepath.Join(exePath, "..", "..", "..", "config", "log_conf.yaml")
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
	fmt.Printf("exe path: %v\n", exePath)

	configPath := filepath.Join(exePath, "config", "config.yaml")
	if file.Exists(configPath) {
		fmt.Printf("path of config.yaml: %v\n", configPath)
		config, _ = model.LoadConfig(configPath)
	} else {
		// Load cb-network config from the project directory (usually for the development)
		configPath = filepath.Join(exePath, "..", "..", "..", "config", "config.yaml")

		if file.Exists(configPath) {
			config, _ = model.LoadConfig(configPath)
		} else {
			err := errors.New("fail to load config.yaml")
			panic(err)
		}
	}

	CBLogger.Debugf("Load %v", configPath)

	endpointNetworkService = config.Service.Endpoint
	endpointEtcd = config.ETCD.Endpoints

	fmt.Println("End......... init() of admin-web.go")
}

var endpointTB = "localhost:1323"
var nsID = "ns01"
var mcisID = "yk01perf"

var endpointNetworkService string

var endpointEtcd []string

func main() {

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	// A context for graceful shutdown (It is based on the signal package)
	// NOTE -
	// Use os.Interrupt Ctrl+C or Ctrl+Break on Windows
	// Use syscall.KILL for Kill(can't be caught or ignored) (POSIX)
	// Use syscall.SIGTERM for Termination (ANSI)
	// Use syscall.SIGINT for Terminal interrupt (ANSI)
	// Use syscall.SIGQUIT for Terminal quit (POSIX)
	// Use syscall.SIGHUP for Hangup (POSIX)
	// Use syscall.SIGABRT for Abort (POSIX)
	gracefulShutdownContext, stop := signal.NotifyContext(context.TODO(),
		os.Interrupt, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGABRT)
	defer stop()

	// nsID := "ns01"
	// mcisID := "perfcbnet01"

	// cladnetName := mcisID
	// cladnetDescription := "It's a recommended cladnet"

	wg.Add(1)
	go watchStatusInformation(gracefulShutdownContext, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	wg.Add(1)
	go watchHostNetworkInformation(gracefulShutdownContext, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	option := "-"

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		// Block until a signal is triggered
		<-gracefulShutdownContext.Done()

		option = "q"

		// Stop this cb-network agent
		fmt.Println("[Stop] Performance evaluation")

		// Wait for a while
		time.Sleep(1 * time.Second)
	}(&wg)

	fmt.Println("\n\n###################################")
	fmt.Println("## Ready to evaluate performance ##")
	fmt.Println("###################################")

	// wg.Add(1)
	// go doTest(gracefulShutdownContext, &wg)

	for option != "q" {

		printOptions()

		option = readOption()

		start := time.Now()

		handleOption(option)

		elapsed := time.Since(start)
		fmt.Printf("\nElapsed time: %s\n", elapsed)

		fmt.Println("Sleep 3 sec ( _ _ )zZ")
		time.Sleep(3 * time.Second)

	}
	// duration := 3 * time.Second
	// testInEvery(duration)

	stop()

	wg.Wait()
	fmt.Println("End test")
}

func watchStatusInformation(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// etcd Section
	// Connect to the etcd cluster
	etcdClient, etcdErr := clientv3.New(clientv3.Config{
		Endpoints:   endpointEtcd,
		DialTimeout: 5 * time.Second,
	})

	if etcdErr != nil {
		log.Println(etcdErr)
	}

	defer func() {
		errClose := etcdClient.Close()
		if errClose != nil {
			log.Printf("Can't close the etcd client (%v)\n", errClose)
		}
	}()

	log.Println("The etcdClient is connected.")

	// Watch "/registry/cloud-adaptive-network/status/information/{cladnet-id}/{host-id}"
	log.Printf("Start to watch \"%v\"", etcdkey.StatusInformation)
	watchChan1 := etcdClient.Watch(ctx, etcdkey.StatusInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			log.Printf("Watch - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			slicedKeys := strings.Split(string(event.Kv.Key), "/")
			parsedHostID := slicedKeys[len(slicedKeys)-1]
			log.Printf("ParsedHostID: %v", parsedHostID)

			status := string(event.Kv.Value)
			log.Printf("The status: %v", status)

			var networkStatus model.NetworkStatus

			err := json.Unmarshal(event.Kv.Value, &networkStatus)
			if err != nil {
				log.Println(err)
			}
			log.Println(networkStatus)
		}
	}
	log.Printf("End to watch \"%v\"", etcdkey.Status)
}

func watchHostNetworkInformation(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	// etcd Section
	// Connect to the etcd cluster
	etcdClient, etcdErr := clientv3.New(clientv3.Config{
		Endpoints:   endpointEtcd,
		DialTimeout: 5 * time.Second,
	})

	if etcdErr != nil {
		log.Println(etcdErr)
	}

	defer func() {
		errClose := etcdClient.Close()
		if errClose != nil {
			log.Printf("Can't close the etcd client (%v)\n", errClose)
		}
	}()

	log.Println("The etcdClient is connected.")

	// Watch "/registry/cloud-adaptive-network/host-network-information"
	log.Printf("Start to watch \"%v\"\n", etcdkey.HostNetworkInformation)

	watchChan2 := etcdClient.Watch(ctx, etcdkey.HostNetworkInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan2 {
		for _, event := range watchResponse.Events {
			switch event.Type {
			case mvccpb.PUT: // The watched value has changed.
				log.Printf("\nWatch - %s %q : %q\n", event.Type, event.Kv.Key, event.Kv.Value)

				var currentHostNetworkInformation model.HostNetworkInformation
				if err := json.Unmarshal(event.Kv.Value, &currentHostNetworkInformation); err != nil {
					log.Println(err)
				}
				log.Printf("Current host network information: %v\n", currentHostNetworkInformation)
				// curHostName := currentHostNetworkInformation.HostName
				curHostPublicIP := currentHostNetworkInformation.PublicIP

				// Find default host network interface and set IP and IPv4CIDR
				curHostIP, _, err := getDefaultInterfaceInfo(currentHostNetworkInformation.NetworkInterfaces)
				if err != nil {
					log.Println(err)
				}

				// Get the count of networking rule
				key := string(event.Kv.Key)
				CBLogger.Tracef("Key: %v", key)
				resp, respErr := etcdClient.Get(context.TODO(), key, clientv3.WithRev(event.Kv.ModRevision-1))
				if respErr != nil {
					CBLogger.Error(respErr)
				}

				var previousHostNetworkInformation model.HostNetworkInformation
				if err := json.Unmarshal(resp.Kvs[0].Value, &previousHostNetworkInformation); err != nil {
					log.Println(err)
				}
				log.Printf("Previous host network information: %v\n", previousHostNetworkInformation)
				prevHostName := previousHostNetworkInformation.HostName
				prevHostPublicIP := previousHostNetworkInformation.PublicIP

				// Find default host network interface and set IP and IPv4CIDR
				prevHostIP, _, err := getDefaultInterfaceInfo(previousHostNetworkInformation.NetworkInterfaces)
				if err != nil {
					log.Println(err)
				}

				if prevHostPublicIP != curHostPublicIP || prevHostIP != curHostIP {
					msg := fmt.Sprintf("%v, %v, %v, %v, %v", prevHostName, prevHostPublicIP, curHostPublicIP, prevHostIP, curHostIP)
					log.Println(msg)
				}

			case mvccpb.DELETE: // The watched key has been deleted.
				log.Printf("Watch - %s %q : %q\n", event.Type, event.Kv.Key, event.Kv.Value)
			default:
				log.Printf("Known event (%s), Key(%q), Value(%q)\n", event.Type, event.Kv.Key, event.Kv.Value)
			}
		}
	}
	log.Printf("End to watch \"%v\"\n", etcdkey.HostNetworkInformation)
}

func getDefaultInterfaceInfo(networkInterfaces []model.NetworkInterface) (ipAddr string, ipNet string, err error) {
	// Find default host network interface and set IP and IPv4CIDR

	for _, networkInterface := range networkInterfaces {
		if networkInterface.Name == "eth0" || networkInterface.Name == "ens4" || networkInterface.Name == "ens5" {
			return networkInterface.IPv4, networkInterface.IPv4CIDR, nil
		}
	}
	return "", "", errors.New("could not find default network interface")
}

func doTest(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	option := "-"

	for option != "q" {
		// NOTE - Default Selection
		// The default case in a select is run if no other case is ready.
		// Use a default case to try a send or receive without blocking:

		select {
		case <-ctx.Done():
			fmt.Println("Break the loop")
			return nil

		default:
			printOptions()

			option = readOption()

			start := time.Now()

			handleOption(option)

			elapsed := time.Since(start)
			fmt.Printf("Elapsed time: %s\n", elapsed)

			fmt.Println("End test")

		}

		// time.Sleep(100 * time.Millisecond)
	}

	return nil
}

// func doTestInOrder() {
// 	for
// }

// func testInEvery(duration time.Duration) error {

// 	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
// 	defer stop()

// 	for {
// 		// NOTE - Default Selection
// 		// The default case in a select is run if no other case is ready.
// 		// Use a default case to try a send or receive without blocking:

// 		select {
// 		case <-ctx.Done():
// 			fmt.Println("Break the loop")
// 			return nil
// 		case <-time.After(duration):
// 			fmt.Println("Start test")

// 			start := time.Now()
// 			printOptions()

// 			option := readOption()

// 			handleOption(option)

// 			elapsed := time.Since(start)
// 			fmt.Printf("Elapsed time: %s\n", elapsed)

// 			fmt.Println("End test")

// 			// default:
// 			// 	fmt.Print(".")
// 		}

// 		fmt.Println("Bottom of for loop")
// 		// time.Sleep(100 * time.Millisecond)
// 	}
// }

func printOptions() {
	fmt.Printf("\n%s[Usage] Select a option: %s\n", string(colorYellow), string(colorReset))
	fmt.Println("    - 1. Check CB-Tumblebug Health")
	fmt.Println("    - 2. Resume MCIS")
	fmt.Println("    - 3. Test Performance (RuleType: basic, Encryption: disabled)")
	fmt.Println("    - 4. Test Performance (RuleType: basic, Encryption: enabled)")
	fmt.Println("    - 5. Test Performance (RuleType: cost-prioritized, Encryption: disabled)")
	fmt.Println("    - 6. Test Performance (RuleType: cost-prioritized, Encryption: enabled)")
	fmt.Println("    - 7. Suspend MCIS")
	fmt.Println("    - 8. Set RuleType(basic)")
	fmt.Println("    - 'q'(Q).  to quit")
}

func readOption() string {
	fmt.Print(">> ")

	var line string
	// fmt.Scanln(&input)

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		line = scanner.Text()
	}
	return line
}

func handleOption(option string) {

	fmt.Printf("Option: %v\n", option)
	switch option {
	case "1":
		checkTumblebugHealth()

	case "2":
		resumeMCIS()

	case "3":
		testPerformance(ruletype.Basic, cmdtype.DisableEncryption)

	case "4":
		testPerformance(ruletype.Basic, cmdtype.EnableEncryption)

	case "5":
		testPerformance(ruletype.CostPrioritized, cmdtype.DisableEncryption)

	case "6":
		testPerformance(ruletype.CostPrioritized, cmdtype.EnableEncryption)

	case "7":
		suspendMCIS()

	case "8":
		setRuleType(ruletype.Basic)

	case "q":
		fmt.Println("q(Q). xxx")

	default:
		fmt.Println("default")

	}
}

func checkTumblebugHealth() {
	fmt.Println("\n\n##### Start ---------- checkTumblebugHealth()")

	client := resty.New()
	client.SetBasicAuth("default", "default")

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Get(fmt.Sprintf("http://%s/tumblebug/health", endpointTB))

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- checkTumblebugHealth()")
}

func resumeMCIS() {
	fmt.Println("\n\n##### Start ---------- resumeMCIS()")

	client := resty.New()
	client.SetBasicAuth("default", "default")

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId":   nsID,
			"mcisId": mcisID,
		}).
		SetQueryParams(map[string]string{
			"action": "resume",
		}).
		Get(fmt.Sprintf("http://%s/tumblebug/ns/{nsId}/control/mcis/{mcisId}", endpointTB))

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
		Get(fmt.Sprintf("http://%s/tumblebug/ns/{nsId}/mcis/{mcisId}", endpointTB))

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	mcisStatus := gjson.Get(resp.String(), "status.status")
	fmt.Printf("=====> status: %#v\n", mcisStatus.String())

	fmt.Println("\n\n##### End ---------- resumeMCIS()")
}

func suspendMCIS() {
	fmt.Println("\n\n##### Start ---------- suspendMCIS()")

	client := resty.New()
	client.SetBasicAuth("default", "default")

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId":   nsID,
			"mcisId": mcisID,
		}).
		SetQueryParams(map[string]string{
			"action": "suspend",
		}).
		Get(fmt.Sprintf("http://%s/tumblebug/ns/{nsId}/control/mcis/{mcisId}", endpointTB))

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	// resp, err = client.R().
	// 	SetHeader("Content-Type", "application/json").
	// 	SetHeader("Accept", "application/json").
	// 	SetPathParams(map[string]string{
	// 		"nsId":   nsID,
	// 		"mcisId": mcisID,
	// 	}).
	// 	SetQueryParams(map[string]string{
	// 		"option": "status",
	// 	}).
	// 	Get(fmt.Sprintf("http://%s/tumblebug/ns/{nsId}/mcis/{mcisId}", endpointTB))

	// // Output print
	// fmt.Printf("\nError: %v\n", err)
	// fmt.Printf("Time: %v\n", resp.Time())
	// fmt.Printf("Body: %v\n", resp)

	// mcisStatus := gjson.Get(resp.String(), "status.status")
	// fmt.Printf("=====> status: %#v\n", mcisStatus.String())

	fmt.Println("\n\n##### End ---------- suspendMCIS()")
}

func testPerformance(ruleType string, encryptionCommand string) {
	fmt.Println("\n\n##### Start ---------- testPerformance()")

	trialCount := 10

	// // Create a context
	// ctx, cancel := context.WithTimeout(context.TODO(), 10*time.Second)
	// defer cancel()

	//// Initialize cb-network service
	// Connect to the gRPC server
	// Register CloudAdaptiveNetwork handler to gwmux
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	grpcConn, err := grpc.Dial(endpointNetworkService, options...)
	if err != nil {
		log.Printf("Cannot connect: %v\n", err)
		// return model.CLADNetSpecification{}, err
	}
	defer grpcConn.Close()

	// Create stubs of cb-network service
	// cladnetClient := pb.NewCloudAdaptiveNetworkServiceClient(grpcConn)
	systemClient := pb.NewSystemManagementServiceClient(grpcConn)

	if ruleType == ruletype.CostPrioritized {
		client := resty.New()
		client.SetBasicAuth("default", "default")

		placeHolder := `{"etcdEndpoints": %s, "serviceEndpoint": "%s"}`

		endpointEtcdJson, err := json.Marshal(endpointEtcd)
		if err != nil {
			fmt.Println(err)
		}
		body := fmt.Sprintf(placeHolder, endpointEtcdJson, endpointNetworkService)

		resp, err := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId":   nsID,
				"mcisId": mcisID,
			}).
			SetBody(body).
			Put(fmt.Sprintf("http://%s/tumblebug/ns/{nsId}/network/mcis/{mcisId}", endpointTB))

		// Output print
		fmt.Printf("\nError: %v\n", err)
		fmt.Printf("Time: %v\n", resp.Time())
		fmt.Printf("Body: %v\n", resp)
	}

	//// Set end-to-end encryption
	controlRequest := &pb.ControlRequest{
		CladnetId:   mcisID,
		CommandType: pb.CommandType(pb.CommandType_value[encryptionCommand]),
	}

	controlRes, err := systemClient.ControlCloudAdaptiveNetwork(context.TODO(), controlRequest)
	if err != nil {
		log.Println(err)
	}
	log.Printf("Control response: %v\n", controlRes)

	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

	// Request test
	tempSpec, err := json.Marshal(model.TestSpecification{
		CladnetID:  mcisID,
		TrialCount: trialCount,
	})
	log.Printf("Test specification: %v\n", tempSpec)

	if err != nil {
		log.Println(err)
	}
	testSpec := string(tempSpec)

	testRequest := &pb.TestRequest{
		CladnetId: mcisID,
		TestType:  pb.TestType_CONNECTIVITY,
		TestSpec:  testSpec,
	}

	testRes, err := systemClient.TestCloudAdaptiveNetwork(context.TODO(), testRequest)
	if err != nil {
		log.Println(err)
	}
	log.Printf("Test response: %v\n", testRes)

	sleepTime := trialCount + 5
	fmt.Printf("Sleep %d sec ( _ _ )zZ\n", sleepTime)
	time.Sleep(time.Duration(sleepTime) * time.Second)

	fmt.Println("\n\n##### End ---------- testPerformance()")
}

func setRuleType(ruleType string) {

	fmt.Println("\n\n##### Start ---------- setRuleType()")

	// Create a context
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	//// Initialize cb-network service
	// Connect to the gRPC server
	// Register CloudAdaptiveNetwork handler to gwmux
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	grpcConn, err := grpc.Dial(endpointNetworkService, options...)
	if err != nil {
		log.Printf("Cannot connect: %v\n", err)
		// return model.CLADNetSpecification{}, err
	}
	defer grpcConn.Close()

	// Create stubs of cb-network service
	cladnetClient := pb.NewCloudAdaptiveNetworkServiceClient(grpcConn)

	//// Set rule type
	// Get the specification of a Cloud Adaptive Network
	cladnetRequest := &pb.CLADNetRequest{
		CladnetId: mcisID,
	}

	cladnetSpec, err := cladnetClient.GetCLADNet(ctx, cladnetRequest)
	if err != nil {
		log.Println(err)
	}

	// Assign rule
	cladnetSpec.RuleType = ruleType

	// Update the specification
	updatedCladnetSpec, err := cladnetClient.UpdateCLADNet(ctx, cladnetSpec)
	if err != nil {
		log.Println(err)
	}
	log.Printf("Update cladnet spec: %v\n", updatedCladnetSpec)

	fmt.Println("\n\n##### End ---------- setRuleType()")
}
