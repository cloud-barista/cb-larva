package main

import (
	"context"
	"log"
	"time"

	pb "github.com/cloud-barista/cb-larva/poc-cb-net/pkg/api/gen/go/cbnetwork"
	"google.golang.org/grpc"
)

func main() {

	conn, err := grpc.Dial("localhost:8088", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	c := pb.NewCloudAdaptiveNetworkClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.CreateCLADNet(ctx, &pb.CreateCLADNetRequest{
		CladnetSpecification: &pb.CLADNetSpecification{
			Id:               "",
			Name:             "CLADNet01",
			Ipv4AddressSpace: "192.168.77.0/26",
			Description:      "Alvin's CLADNet01"}})

	if err != nil {
		log.Fatalf("could not request: %v", err)
	}

	log.Printf("Config: %v", r)
}
