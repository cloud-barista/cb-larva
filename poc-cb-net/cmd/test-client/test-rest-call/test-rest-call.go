package main

import (
	"encoding/json"
	"fmt"

	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	nethelper "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/network-helper"
	"github.com/go-resty/resty/v2"
)

func main() {
	// var placeHolder = `{"ipNetworks": %s }`
	var ipNetworks = []string{
		"192.168.0.0/24",
		"192.168.0.0/25",
		"192.168.0.0/26",
		"192.168.0.0/27",
		"172.16.0.0/16",
		"172.16.0.0/16",
		"10.0.0.0/16",
		"10.0.0.0/10",
	}

	ipNets := &pb.IPNetworks{IpNetworks: ipNetworks}

	ipNetworksJson, _ := json.Marshal(ipNets)
	fmt.Println(string(ipNetworksJson))
	// Set request body
	// body := fmt.Sprintf(placeHolder, ipNetworksJson)
	// fmt.Printf("body: %#v\n", body)

	client := resty.New()
	// client.SetBasicAuth("default", "default")
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetBody(ipNetworksJson).
		Post(fmt.Sprintf("%s/v1/cladnet/available-ipv4-address-spaces", "http://localhost:8053"))
	// Output print
	fmt.Printf("\nError: %v\n", err)
	fmt.Printf("Time: %v\n", resp.Time())
	fmt.Printf("Body: %v\n", resp)

	var availableIPv4PrivateAddressSpaces nethelper.AvailableIPv4PrivateAddressSpaces

	json.Unmarshal(resp.Body(), &availableIPv4PrivateAddressSpaces)

	fmt.Printf("%#v\n", availableIPv4PrivateAddressSpaces)
}
