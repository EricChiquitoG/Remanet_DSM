package main

import (
	"context"
	"log"
	"net"
	"os"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
	"github.com/EricChiquitoG/Remanet_DSM/server_customer/src"
	"google.golang.org/grpc"
)

// Create a struct and embed our UnimplementCofeeShopServer
// We provide a full implementation to the methods that this embedded struct specifies down below
type server struct {
	pb.UnimplementedSubmissionServiceServer
}

func (s *server) CheckInterest(context context.Context, pr *pb.Purchase) (*pb.PurchaseResponse, error) {
	company := os.Getenv("Company")
	CData, err := src.InitializeData((company))
	if err != nil {
		return &pb.PurchaseResponse{
			Status:  "Unable to set up data",
			Message: "Unable to set up data in server",
		}, nil
	}
	Matched := src.Match(pr, CData)

	if Matched {
		return &pb.PurchaseResponse{
			Capability: true,
			Message:    "PRODUCT MATCHED SUCCESFULLY",
		}, nil
	}

	return &pb.PurchaseResponse{
		Capability: false,
		Message:    "PRODUCT NOT PURCHASEABLE",
	}, nil

}

func main() {

	port := os.Getenv("GRPC_PORT")
	// setup a listener on port 9001
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// create a new grpc server
	grpcServer := grpc.NewServer()

	// register our server struct as a handle for the CoffeeShopService rpc calls that come in through grpcServer
	pb.RegisterSubmissionServiceServer(grpcServer, &server{})

	// Serve traffic
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}
