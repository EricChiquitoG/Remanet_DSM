package src

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProcessDirectory using goroutines
func ProcessDirectory(dir Directory) {
	results := make(chan string) // Channel to collect results
	defer close(results)

	// Launch a goroutine for each contact
	for _, contact := range dir.Contacts {
		go func(contact Contact) {
			address := contact.Address

			// Establish a gRPC connection to the contact's server
			conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				results <- fmt.Sprintf("Failed to connect to %s at %s: %v", contact.Name, address, err)
				return
			}
			defer conn.Close()

			// Create a client
			client := pb.NewSubmissionServiceClient(conn)

			// Ping the server, need to change later when we add different messages
			response := Ping(client)
			results <- fmt.Sprintf("Response from %s: %v", contact.Name, response)
		}(contact)
	}

	// Collect and log results
	for range dir.Contacts {
		log.Println(<-results)
	}
}

func Ping(c pb.SubmissionServiceClient) *pb.ProcessResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	now := time.Now()
	submittedAt := timestamppb.New(now)
	process := &pb.Process{
		StepName:         "Quality Inspection",
		ProductType:      "Steel Blades",
		EconomicOperator: "ABC Corp",
		SubmittedAt:      submittedAt,
		Requirements: []string{
			"ISO certification",
			"Safety compliance check",
			"Material origin verification",
		},
	}

	// Make a simple call to order all the items on the menu
	serverResponse, err := c.CheckAvailabilty(ctx, process)
	if err != nil {
		// Handle the error case, return a default response or log the error
		return &pb.ProcessResponse{
			Status:  "Error",
			Message: fmt.Sprintf("Failed to check availability: %v", err),
		}
	}

	// Return the successful server response
	return serverResponse
}
