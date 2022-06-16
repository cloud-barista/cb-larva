package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
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

// CBLogger represents a logger to show execution processes according to the logging level.
var CBLogger *logrus.Logger

// CB-Tumblebug
var endpointTB = "localhost:1323"
var nsID = "ns01"
var mcisID = "clad00perf"

// The cb-network system
var config model.Config
var endpointNetworkService string
var endpointEtcd []string
var timeoutToCheckMCIS time.Duration = 180 * time.Second
var durationToCheckMCIS time.Duration = 15 * time.Second

// For this test
var totalTestPeriod = 3 * time.Hour
var deadline time.Time = time.Now().Add(totalTestPeriod)
var timeout time.Duration = totalTestPeriod
var duration time.Duration = 1 * time.Hour
var trialNo int = 0
var pingTrialCount int = 60
var testCase string = "1"
var ruleType string
var cmdType string

func init() {
	fmt.Println("\nStart......... init() of admin-web.go")

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
	// fmt.Printf("exe path: %v\n", exePath)

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
	fmt.Println("")
}

func main() {
	CBLogger.Debug("Start.........")

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

	// ctx, cancel := context.WithDeadline(context.TODO(), deadline)
	// defer cancel()

	timeoutContext, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	// Watch status information for checking latency
	wg.Add(1)
	go watchStatusInformation(gracefulShutdownContext, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	// Watch host network information for checking host network changes
	wg.Add(1)
	go watchHostNetworkInformation(gracefulShutdownContext, &wg)
	// Wait until the goroutine is started
	time.Sleep(200 * time.Millisecond)

	wg.Add(1)
	go func(wg *sync.WaitGroup) {
		defer wg.Done()

		// Block until a signal is triggered
		<-gracefulShutdownContext.Done()

		cancel()

		// Stop this cb-network agent
		fmt.Println("[Stop] Performance evaluation")

		// Wait for a while
		time.Sleep(1 * time.Second)
	}(&wg)

	fmt.Println("\n\n###################################")
	fmt.Println("## Ready to evaluate performance ##")
	fmt.Println("###################################")

	option := "-"

	for option != "q" {
		fmt.Printf("\n%s[Usage] Select a test option: %s\n", string(colorYellow), string(colorReset))
		fmt.Println("    - 1. Interactive test")
		fmt.Println("    - 2. Scheduled test")
		fmt.Println("    - 'q'(Q).  to quit")

		option = readOption()

		fmt.Printf("Option: %v\n", option)
		switch option {
		case "1":
			doInteractiveTest()
			option = "q"

		case "2":
			var twg sync.WaitGroup

			twg.Add(1)
			doScheduledTest(timeoutContext, &twg)
			twg.Wait()

			option = "q"
		default:
			fmt.Printf("\n%sPlease, check option%s\n", string(colorRed), string(colorReset))
		}
	}

	stop()

	wg.Wait()

	CBLogger.Debug("End.........")
}

func watchStatusInformation(ctx context.Context, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")
	defer wg.Done()

	// etcd Section
	// Connect to the etcd cluster
	etcdClient, etcdErr := clientv3.New(clientv3.Config{
		Endpoints:   endpointEtcd,
		DialTimeout: 5 * time.Second,
	})

	if etcdErr != nil {
		CBLogger.Error(etcdErr)
	}

	defer func() {
		errClose := etcdClient.Close()
		if errClose != nil {
			CBLogger.Errorf("Can't close the etcd client (%v)", errClose)
		}
	}()
	CBLogger.Debug("The etcdClient is connected.")

	// Prepare out file
	t := time.Now()
	filename := fmt.Sprintf("output-performance-evaluation-%d%02d%02d%02d%02d%02d.csv", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())

	// Create directory or folder if not exist
	outDirectory := "./result"
	_, err := os.Stat(outDirectory)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(outDirectory, 0600)
		if errDir != nil {
			log.Fatal(err)
		}
	}

	outFilePath := filepath.Join(".", "result", filename)

	file, err := os.Create(outFilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create a csv writer
	wr := csv.NewWriter(bufio.NewWriter(file))

	// Write header
	// {
	// 	"sourceIP": "192.168.0.2",
	// 	"sourceName": "ip-192-168-4-133",
	// 	"destinationIP": "192.168.0.3",
	// 	"destinationName": "ip-192-168-4-136",
	// 	"minimunRTT": 0.003670492,
	// 	"averageRTT": 0.00463211,
	// 	"maximumRTT": 0.00744713,
	// 	"stddevRTT": 0.001027091,
	// 	"packetsReceive": 10,
	// 	"packetLoss": 0,
	// 	"bytesReceived": 240
	// }
	wr.Write([]string{"Trial no.", "Test case", "Rule type", "Command type",
		"Source IP", "Source name", "Destination IP", "Destination name",
		"Minimun RTT (ms)", "Average RTT (ms)", "Maximum RTT (ms)", "Stddev RTT (ms)", "Packets receive", "Packet loss", "Bytes received", "Timestamp"})
	wr.Flush()

	file.Close()

	CBLogger.Info("Watch the result of performance evaluation")
	// Watch "/registry/cloud-adaptive-network/status/information/{cladnet-id}/{host-id}"
	CBLogger.Debugf("Watch \"%v\"", etcdkey.StatusInformation)
	watchChan1 := etcdClient.Watch(ctx, etcdkey.StatusInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan1 {
		for _, event := range watchResponse.Events {
			CBLogger.Tracef("\nevent - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			slicedKeys := strings.Split(string(event.Kv.Key), "/")
			parsedHostID := slicedKeys[len(slicedKeys)-1]
			CBLogger.Tracef("ParsedHostID: %v", parsedHostID)

			status := string(event.Kv.Value)
			CBLogger.Tracef("The status: %v", status)

			var networkStatus model.NetworkStatus

			err := json.Unmarshal(event.Kv.Value, &networkStatus)
			if err != nil {
				CBLogger.Error(err)
			}
			CBLogger.Tracef("%v", networkStatus)

			// Open file
			f, err := os.OpenFile(outFilePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
			if err != nil {
				CBLogger.Error(err)
			}

			// Create a csv writer
			w := csv.NewWriter(bufio.NewWriter(f))

			// Write data
			for _, status := range networkStatus.InterHostNetworkStatus {

				// // Get IP address from string
				// srcIP := net.ParseIP(status.SourceIP)

				// // Convert IP address to value (uint32)
				// srcValue := binary.BigEndian.Uint32(srcIP.To4())
				// CBLogger.Tracef("Source information: %s (%d)", srcIP.String(), srcValue)

				// // Get IP address from string
				// desIP := net.ParseIP(status.DestinationIP)

				// // Convert IP address to value (uint32)
				// desValue := binary.BigEndian.Uint32(desIP.To4())
				// CBLogger.Tracef("Destination information: %s (%d)", desIP.String(), desValue)

				unitCalibration := 1000.0

				minRTT := fmt.Sprintf("%.2f", (status.MininumRTT * unitCalibration))
				avgRTT := fmt.Sprintf("%.2f", (status.AverageRTT * unitCalibration))
				maxRtt := fmt.Sprintf("%.2f", (status.MaximumRTT * unitCalibration))
				stdDevRTT := fmt.Sprintf("%.2f", (status.StdDevRTT * unitCalibration))
				packetReceive := strconv.Itoa(status.PacketsReceive)
				packetLoss := strconv.Itoa(status.PacketsLoss)
				bytesReceived := strconv.Itoa(status.BytesReceived)

				// wr.Write([]string{"Trial no.", "Test case", "Rule type", "Command type",
				// 	"Source IP", "Source name", "Destination IP", "Destination name",
				// 	"Minimun RTT", "Average RTT", "Maximum RTT", "Stddev RTT", "Packets receive", "Packet loss", "Bytes received"})
				now := time.Now().Format("2006-01-02 15:04:05")

				w.Write([]string{strconv.Itoa(trialNo), testCase, ruleType, cmdType,
					status.SourceIP, status.SourceName, status.DestinationIP, status.DestinationName,
					minRTT, avgRTT, maxRtt, stdDevRTT, packetReceive, packetLoss, bytesReceived, now})
				// if srcValue < desValue {
				// 	w.Write([]string{strconv.Itoa(trialNo), testCase, ruleType, cmdType,
				// 		status.SourceIP, status.SourceName, status.DestinationIP, status.DestinationName,
				// 		minRTT, avgRTT, maxRtt, stdDevRTT, packetReceive, packetLoss, bytesReceived, now})
				// }
				// } else {
				// 	w.Write([]string{strconv.Itoa(trialNo), testCase, ruleType, cmdType,
				// 		status.DestinationIP, status.DestinationName, status.SourceIP, status.SourceName,
				// 		minRTT, avgRTT, maxRtt, stdDevRTT, packetReceive, packetLoss, bytesReceived})
				// }
				w.Flush()
			}
			// Close file
			f.Close()
		}
	}
	CBLogger.Debug("End.........")
}

func watchHostNetworkInformation(ctx context.Context, wg *sync.WaitGroup) {
	CBLogger.Debug("Start.........")

	defer wg.Done()

	// etcd Section
	// Connect to the etcd cluster
	etcdClient, etcdErr := clientv3.New(clientv3.Config{
		Endpoints:   endpointEtcd,
		DialTimeout: 5 * time.Second,
	})

	if etcdErr != nil {
		CBLogger.Error(etcdErr)
	}

	defer func() {
		errClose := etcdClient.Close()
		if errClose != nil {
			CBLogger.Errorf("Can't close the etcd client (%v)", errClose)
		}
	}()

	CBLogger.Debug("The etcdClient is connected.")

	// Prepare out file
	t := time.Now()
	filename := fmt.Sprintf("output-network-changes-%d%02d%02d%02d%02d%02d.csv", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())

	// Create directory or folder if not exist
	outDirectory := "./result"
	_, err := os.Stat(outDirectory)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(outDirectory, 0600)
		if errDir != nil {
			log.Fatal(err)
		}
	}

	outFilePath := filepath.Join(".", "result", filename)

	file, err := os.Create(outFilePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	// Create a csv writer
	wr := csv.NewWriter(bufio.NewWriter(file))

	// Write header
	// ns01-yk01perf-gcp-asia-east1-1-c9tmn7bcp5hm4p4muo0g, 34.81.80.251, 34.80.187.51, 192.168.3.131, 192.168.3.131
	wr.Write([]string{"Trial no.", "Hostname", "Previous public IP", "Current public IP", "Previous private IP", "Current private IP", "Timestamp"})
	wr.Flush()

	file.Close()

	CBLogger.Info("Watch the network information of hosts")
	// Watch "/registry/cloud-adaptive-network/host-network-information"
	CBLogger.Debugf("Watch \"%v\"", etcdkey.HostNetworkInformation)

	watchChan2 := etcdClient.Watch(ctx, etcdkey.HostNetworkInformation, clientv3.WithPrefix())
	for watchResponse := range watchChan2 {
		for _, event := range watchResponse.Events {
			switch event.Type {
			case mvccpb.PUT: // The watched value has changed.
				CBLogger.Tracef("\nevent - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)

				// Get the previsou Kv
				key := string(event.Kv.Key)
				CBLogger.Tracef("Key: %v", key)
				resp, respErr := etcdClient.Get(context.TODO(), key, clientv3.WithRev(event.Kv.ModRevision-1))
				if respErr != nil {
					CBLogger.Error(respErr)
				}

				if resp.Count > 0 {

					//Openfile
					f, err := os.OpenFile(outFilePath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
					if err != nil {
						CBLogger.Error(err)
					}

					// Create a csv writer
					w := csv.NewWriter(bufio.NewWriter(f))

					CBLogger.Tracef("current revision(%v), previous revision (%v)", event.Kv.ModRevision, resp.Kvs[0].ModRevision)
					isChanged, hostname, prevHostPublicIP, curHostPublicIP, prevHostIP, curHostIP, err := checkNetworkChanges(event.Kv.Value, resp.Kvs[0].Value)
					if err != nil {
						CBLogger.Error(err)
					}

					if isChanged {
						// Write data
						// wr.Write([]string{"Trial no.", "Hostname", "Previous public IP", "Current public IP", "Previous private IP", "Current private IP"})
						now := time.Now().Format("2006-01-02 15:04:05")
						w.Write([]string{strconv.Itoa(trialNo), hostname, prevHostPublicIP, curHostPublicIP, prevHostIP, curHostIP, now})
						w.Flush()
					}
					// Close file
					f.Close()
				}

			case mvccpb.DELETE: // The watched key has been deleted.
				CBLogger.Tracef("\nevent - %s %q : %q", event.Type, event.Kv.Key, event.Kv.Value)
			default:
				CBLogger.Tracef("\nunknown event (%s), Key(%q), Value(%q)", event.Type, event.Kv.Key, event.Kv.Value)
			}
		}
	}
	CBLogger.Debug("End.........")
}

func checkNetworkChanges(prevBytes, curBytes []byte) (bool, string, string, string, string, string, error) {
	CBLogger.Debug("Start.........")

	var currentHostNetworkInformation model.HostNetworkInformation
	if err := json.Unmarshal(prevBytes, &currentHostNetworkInformation); err != nil {
		CBLogger.Error(err)
		return false, "", "", "", "", "", err
	}
	CBLogger.Tracef("Current host network information: %v", currentHostNetworkInformation)
	curHostName := currentHostNetworkInformation.HostName
	curHostPublicIP := currentHostNetworkInformation.PublicIP

	// Find default host network interface and set IP and IPv4CIDR
	curHostIP, _, err := getDefaultInterfaceInfo(currentHostNetworkInformation.NetworkInterfaces)
	if err != nil {
		CBLogger.Error(err)
		return false, "", "", "", "", "", err
	}

	var previousHostNetworkInformation model.HostNetworkInformation
	if err2 := json.Unmarshal(curBytes, &previousHostNetworkInformation); err2 != nil {
		CBLogger.Error(err2)
		return false, "", "", "", "", "", err2
	}
	CBLogger.Tracef("Previous host network information: %v", previousHostNetworkInformation)
	// prevHostName := previousHostNetworkInformation.HostName
	prevHostPublicIP := previousHostNetworkInformation.PublicIP

	// Find default host network interface and set IP and IPv4CIDR
	prevHostIP, _, err := getDefaultInterfaceInfo(previousHostNetworkInformation.NetworkInterfaces)
	if err != nil {
		CBLogger.Error(err)
		return false, "", "", "", "", "", err
	}

	if prevHostPublicIP != curHostPublicIP || prevHostIP != curHostIP {
		fmt.Printf("\n%sHost network information changed%s\n", string(colorYellow), string(colorReset))
		msg := fmt.Sprintf("%v, %v, %v, %v, %v", curHostName, prevHostPublicIP, curHostPublicIP, prevHostIP, curHostIP)
		CBLogger.Info(msg)
		CBLogger.Debug("End.........")
		return true, curHostName, prevHostPublicIP, curHostPublicIP, prevHostIP, curHostIP, nil
	}

	CBLogger.Debug("End.........")
	return false, "", "", "", "", "", nil
}

func getDefaultInterfaceInfo(networkInterfaces []model.NetworkInterface) (ipAddr string, ipNet string, err error) {
	CBLogger.Debug("Start.........")
	// Find default host network interface and set IP and IPv4CIDR

	for _, networkInterface := range networkInterfaces {
		if networkInterface.Name == "eth0" || networkInterface.Name == "ens4" || networkInterface.Name == "ens5" {
			CBLogger.Debug("End.........")
			return networkInterface.IPv4, networkInterface.IPv4CIDR, nil
		}
	}
	CBLogger.Debug("End.........")
	return "", "", errors.New("could not find default network interface")
}

func doInteractiveTest() {
	CBLogger.Debug("Start.........")
	option := "-"

	for option != "q" {

		printOptions()
		option = readOption()

		start := time.Now()

		handleOption(option)

		elapsed := time.Since(start)
		CBLogger.Debugf("\nElapsed time: %s\nSleep 2 sec ( _ _ )zZ", elapsed)
		time.Sleep(2 * time.Second)
	}
	CBLogger.Debug("End.........")
}

func doScheduledTest(ctx context.Context, wg *sync.WaitGroup) error {
	CBLogger.Debug("Start.........")
	defer wg.Done()

	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	// Do
	CBLogger.Infof("Start at %v", time.Now())

	err := testAndSleep()
	if err != nil {
		CBLogger.Error(err)
		return err
	}

	// While
	for {
		// NOTE - Default Selection
		// The default case in a select is run if no other case is ready.
		// Use a default case to try a send or receive without blocking:

		select {
		case <-ctx.Done():
			CBLogger.Info("The scheduled test is finished.")
			CBLogger.Debug("End.........")
			return nil
		case t := <-ticker.C:
			CBLogger.Infof("Tick at %v", t)

			err := testAndSleep()
			if err != nil {
				CBLogger.Error(err)
				return err
			}
		}
	}
}

func testAndSleep() error {
	trialNo = trialNo + 1

	CBLogger.Infof("(Trial: %d) Wake up and test ", trialNo)
	nextStartTime := time.Now().Add(duration)

	for i := 1; i < 10; i++ {

		start := time.Now()

		handleOption(strconv.Itoa(i))

		elapsed := time.Since(start)
		CBLogger.Debugf("Elapsed time: %s", elapsed)

		if i == 2 || i == 3 || i == 4 || i == 5 || i == 6 {
			CBLogger.Info("Sleep 1 min ( _ _ )zZ to test securely")
			time.Sleep(1 * time.Minute)
		}
	}

	CBLogger.Infof("(Trial: %d) End test and sleep", trialNo)

	CBLogger.Trace(deadline)
	CBLogger.Trace(nextStartTime)

	if deadline.Before(nextStartTime) {
		CBLogger.Info("The scheduled test is finished (by remaining time measurement).")
		CBLogger.Debug("End.........")
		return errors.New("not enough time")
	}

	CBLogger.Infof("The next test will start at %v", nextStartTime)
	return nil
}

func printOptions() {
	fmt.Printf("\n%s[Usage] Select a option: %s\n", string(colorYellow), string(colorReset))
	fmt.Println("    - 1. Check CB-Tumblebug Health")
	fmt.Println("    - 2. Resume MCIS")
	fmt.Println("    - 3. Test Performance (RuleType: basic, Encryption: disabled)")
	fmt.Println("    - 4. Test Performance (RuleType: basic, Encryption: enabled)")
	fmt.Println("    - 5. Test Performance (RuleType: cost-prioritized, Encryption: disabled)")
	fmt.Println("    - 6. Test Performance (RuleType: cost-prioritized, Encryption: enabled)")
	fmt.Println("    - 7. Set RuleType(basic)")
	fmt.Println("    - 8. Suspend MCIS")
	fmt.Println("    - 9. Check MCIS stauts")
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
		testCase = "1"
		ruleType = ruletype.Basic
		cmdType = cmdtype.DisableEncryption
		testPerformance(ruleType, cmdType)

	case "4":
		testCase = "2"
		ruleType = ruletype.Basic
		cmdType = cmdtype.EnableEncryption
		testPerformance(ruleType, cmdType)

	case "5":
		testCase = "3"
		ruleType = ruletype.CostPrioritized
		cmdType = cmdtype.DisableEncryption
		testPerformance(ruleType, cmdType)

	case "6":
		testCase = "4"
		ruleType = ruletype.CostPrioritized
		cmdType = cmdtype.EnableEncryption
		testPerformance(ruleType, cmdType)

	case "7":
		setRuleType(ruletype.Basic)

	case "8":
		suspendMCIS()

	case "9":
		checkStatusOfMCIS()

	case "q":
		fmt.Printf("\n%sSee you soon ^^%s\n", string(colorCyan), string(colorReset))

	default:
		fmt.Printf("\n%sPlease, check option%s\n", string(colorRed), string(colorReset))
	}
}

func checkTumblebugHealth() {
	CBLogger.Debug("Start.........")

	client := resty.New()
	client.SetBasicAuth("default", "default")

	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		Get(fmt.Sprintf("http://%s/tumblebug/health", endpointTB))

	// Output print
	CBLogger.Debugf("\nError: %v", err)
	CBLogger.Debugf("\nTime: %v", resp.Time())
	CBLogger.Tracef("\nBody: %v", resp)

	health := (gjson.Get(resp.String(), "message"))

	CBLogger.Infof("%v", health)

	CBLogger.Debug("End.........")
}

func resumeMCIS() {
	CBLogger.Debug("Start.........")

	status := checkStatusOfMCIS()

	isRunning := strings.Contains(status, "Running")

	re := regexp.MustCompile("[0-9]+")
	nums := re.FindAllString(status, -1)

	// if it's not running status
	if !(isRunning && nums[0] == nums[1] && nums[1] == nums[2]) {
		client := resty.New()
		client.SetBasicAuth("default", "default")

		CBLogger.Infof("Resume an MCIS (%v) in a ns (%v)", mcisID, nsID)
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
		CBLogger.Debugf("\nError: %v", err)
		CBLogger.Debugf("\nTime: %v", resp.Time())
		CBLogger.Tracef("\nBody: %v", resp)

		// Check if all VMs run
		CBLogger.Infof("Check MCIS status for %v (interval: %v)", timeoutToCheckMCIS, durationToCheckMCIS)
		ctx, cancel := context.WithTimeout(context.TODO(), timeoutToCheckMCIS)
		defer cancel()

		err = checkRunning(ctx)
		if err != nil {
			CBLogger.Error(err)
			CBLogger.Info("Wait for an additional 30 seconds")
			time.Sleep(30 * time.Second)
		}
	}

	CBLogger.Debug("End.........")
}

func checkRunning(ctx context.Context) error {
	CBLogger.Debug("Start.........")

	status := checkStatusOfMCIS()

	isRunning := strings.Contains(status, "Running")

	re := regexp.MustCompile("[0-9]+")
	nums := re.FindAllString(status, -1)

	if isRunning && nums[0] == nums[1] && nums[1] == nums[2] {
		CBLogger.Debug("End.........")
		return nil
	}

	ticker := time.NewTicker(durationToCheckMCIS)

	for {
		select {
		case <-ctx.Done():
			CBLogger.Debug("End.........")
			return errors.New("timeout")
		case <-ticker.C:
			status := checkStatusOfMCIS()

			isRunning := strings.Contains(status, "Running")

			re := regexp.MustCompile("[0-9]+")
			nums := re.FindAllString(status, -1)

			if isRunning && nums[0] == nums[1] && nums[1] == nums[2] {
				CBLogger.Debug("End.........")
				return nil
			}
		}
	}
}

func checkStatusOfMCIS() string {
	CBLogger.Debug("Start.........")

	client := resty.New()
	client.SetBasicAuth("default", "default")

	CBLogger.Info("Check MCIS status")
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
		Get(fmt.Sprintf("http://%s/tumblebug/ns/{nsId}/mcis/{mcisId}", endpointTB))

	// Output print
	CBLogger.Debugf("\nError: %v", err)
	CBLogger.Debugf("\nTime: %v", resp.Time())
	// CBLogger.Tracef("\nBody: %v", resp)

	mcisStatus := gjson.Get(resp.String(), "status.status")
	CBLogger.Infof("\n ==> MCIS status: %#v", mcisStatus.String())

	CBLogger.Debug("End.........")
	return mcisStatus.String()
}

func suspendMCIS() {
	CBLogger.Debug("Start.........")

	client := resty.New()
	client.SetBasicAuth("default", "default")

	CBLogger.Infof("Suspend an MCIS (%v) in a ns (%v)", mcisID, nsID)
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
	CBLogger.Debugf("\nError: %v", err)
	CBLogger.Debugf("\nTime: %v", resp.Time())
	CBLogger.Tracef("\nBody: %v", resp)

	// Check if all VMs are suspended
	CBLogger.Infof("Check MCIS status for %v (interval: %v)", timeoutToCheckMCIS, durationToCheckMCIS)
	ctx, cancel := context.WithTimeout(context.TODO(), timeoutToCheckMCIS)
	defer cancel()

	err = checkSuspended(ctx)
	if err != nil {
		CBLogger.Error(err)
		CBLogger.Info("Wait for an additional 30 seconds")
		time.Sleep(30 * time.Second)
	}

	CBLogger.Debug("End.........")
}

func checkSuspended(ctx context.Context) error {
	CBLogger.Debug("Start.........")

	status := checkStatusOfMCIS()

	isRunning := strings.Contains(status, "Suspended")

	re := regexp.MustCompile("[0-9]+")
	nums := re.FindAllString(status, -1)

	if isRunning && nums[0] == nums[1] && nums[1] == nums[2] {
		CBLogger.Debug("End.........")
		return nil
	}

	ticker := time.NewTicker(durationToCheckMCIS)

	for {
		select {
		case <-ctx.Done():
			CBLogger.Debug("End.........")
			return errors.New("timeout")
		case <-ticker.C:
			status := checkStatusOfMCIS()

			isRunning := strings.Contains(status, "Suspended")

			re := regexp.MustCompile("[0-9]+")
			nums := re.FindAllString(status, -1)

			if isRunning && nums[0] == nums[1] && nums[1] == nums[2] {
				CBLogger.Debug("End.........")
				return nil
			}
		}
	}
	// fmt.Println("\n\n##### End ---------- checkRunning()")
	// return errors.New("unknown")
}

func testPerformance(ruleType string, encryptionCommand string) {
	CBLogger.Debug("Start.........")

	//// Initialize cb-network service
	// Connect to the gRPC server
	// Register CloudAdaptiveNetwork handler to gwmux
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	grpcConn, err := grpc.Dial(endpointNetworkService, options...)
	if err != nil {
		CBLogger.Errorf("Cannot connect: %v", err)
		// return model.CLADNetSpecification{}, err
	}
	defer grpcConn.Close()

	// Create stubs of cb-network service
	cladnetClient := pb.NewCloudAdaptiveNetworkServiceClient(grpcConn)
	systemClient := pb.NewSystemManagementServiceClient(grpcConn)

	if ruleType == ruletype.CostPrioritized {
		CBLogger.Info("Inject cloud information (automatically set rule type (cost-prioritized)")
		client := resty.New()
		client.SetBasicAuth("default", "default")

		placeHolder := `{"etcdEndpoints": %s, "serviceEndpoint": "%s"}`

		endpointEtcdJSON, errMashal := json.Marshal(endpointEtcd)
		if errMashal != nil {
			CBLogger.Error(errMashal)
		}
		body := fmt.Sprintf(placeHolder, endpointEtcdJSON, endpointNetworkService)

		resp, errResp := client.R().
			SetHeader("Content-Type", "application/json").
			SetHeader("Accept", "application/json").
			SetPathParams(map[string]string{
				"nsId":   nsID,
				"mcisId": mcisID,
			}).
			SetBody(body).
			Put(fmt.Sprintf("http://%s/tumblebug/ns/{nsId}/network/mcis/{mcisId}", endpointTB))

		// Output print
		CBLogger.Debugf("\nError: %v", errResp)
		CBLogger.Debugf("Time: %v", resp.Time())
		CBLogger.Tracef("Body: %v", resp)
	}

	//// Set end-to-end encryption
	CBLogger.Infof("Set end-to-end encryption (%v)", encryptionCommand)
	controlRequest := &pb.ControlRequest{
		CladnetId:   mcisID,
		CommandType: pb.CommandType(pb.CommandType_value[encryptionCommand]),
	}

	controlRes, err := systemClient.ControlCloudAdaptiveNetwork(context.TODO(), controlRequest)
	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Control response: %v", controlRes)

	// Get the specification of a Cloud Adaptive Network
	peerRequest := &pb.PeerRequest{
		CladnetId: mcisID,
		HostId:    "",
	}

	peerListResp, err := cladnetClient.GetPeerList(context.TODO(), peerRequest)
	if err != nil {
		CBLogger.Error(err)
	}

	numOfPeers := len(peerListResp.Peers)

	sleepTime := time.Duration(3*numOfPeers) * time.Second

	CBLogger.Infof("Wait %v ( _ _ )zZ to setup securely", sleepTime)
	time.Sleep(sleepTime)

	// Request test
	tempSpec, err := json.Marshal(model.TestSpecification{
		CladnetID:  mcisID,
		TrialCount: pingTrialCount,
	})
	CBLogger.Tracef("Test specification: %v", tempSpec)

	if err != nil {
		CBLogger.Error(err)
	}
	testSpec := string(tempSpec)

	testRequest := &pb.TestRequest{
		CladnetId: mcisID,
		TestType:  pb.TestType_CONNECTIVITY,
		TestSpec:  testSpec,
	}

	CBLogger.Info("Request performance evaluation")
	testRes, err := systemClient.TestCloudAdaptiveNetwork(context.TODO(), testRequest)
	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Test response: %v", testRes)

	sleepTime2 := time.Duration(pingTrialCount+5) * time.Second
	CBLogger.Infof("Wait %v to test securely", sleepTime2)
	time.Sleep(sleepTime2)

	CBLogger.Debug("End.........")
}

func setRuleType(ruleType string) {
	CBLogger.Debug("Start.........")

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
		CBLogger.Errorf("could not connect: %v", err)
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
		CBLogger.Error(err)
	}

	// Assign rule
	cladnetSpec.RuleType = ruleType

	// Update the specification
	CBLogger.Info("Set rule type")
	updatedCladnetSpec, err := cladnetClient.UpdateCLADNet(ctx, cladnetSpec)
	if err != nil {
		CBLogger.Error(err)
	}
	CBLogger.Tracef("Update cladnet spec: %v", updatedCladnetSpec)

	CBLogger.Debug("End.........")
}
