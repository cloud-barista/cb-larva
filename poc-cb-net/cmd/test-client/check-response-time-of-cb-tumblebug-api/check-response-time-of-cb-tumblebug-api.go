package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

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

func selectOptions(isOn bool) string {
	line := ""
	if isOn {
		for {
			fmt.Printf("\n\n%s[Usage] Select an option below%s\n", string(colorYellow), string(colorReset))
			fmt.Println("    - Option '1' to check response time when requesting remote command to an MCIS")
			fmt.Println("    - Option '2' to check response time when requesting remote command to each VM in an MCIS")
			fmt.Println("    - Option 'q' to quit")
			fmt.Print(">> ")

			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				line = scanner.Text()
				if line == "1" || line == "2" || line == "q" {
					return line
				}
			}
		}
	}
	return line
}

func main() {

	// Wait for multiple goroutines to complete
	var wg sync.WaitGroup

	start := time.Now()

	nsID := "ns01"
	mcisID := "mcis01"

	client := resty.New()
	client.SetBasicAuth("default", "default")

	// Step 1: Health-check CB-Tumblebug
	fmt.Println("\n\n##### Start ---------- Step 1: Health-check CB-Tumblebug")

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

	// Step 3: Retrieve the lead/master VM ID and all VM IDs
	// curl -X GET "http://localhost:1323/tumblebug/ns/ns01/mcis/mcis01?option=status" -H "accept: application/json"
	fmt.Println("\n\n##### Start ---------- Step 3: Retrieve the lead/master VM ID and all VM IDs")

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

	retMasterVMID := gjson.Get(resp.String(), "status.masterVmId")
	retVMIDs := gjson.Get(resp.String(), "status.vm.#.id")
	fmt.Printf("retMasterVMID: %#v\n", retMasterVMID.String())
	fmt.Printf("retVMIDs: %#v\n", retVMIDs.String())

	fmt.Println("##### End ---------- Step 3: Retrieve the lead/master VM ID and all VM IDs")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Step 4: Check response time
	fmt.Println("\n\n##### Start ---------- Step 4: Check response time")

	// Set command
	commandToTest := `sleep 10`

	// Set request body
	body := fmt.Sprintf(placeHolderBody, commandToTest)

CheckResponseTime:
	for {
		option := selectOptions(true)
		testStart := time.Now()

		fmt.Printf("body: %#v\n", body)

		switch option {
		case "1":
			respEach, errEach := client.R().
				SetHeader("Content-Type", "application/json").
				SetHeader("Accept", "application/json").
				SetPathParams(map[string]string{
					"nsId":   nsID,
					"mcisId": mcisID,
				}).
				SetBody(body).
				Post(fmt.Sprintf("%s/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}", endpointTB))

			// Output print
			fmt.Printf("\nError: %v\n", errEach)
			fmt.Printf("Time: %v\n", respEach.Time())
			// fmt.Printf("Body: %v\n", respEach)
			ret := gjson.Get(respEach.String(), "result")
			fmt.Println("[Result]")
			fmt.Println(ret)

		case "2":

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
				time.Sleep(1 * time.Second)
			}
			wg.Wait()
		case "q":
			break CheckResponseTime
		default:
			fmt.Printf("Unknown option: %s\n", option)
		}

		testElapsed := time.Since(testStart)
		fmt.Printf("Response time: %s\n", testElapsed)
	}

	fmt.Println("##### End ---------- Step 4: Check response time")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Step 5: Delete MCIS
	// curl -X DELETE "http://localhost:1323/tumblebug/ns/ns01/mcis/mcis01?option=terminate" -H "accept: application/json"
	fmt.Println("\n\n##### Start ---------- // Step 5: Delete MCIS")
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
		Delete("http://localhost:1323/tumblebug/ns/{nsId}/mcis/{mcisId}")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- // Step 5: Delete MCIS")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Step 6: Delete defaultResources
	// curl -X DELETE "http://localhost:1323/tumblebug/ns/ns01/defaultResources" -H "accept: application/json"
	fmt.Println("\n\n##### Start ---------- Step 6: Delete defaultResources")
	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId": nsID,
		}).
		Delete("http://localhost:1323/tumblebug/ns/{nsId}/defaultResources")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- Step 6: Delete defaultResources")

	elapsed := time.Since(start)
	fmt.Printf("Elapsed time: %s\n", elapsed)
}
