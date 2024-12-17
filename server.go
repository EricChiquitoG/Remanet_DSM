package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
	"github.com/EricChiquitoG/Remanet_DSM/src"
	"google.golang.org/grpc"
)

// Create a struct and embed our UnimplementCofeeShopServer
// We provide a full implementation to the methods that this embedded struct specifies down below
type server struct {
	pb.UnimplementedSubmissionServiceServer
}

func (s *server) CheckAvailabilty(context context.Context, pr *pb.Process) (*pb.ProcessResponse, error) {
	company := os.Getenv("Company")
	compData, err := src.InitializeData((company))
	if err != nil {
		return &pb.ProcessResponse{
			Status:  "Unable to set up data",
			Message: "Unable to set up data in server",
		}, nil
	}
	for _, process := range src.AvailabilityExample.P {
		// Check if the process exists in exampleData
		fmt.Println("Is this working?")
		foundProcess := false

		if process.Process == compData.Process {
			// Process matched, now check for productType
			fmt.Println(compData)
			for _, product := range compData.ProductTypes {
				fmt.Println(process.ProductType, product)

				if process.ProductType == product {
					fmt.Printf("Process '%s' and ProductType '%s' exists in AvailableProcesses Activity '%s'.\n",
						process.Process, process.ProductType, compData.Process)
					foundProcess = true
					break

				}

			}
		}
		if foundProcess {
			return &pb.ProcessResponse{
				Status:  "process found",
				Message: "Process Found",
			}, nil
		}

	}
	return &pb.ProcessResponse{
		Status:  "Process not found",
		Message: "Process not Found",
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
