package src

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"sync"
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
	ContactName    string
	Matches        []string
	Response       string
	Capability     bool
	TotalLogistics float64
}

type RouteRequest struct {
	ProductMatch  string              `json:"product_match"`
	Routes        map[string][]string `json:"routes"`
	StartingPoint []float64           `json:"starting_point"`
}

type ProcessedSolution struct {
	SolNumber         int
	UserIDs           []string // The sequence of assigned actor IDs
	TransportCost     float64  // The transport cost for this solution
	ManufacturingCost float64  // The manufacturing cost for this solution
}

// OptimizationResults holds the processed results from the gRPC response.
// This struct is returned by the processOptimizationResponse function.
type OptimizationResults struct {
	Solutions []ProcessedSolution // A list of the processed solutions found
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

	dir, err := MyDir("./data/directory.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	addLoc, err := MyLocs("./data/locationAdd.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	PClasses, err := getInterests("./data/interests.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	costs, err := Costs("./data/cost.json")
	if err != nil {
		log.Fatalf("Error: %v", err)

	}
	results := make(chan Result) // Channel to collect results

	processToFetch := GetDistinct(request.Routes)

	resultCollection := ResultCollection{}

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
					//Response:    fmt.Sprintf("Failed to connect: %v", err),
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
		resultCollection.Results = append(resultCollection.Results, <-results)
	}
	/* 	originNode := Node{
	   		ID:      0,
	   		Company: "origin",
	   		Step:    0,
	   		Process: "Alpha",
	   	}
	   	endNode := Node{
	   		ID:      100,
	   		Company: "end",
	   		Step:    100,
	   		Process: "Omega",
	   	} */
	resultMap := CreateMap(processToFetch, resultCollection)
	resCollectionCustomer, custResponses := customerSearch(product_match, PClasses, dir)
	// Append elements from both customer and TechProvider responses
	for key, val := range custResponses {
		resultMap[key] = val
	}
	resultCollection.Results = append(resultCollection.Results, resCollectionCustomer.Results...)

	CostMatrix, err := CreateCostMatrixFromResults(resultCollection, costs)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	distMatrix := DistanceMatrixConstructor(addLoc)
	processToFetch = append(processToFetch, "end")
	jsonData := exportToJson(processToFetch, CostMatrix, distMatrix)
	conn, err := connectToServer("pymoo-runner:50060")
	if err != nil {
		log.Fatalf("failed to connect to server: %v", err)
	}
	defer conn.Close()
	client := pb.NewSubmissionServiceClient(conn)
	fmt.Println("sending", jsonData)
	res, err := sendOptimize(jsonData, client)
	if err != nil {
		log.Fatalf("failed to send optimization request: %v", err)
	}
	// Process the response
	optResults := processOptimizationResponse(res)
	c.JSON(http.StatusOK, gin.H{
		"message": "Data received successfully",
		"Options": optResults,
	})

}

func exportToJson(processes []string, CostMatrix map[string]map[string]float64, distM map[string]map[string]float64) map[string]interface{} {
	data := map[string]interface{}{
		"processes":  processes,
		"CostMatrix": CostMatrix,
		"matrix":     distM,
	}

	file, err := os.Create("output.json")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return nil
	}
	defer file.Close()

	// Encode with indentation for readability
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		fmt.Println("Error writing JSON:", err)
	} else {
		fmt.Println("Data written to output.json")
	}
	return data
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

func CheckCustomer(c pb.SubmissionServiceClient, product string) *pb.PurchaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	defer cancel()
	process := &pb.Purchase{
		ProductType: product,
		Amount:      "1",
	}
	fmt.Println(process)

	// Make a simple call to order all the items on the menu
	serverResponse, err := c.CheckInterest(ctx, process)
	if err != nil {
		// Handle the error case, return a default response or log the error
		return &pb.PurchaseResponse{
			Status:  "Error",
			Message: fmt.Sprintf("Failed to check availability: %v", err),
		}
	}

	// Return the successful server response
	return serverResponse
}

func sendOptimize(jsonData map[string]interface{}, c pb.SubmissionServiceClient) (*pb.OptimizationResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	data, err := json.Marshal(jsonData)
	if err != nil {
		log.Fatalf("failed to marshal problem data to JSON: %v", err)
	}
	fmt.Println(string(data))
	req := &pb.OptimizationRequest{
		JsonProblemData: string(data),
	}
	// Call the Optimize RPC method
	log.Printf("Sending optimization request...")
	res, err := c.Optimize(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("could not optimize: %v", err)
	}

	return res, nil
}

// processOptimizationResponse handles printing the results or error message from the response.
func processOptimizationResponse(res *pb.OptimizationResponse) OptimizationResults {
	results := OptimizationResults{} // Initialize the results struct

	if res.ErrorMessage != "" {
		// If there's an error message from the server

		log.Printf("Optimization returned an error message from the server: %s", res.ErrorMessage) // Log the server-side error
	} else if len(res.Solutions) > 0 {
		// If solutions were returned

		log.Printf("Optimization successful. Received %d solutions.", len(res.Solutions)) // Log success and count

		// Populate the Solutions slice in the results struct
		results.Solutions = make([]ProcessedSolution, len(res.Solutions))
		for i, sol := range res.Solutions {
			results.Solutions[i] = ProcessedSolution{
				SolNumber:         i,
				UserIDs:           sol.UserIds,
				TransportCost:     sol.TransportCost,
				ManufacturingCost: sol.ManufacturingCost,
			}
			// Note: The fmt.Printf lines that were here are removed.
		}
	} else {
		// If no solutions were returned and no explicit error message
		log.Println("Optimization finished but returned no solutions (might be infeasible).") // Log this case
	}

	return results // Return the populated struct
}

// connectToServer establishes a gRPC connection to the specified address.
func connectToServer(address string) (*grpc.ClientConn, error) {
	// Set up a connection to the server.
	// Use grpc.WithTransportCredentials(insecure.NewCredentials()) for development/testing without TLS
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}
	return conn, nil
}

func customerSearch(product_match string, PClasses *PClasses, dir *Directory) (ResultCollection, map[string][]string) {
	var wg sync.WaitGroup
	endMap := make(map[string][]string)
	resultCollectionLog := ResultCollection{}
	results_log := make(chan Result)
	for _, PClass := range PClasses.PClasses {
		if PClass.PClass == product_match {
			for _, customer := range PClass.Users {
				wg.Add(1)
				go func(customer Customer) {
					defer wg.Done() // Ensure goroutine is marked as done

					distanceMap := make(map[string]float64)
					for _, contact := range dir.Contacts {
						distanceMap[contact.Name] = Haversine(contact.Location, customer.Location)
					}
					// Establish a gRPC connection to the contact's server
					conn, err := grpc.NewClient(customer.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						return
					}
					defer conn.Close()

					// Create a client
					client := pb.NewSubmissionServiceClient(conn)

					response := CheckCustomer(client, product_match)

					if response.Capability {
						results_log <- Result{
							ContactName: customer.UID,
							Matches:     []string{"end"},
							Capability:  true,
						}
						return

					}
				}(customer)

			}
			go func() {
				wg.Wait()
				close(results_log) // Close channel when done
			}()
			for res := range results_log {
				resultCollectionLog.Results = append(resultCollectionLog.Results, res)
			}
			if len(resultCollectionLog.Results) == 0 {
				fmt.Println("No results received.")
			} else {
				fmt.Println("Results received:", resultCollectionLog.Results)
				break
			}

		}
	}

	// Collect results
	for _, item := range resultCollectionLog.Results {
		endMap["end"] = append(endMap["end"], item.ContactName)
	}
	return resultCollectionLog, endMap
}
