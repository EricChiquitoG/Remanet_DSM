package src

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sort"
	"time"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ResultCollection struct {
	Results []Result
}

type Result struct {
	ContactName string
	Matches     []string
	Response    string
}

type RouteRequest struct {
	ProductMatch string              `json:"product_match"`
	Routes       map[string][]string `json:"routes"`
}

// ProcessDirectory using goroutines
func ProcessDirectory(c *gin.Context) {
	var request RouteRequest

	// Bind JSON to the struct
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	product_match := request.ProductMatch
	routes := request.Routes

	fmt.Println(routes)

	dir, err := MyDir("./data/directory.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	costs, err := Costs("./data/cost.json")
	if err != nil {
		log.Fatalf("Error: %v", err)

	}
	results := make(chan Result)                  // Channel to collect results
	var PossibleRoutesI [][]string                //Individual routes
	PossibleRoutes := make(map[string][][]string) //All routes
	var taskN = 0

	processToFetch := GetDistinct(routes)

	resultCollection := ResultCollection{}
	allCost := AllCost{}

	defer close(results)

	// Launch a goroutine for each contact
	for _, contact := range dir.Contacts {
		go func(contact Contact) {
			address := contact.Address
			prList := FindCommon(processToFetch, contact.Offerings)

			// Establish a gRPC connection to the contact's server
			conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				results <- Result{
					ContactName: contact.Name,
					Response:    fmt.Sprintf("Failed to connect: %v", err),
				}
				return
			}
			defer conn.Close()

			// Create a client
			client := pb.NewSubmissionServiceClient(conn)

			// Ping the server, need to change later when we add different messages
			response := Ping(client, prList, product_match)
			results <- Result{
				ContactName: contact.Name,
				Matches:     response.Capability,
			}
		}(contact)
	}

	// Collect and log results
	for range dir.Contacts {
		resultCollection.Results = append(resultCollection.Results, <-results)

	}
	fmt.Println(resultCollection)
	resultMap := CreateMap(processToFetch, resultCollection)
	routeIndex := indexBuilder(resultMap)
	for index, route := range routes {

		route_c := []string{}
		taskN = 0
		pathList := pathMaker(route_c, taskN, resultMap, routeIndex, PossibleRoutesI, route)
		PossibleRoutes[index] = pathList
	}
	distances := costCalculator(dir, PossibleRoutes, routes, costs, &allCost)
	fmt.Println(distances)
	c.JSON(http.StatusOK, gin.H{
		"message": "Data received successfully",
		"Options": distances,
	})

}

func indexBuilder(taskMap map[string][]string) []string {
	keys := make([]string, 0, len(taskMap))
	for key := range taskMap {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

func pathMaker(path_c []string, taskN int, taskMap map[string][]string, index []string, routes [][]string, route []string) [][]string {
	if taskN == len(route) {
		routes = append(routes, append([]string{}, path_c...))

	} else {

		for _, key := range index {
			if key == route[taskN] { // Match found
				for _, com := range taskMap[key] {
					newPath := append([]string{}, path_c...)
					newPath = append(newPath, com)
					routes = pathMaker(newPath, taskN+1, taskMap, index, routes, route) // Move to the next in `route`
				}
				break // Stop searching once we find the correct key
			}
		}

	}
	return routes

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

// Creates the mao of the actors that can provide certain activities
func CreateMap(Activities []string, resultCollection ResultCollection) map[string][]string {
	activityMap := make(map[string][]string)

	for _, task := range Activities {
		for _, result := range resultCollection.Results {
			if slices.Contains(result.Matches, task) {
				activityMap[task] = append(activityMap[task], result.ContactName)
			}
		}
	}
	return activityMap
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

func Ping(c pb.SubmissionServiceClient, tasks []string, product string) *pb.ProcessResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	now := time.Now()
	submittedAt := timestamppb.New(now)
	process := &pb.Process{
		ProductType:  product,
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
