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
func ProcessDirectory(dir *Directory) {
	results := make(chan string) // Channel to collect results

	routes := map[string][]string{
		"R1": {"KL01", "KL02", "KL03"},
		"R2": {"KL01", "KL02", "KL04"},
		"R3": {"KL01", "KL05"},
	}
	processToFetch := GetDistinct(routes)

	defer close(results)

	// Launch a goroutine for each contact
	for _, contact := range dir.Contacts {
		go func(contact Contact) {
			address := contact.Address
			prList := FindCommon(processToFetch, contact.Offerings)

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
			response := Ping(client, prList)
			results <- fmt.Sprintf("Response from %s: %v", contact.Name, response)
		}(contact)
	}

	// Collect and log results
	for range dir.Contacts {
		log.Println(<-results)
	}
}

func FindCommon(list1, list2 []string) []string {
	var common []string
	elementMap := make(map[string]bool)

	for _, item := range list2 {
		elementMap[item] = true
	}

	for _, item := range list1 {
		if elementMap[item] {
			common = append(common, item)
		}
	}
	return common
}

func GetDistinct(routes map[string][]string) []string {
	uniqueKL := make(map[string]bool)

	// Iterate through the routes and collect unique KL values
	for _, steps := range routes {
		for _, kl := range steps {
			uniqueKL[kl] = true
		}
	}

	// Convert map keys to a slice (if needed)
	var klList []string
	for kl := range uniqueKL {
		klList = append(klList, kl)
	}
	return klList

}

func Ping(c pb.SubmissionServiceClient, tasks []string) *pb.ProcessResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	now := time.Now()
	submittedAt := timestamppb.New(now)
	process := &pb.Process{
		SubmittedAt:  submittedAt,
		Requirements: tasks,
	}
	fmt.Println(process)

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
