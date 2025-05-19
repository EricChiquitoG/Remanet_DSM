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
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid JSON format: %v", err)})
		return
	}

	product_match := request.ProductMatch
	processToFetch := GetDistinct(request.Routes) // Assuming request.Routes is the source

	// --- Load Data from Files ---
	// Replace log.Fatalf with error checking and JSON responses

	dir, err := MyDir("./data/directory.json")
	if err != nil {
		log.Printf("Error loading directory.json: %v", err) // Log the error on the server side
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load directory data: %v", err)})
		return // Stop processing and return error response
	}

	addLoc, err := MyLocs("./data/locationAdd.json")
	if err != nil {
		log.Printf("Error loading locationAdd.json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load location data: %v", err)})
		return
	}

	PClasses, err := getInterests("./data/interests.json")
	if err != nil {
		log.Printf("Error loading interests.json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load interests data: %v", err)})
		return
	}

	costs, err := Costs("./data/cost.json")
	if err != nil {
		log.Printf("Error loading cost.json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load cost data: %v", err)})
		return
	}

	// --- Process Contacts using Goroutines ---
	// Correctly collect results from goroutines

	// Use a buffered channel if you know the number of contacts, or unbuffered is fine too.
	// The size should be at least the number of goroutines launched.
	resultsChan := make(chan Result, len(dir.Contacts))

	// Launch a goroutine for each contact
	for _, contact := range dir.Contacts {
		go func(contact Contact) {

			address := contact.Address
			prList := FindCommon(processToFetch, contact.Offerings)

			// Add a timeout context for the gRPC dial and call
			// Establish a gRPC connection to the contact's server with context
			conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to connect to %s (%s): %v", contact.Name, address, err) // Log the specific failure
				resultsChan <- Result{
					ContactName: contact.Name,
					Response:    fmt.Sprintf("Failed to connect to %s: %v", address, err), // Send error message in Response field
				}
				return // Exit this goroutine
			}
			defer conn.Close() // Ensure connection is closed

			// Create a client
			client := pb.NewSubmissionServiceClient(conn) // Assuming this is the correct client type for Ping

			// Ping the server with context
			response, err := Ping(client, prList, product_match)
			if err != nil {
				log.Printf("gRPC Ping failed for %s (%s): %v", contact.Name, address, err) // Log the specific failure
				resultsChan <- Result{
					ContactName: contact.Name,
					Response:    fmt.Sprintf("gRPC Ping failed: %v", err), // Send error message
				}
				return // Exit this goroutine
			}

			// If successful, send the result to the channel
			resultsChan <- Result{
				ContactName: contact.Name,
				Matches:     response.Capability, // Assuming response has a Capability field
				Response:    "Success",           // Indicate success
			}
		}(contact) // Pass the contact variable to the goroutine
	}

	// Collect results from the channel
	resultCollection := ResultCollection{}

	// If not using WaitGroup (simpler for fixed number of goroutines), collect exactly len(dir.Contacts) results:
	for i := 0; i < len(dir.Contacts); i++ {
		resultCollection.Results = append(resultCollection.Results, <-resultsChan)
	}

	// --- Process Customer Search ---
	resCollectionCustomer, custResponses := customerSearch(product_match, PClasses)

	originProcess := ResultCollection{}
	originProcess.Results = append(originProcess.Results, Result{ContactName: "origin",
		Matches:    []string{"origin"},
		Capability: true})

	// --- Combine Results ---
	resultMap := CreateMap(processToFetch, resultCollection) // Create map from contact results
	for key, val := range custResponses {
		resultMap[key] = val // Append customer results to the map
	}
	resultCollection.Results = append(resultCollection.Results, resCollectionCustomer.Results...) // Append customer results to the collection

	resultCollection.Results = append(resultCollection.Results, originProcess.Results...)
	// --- Create Cost Matrix and Distance Matrix ---
	CostMatrix, err := CreateCostMatrixFromResults(resultCollection, costs)
	if err != nil {
		log.Printf("Error creating CostMatrix: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create cost matrix: %v", err)})
		return
	}

	origin := request.StartingPoint                         // Assuming StartingPoint is in RouteRequest
	distMatrix := DistanceMatrixConstructor(origin, addLoc) // Assuming this function exists and works

	// --- Prepare and Send Optimization Request to Pymoo Runner ---
	// Replace log.Fatalf with error checking and JSON responses

	// Prepare JSON data for the optimization service
	// Ensure processToFetch includes all necessary processes for the optimization service
	optimizationProcesses := make([]string, 0, len(processToFetch)+2)
	optimizationProcesses = append(optimizationProcesses, "origin")
	optimizationProcesses = append(optimizationProcesses, processToFetch...)
	optimizationProcesses = append(optimizationProcesses, "end")
	fmt.Println(optimizationProcesses)
	jsonData := exportToJson(optimizationProcesses, CostMatrix, distMatrix)

	// Connect to the pymoo-runner gRPC server with context
	// Replace "pymoo-runner:50060" with the correct address (e.g., service name in Docker Compose)
	conn, err := connectToServer("pymoo-runner:50060")
	if err != nil {
		log.Printf("Failed to connect to pymoo-runner: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to optimization service: %v", err)})
		return
	}
	defer conn.Close()

	// Create the optimization client
	clientOpt := pb.NewSubmissionServiceClient(conn) // Assuming OptimizationService is defined in your proto

	// Send the optimization request with context
	fmt.Println("sending", jsonData)              // Keep for debugging if needed
	res, err := sendOptimize(jsonData, clientOpt) // Pass context and clientOpt
	if err != nil {
		log.Printf("Failed to send optimization request to pymoo-runner: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Optimization request failed: %v", err)})
		return
	}

	// --- Process Optimization Response and Return to Client ---
	// processOptimizationResponse now returns JSON string and error
	optResultsJSON, err := processOptimizationResponse(res)
	if err != nil {
		log.Printf("Error processing optimization response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to process optimization results: %v", err)})
		return
	}

	// Return success response with optimization results JSON
	// Use c.Data or c.String if you want to return raw JSON string directly
	// Or if you want to wrap it in another JSON object:
	c.JSON(http.StatusOK, gin.H{
		"message":             "Optimization completed successfully",
		"optimizationResults": optResultsJSON, // Embed the JSON string as raw JSON
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

func Ping(c pb.SubmissionServiceClient, tasks []string, product string) (*pb.ProcessResponse, error) {
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
		}, err
	}

	// Return the successful server response
	return serverResponse, nil
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
func processOptimizationResponse(res *pb.OptimizationResponse) (OptimizationResults, error) {
	results := OptimizationResults{} // Initialize the results struct

	if res.ErrorMessage != "" {
		// If there's an error message from the server

		log.Printf("Optimization returned an error message from the server: %s", res.ErrorMessage) // Log the server-side error
		return results, fmt.Errorf(res.ErrorMessage)
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

	return results, nil // Return the populated struct
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

func customerSearch(product_match string, PClasses *PClasses) (ResultCollection, map[string][]string) {
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
