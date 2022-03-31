package main

import (
	"encoding/json"
	"fmt"
	"log"

	model "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/cb-network/model"
	nethelper "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/network-helper"
	"github.com/go-resty/resty/v2"
)

func main() {
	// var placeHolder = `{"ipNetworks": %s }`
	gRPCServiceEndpoint := "localhost:8053"
	cladnetName := "cbnet01"
	cladnetDescription := "It's a recommended cladnet"

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

	var spec model.CLADNetSpecification

	ipNetworksHolder := `{"ipNetworks": %s}`
	tempJSON, _ := json.Marshal(ipNetworks)
	ipNetworksString := fmt.Sprintf(ipNetworksHolder, string(tempJSON))
	fmt.Printf("%#v\n", ipNetworksString)

	client := resty.New()
	// client.SetBasicAuth("default", "default")

	// Request a recommendation of available IPv4 private address spaces.
	resp, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetBody(ipNetworksString).
		Post(fmt.Sprintf("http://%s/v1/cladnet/available-ipv4-address-spaces", gRPCServiceEndpoint))
	// Output print
	log.Printf("\nError: %v\n", err)
	log.Printf("Time: %v\n", resp.Time())
	log.Printf("Body: %v\n", resp)

	if err != nil {
		log.Printf("Could not request: %v\n", err)
		// return model.CLADNetSpecification{}, err
	}

	var availableIPv4PrivateAddressSpaces nethelper.AvailableIPv4PrivateAddressSpaces

	json.Unmarshal(resp.Body(), &availableIPv4PrivateAddressSpaces)
	log.Printf("%+v\n", availableIPv4PrivateAddressSpaces)
	log.Printf("RecommendedIpv4PrivateAddressSpace: %#v", availableIPv4PrivateAddressSpaces.RecommendedIPv4PrivateAddressSpace)

	cladnetSpecHolder := `{"id": "", "name": "%s", "ipv4AddressSpace": "%s", "description": "%s"}`
	cladnetSpecString := fmt.Sprintf(cladnetSpecHolder,
		cladnetName, availableIPv4PrivateAddressSpaces.RecommendedIPv4PrivateAddressSpace, cladnetDescription)
	fmt.Printf("%#v\n", cladnetSpecString)

	// Request to create a Cloud Adaptive Network
	resp, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetBody(cladnetSpecString).
		Post(fmt.Sprintf("http://%s/v1/cladnet", gRPCServiceEndpoint))
	// Output print
	log.Printf("\nError: %v\n", err)
	log.Printf("Time: %v\n", resp.Time())
	log.Printf("Body: %v\n", resp)

	if err != nil {
		log.Printf("Could not request: %v\n", err)
		// return model.CLADNetSpecification{}, err
	}

	json.Unmarshal(resp.Body(), &spec)
	log.Printf("%#v\n", spec)
}
