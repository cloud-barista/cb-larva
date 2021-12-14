package main

import (
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/tidwall/gjson"
)

func main() {

	start := time.Now()

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

	tbMCISInfo := resp

	fmt.Println("##### End ---------- Step 2: Create MCIS dynamically")
	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

	// Step 3: (Test) Send a command to specified MCIS
	fmt.Println("\n\n##### Start ---------- Step 3: (Test) Send a command to specified MCIS")
	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetPathParams(map[string]string{
			"nsId":   nsID,
			"mcisId": mcisID,
		}).
		SetBody(`{"command": "hostname", "userName": "cb-user"}`).
		Post("http://localhost:1323/tumblebug/ns/{nsId}/cmd/mcis/{mcisId}")

	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	fmt.Println("##### End ---------- Step 3: (Test) Send a command to specified MCIS")
	fmt.Println("Sleep 5 sec ( _ _ )zZ")
	time.Sleep(5 * time.Second)

	// Step 4: Get VM address spaces
	fmt.Println("\n\n##### Start ---------- Step 4: Get VM address spaces")

	vNetIDs := []string{}

	retVNetIDs := gjson.Get(tbMCISInfo.String(), "vm.#.vNetId")
	fmt.Printf("retVNetIDs: %#v\n", retVNetIDs)

	for _, vNetID := range retVNetIDs.Array() {
		vNetIDs = append(vNetIDs, vNetID.String())
	}
	fmt.Printf("vNetIds: %#v\n", vNetIDs)

	ipNets := []string{}

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
		ipNets = append(ipNets, retIPv4CIDR.String())
	}

	fmt.Printf("IPNets: %#v\n", ipNets)

	fmt.Println("##### End ---------- Step 4: Get VM address spaces")
	fmt.Println("Sleep 10 sec ( _ _ )zZ")
	time.Sleep(10 * time.Second)

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
