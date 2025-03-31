package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
	"github.com/EricChiquitoG/Remanet_DSM/server/src"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Create a struct and embed our UnimplementCofeeShopServer
// We provide a full implementation to the methods that this embedded struct specifies down below
type server struct {
	pb.UnimplementedSubmissionServiceServer
}

func (s *server) CheckAvailabilty(context context.Context, pr *pb.Process) (*pb.ProcessResponse, error) {
	company := os.Getenv("Company")
	CData, err := src.InitializeData((company))
	if err != nil {
		return &pb.ProcessResponse{
			Status:  "Unable to set up data",
			Message: "Unable to set up data in server",
		}, nil
	}
	MatchList := src.Match(pr, CData)

	if len(MatchList) != 0 {
		return &pb.ProcessResponse{
			Capability: MatchList,
			Message:    "Matches found",
		}, nil
	}
	return &pb.ProcessResponse{
		Status:  "Process not found",
		Message: "Process not Found",
	}, nil

}

func main() {

	port := os.Getenv("GRPC_PORT")
	name := os.Getenv("Company")
	servername := os.Getenv("ServiceName")
	Offerings := []string{"KL01", "KL02", "KL03"}
	Location := []float64{65.584816, 22.156704}
	CostH := 101.22

	conn, err := grpc.NewClient("client:50050", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return
	}
	defer conn.Close()

	// Create a client
	client := pb.NewSubmissionServiceClient(conn)

	response := EnrollService(client, name, servername, Location, CostH, Offerings)

	fmt.Println(response)
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

func EnrollService(c pb.SubmissionServiceClient, name string, address string, location []float64, cost float64, offerings []string) *pb.EnrollResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	defer cancel()
	process := &pb.Enroll{
		Name:      name,
		Address:   address,
		Location:  location,
		CostH:     cost,
		Offerings: offerings,
	}
	// Make a simple call to order all the items on the menu
	serverResponse, err := c.EnrollServer(ctx, process)
	if err != nil {
		// Handle the error case, return a default response or log the error
		return &pb.EnrollResponse{
			Status:  "Error",
			Message: fmt.Sprintf("Failed to check availability: %v", err),
		}
	}

	// Return the successful server response
	return serverResponse
}
