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
	ContactName string
	Matches     []string
	Location    []float64
	Response    string
}

type RouteRequest struct {
	ProcessReq    ProcessRequirements  `json:"process_requirements"`
	PurchaseReq   PurchaseRequirements `json:"purchase_requirements"`
	Routes        map[string][]string  `json:"routes"`
	StartingPoint []float64            `json:"starting_point"`
	OptMode       string               `json:"optimization_mode"`
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

// OptimizationResultForSequence holds the result for a single optimization run for a specific process sequence.
type OptimizationResultForSequence struct {
	ProcessSequence  []string            `json:"processSequence"`                   // The sequence of processes for this run
	OptimizationJSON OptimizationResults `json:"optimizationResultsJson,omitempty"` // The JSON results from the optimization service
	Error            string              `json:"error,omitempty"`                   // Any error encountered during this specific run
}

// CombinedOptimizationResponse holds the results from all concurrent optimization runs.
type CombinedOptimizationResponse struct {
	Message string                          `json:"message"` // A general message for the overall request
	Results []OptimizationResultForSequence `json:"results"` // A list of results for each sequence
}

// This struct contains the data required for seller and purchase matchmaking
type PurchaseRequirements struct {
	ProductMatch string `json:"product_match"`
	Quantity     int    `json:"quantity"`
}

// This stuct contains the data required for tech provider matchmaking
type ProcessRequirements struct {
	ProductMatch string  `json:"product_match"`
	Quantity     int     `json:"quantity"`
	AxisHeigth   float64 `json:"axisheigth"`
}

// ProcessDirectory using goroutines
func ProcessDirectory(c *gin.Context) {
	var request RouteRequest

	// Bind JSON to the struct
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid JSON format: %v", err)})
		return
	}

	processRequirements := request.ProcessReq
	processToFetch := GetDistinct(request.Routes)

	resultsChan := make(chan Result, len(directory.Contacts))

	for _, contact := range directory.Contacts {
		go func(contact Contact) {

			address := contact.Address
			prList := FindCommon(processToFetch, contact.Offerings)

			conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Failed to connect to %s (%s): %v", contact.Name, address, err) // Log the specific failure
				resultsChan <- Result{
					ContactName: contact.Name,
					Response:    fmt.Sprintf("Failed to connect to %s: %v", address, err), // Send error message in Response field
				}
				return
			}
			defer conn.Close()

			// Create a client
			client := pb.NewSubmissionServiceClient(conn)

			// Ping the server with context
			response, err := Ping(client, prList, processRequirements.ProductMatch)
			if err != nil {
				log.Printf("gRPC Ping failed for %s (%s): %v", contact.Name, address, err) // Log the specific failure
				resultsChan <- Result{
					ContactName: contact.Name,
					Response:    fmt.Sprintf("gRPC Ping failed: %v", err), // Send error message
				}
				return // Exit this goroutine
			}

			resultsChan <- Result{
				ContactName: contact.Name,
				Matches:     response.Capability,
				Location:    contact.Location,
				Response:    "Success",
			}
		}(contact)
	}

	resultCollection := ResultCollection{}

	for i := 0; i < len(directory.Contacts); i++ {
		resultCollection.Results = append(resultCollection.Results, <-resultsChan)
	}

	// Changes for the different optimization modes
	if request.OptMode == "forward" {
		resCollectionCustomer, _ := customerSearch(processRequirements.ProductMatch, pclasses)

		originProcess := ResultCollection{}
		originProcess.Results = append(originProcess.Results, Result{ContactName: "origin",
			Matches: []string{"origin"}, Location: request.StartingPoint})

		resultCollection.Results = append(resultCollection.Results, resCollectionCustomer.Results...) // Append customer results to the collection

		resultCollection.Results = append(resultCollection.Results, originProcess.Results...)

	} else if request.OptMode == "reverse" {
		purchaseRequirements := request.PurchaseReq
		resCollectionSeller, _ := providerSearch(purchaseRequirements, providersDir)
		endProcess := ResultCollection{}
		endProcess.Results = append(endProcess.Results, Result{ContactName: "end_customer_placeholder",
			Matches: []string{"end"}, Location: request.StartingPoint})
		resultCollection.Results = append(resultCollection.Results, resCollectionSeller.Results...) // Append customer results to the collection

		resultCollection.Results = append(resultCollection.Results, endProcess.Results...)

	} else {
		log.Printf("Invalid optimization mode: ")
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Invalid optimization type")})

		return
	}

	// --- Create Cost Matrix and Distance Matrix ---
	CostMatrix, err := CreateCostMatrixFromResults(resultCollection, costs)
	if err != nil {
		log.Printf("Error creating CostMatrix: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create cost matrix: %v", err)})
		return
	}
	fmt.Println("results", resultCollection.Results)

	distMatrix := DistanceMatrixConstructor(resultCollection)

	// Optimizer connection (running on docker as well)
	conn, err := connectToServer("optimizer:50060")
	if err != nil {
		log.Printf("Failed to connect to pymoo-runner: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to optimization service: %v", err)})
		return
	}
	defer conn.Close()

	// Create the optimization client
	clientOpt := pb.NewSubmissionServiceClient(conn)
	numSequences := len(request.Routes)
	optimizationResultsChan := make(chan OptimizationResultForSequence, numSequences)
	var wgOptimization sync.WaitGroup

	// Optimization process to be optimized
	for _, processSequence := range request.Routes {
		wgOptimization.Add(1)
		go func(sequence []string) {
			defer wgOptimization.Done()
			optimizationProcesses := make([]string, 0, len(sequence)+2)
			optimizationProcesses = append(optimizationProcesses, "origin")
			optimizationProcesses = append(optimizationProcesses, sequence...)
			optimizationProcesses = append(optimizationProcesses, "end")

			jsonData := exportToJson(optimizationProcesses, CostMatrix, distMatrix) // Pass the specific sequence

			res, err := sendOptimize(jsonData, clientOpt)
			if err != nil {
				log.Printf("Failed to send optimization request for sequence %v: %v", sequence, err)
				optimizationResultsChan <- OptimizationResultForSequence{
					ProcessSequence: sequence,
					Error:           fmt.Sprintf("Optimization request failed: %v", err),
				}
				return
			}

			// --- Process Optimization Response for THIS sequence ---
			optResultsJSON, err := processOptimizationResponse(res)
			if err != nil {
				log.Printf("Error processing optimization response for sequence %v: %v", sequence, err)
				optimizationResultsChan <- OptimizationResultForSequence{
					ProcessSequence: sequence,
					Error:           fmt.Sprintf("Failed to process optimization results: %v", err),
				}
				return // Exit this goroutine
			}

			// --- Send the result for THIS sequence to the channel ---
			optimizationResultsChan <- OptimizationResultForSequence{
				ProcessSequence:  sequence,
				OptimizationJSON: optResultsJSON,
				Error:            "",
			}

		}(processSequence)
	}

	wgOptimization.Wait()
	close(optimizationResultsChan)

	finalResponse := CombinedOptimizationResponse{
		Message: "Optimization process initiated for multiple sequences.",
		Results: []OptimizationResultForSequence{},
	}

	for result := range optimizationResultsChan {
		finalResponse.Results = append(finalResponse.Results, result)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "Optimization completed successfully",
		"optimizationResults": finalResponse, // Embed the JSON string as raw JSON
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

	for _, steps := range routes {
		for _, kl := range steps {
			uniqueKL[kl] = true
		}
	}

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

	serverResponse, err := c.CheckAvailabilty(ctx, process)
	if err != nil {
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

	serverResponse, err := c.CheckInterest(ctx, process)
	if err != nil {
		return &pb.PurchaseResponse{
			Status:  "Error",
			Message: fmt.Sprintf("Failed to check availability: %v", err),
		}
	}

	// Return the successful server response
	return serverResponse
}

func sendOptimize(jsonData map[string]interface{}, c pb.SubmissionServiceClient) (*pb.OptimizationResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	data, err := json.Marshal(jsonData)
	if err != nil {
		log.Fatalf("failed to marshal problem data to JSON: %v", err)
	}
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

		log.Printf("Optimization returned an error message from the server: %s", res.ErrorMessage) // Log the server-side error
		return results, fmt.Errorf(res.ErrorMessage)
	} else if len(res.Solutions) > 0 {

		log.Printf("Optimization successful. Received %d solutions.", len(res.Solutions)) // Log success and count

		results.Solutions = make([]ProcessedSolution, len(res.Solutions))
		for i, sol := range res.Solutions {
			results.Solutions[i] = ProcessedSolution{
				SolNumber:         i,
				UserIDs:           sol.UserIds,
				TransportCost:     sol.TransportCost,
				ManufacturingCost: sol.ManufacturingCost,
			}
		}
	} else {
		log.Println("Optimization finished but returned no solutions (might be infeasible).") // Log this case
	}

	return results, nil // Return the populated struct
}

func connectToServer(address string) (*grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}
	return conn, nil
}

func customerSearch(product_match string, Interests *CustomerInterests) (ResultCollection, map[string][]string) {
	var wg sync.WaitGroup
	endMap := make(map[string][]string)
	resultCollectionLog := ResultCollection{}
	results_log := make(chan Result)

	for _, cust := range Interests.Customers {
		if !slices.Contains(cust.PClasses, product_match) {
			continue
		}
		c := cust
		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := grpc.NewClient(c.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return
			}
			defer conn.Close()

			client := pb.NewSubmissionServiceClient(conn)
			resp := CheckCustomer(client, product_match)
			if resp.Capability {
				results_log <- Result{
					ContactName: c.Name,
					Matches:     []string{"end"},
					Location:    c.Location,
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results_log)
	}()
	for res := range results_log {
		resultCollectionLog.Results = append(resultCollectionLog.Results, res)
	}
	if len(resultCollectionLog.Results) == 0 {
		fmt.Println("No results received.")
	}
	for _, item := range resultCollectionLog.Results {
		endMap["end"] = append(endMap["end"], item.ContactName)
	}

	return resultCollectionLog, endMap
}
func providerSearch(requirements PurchaseRequirements, Providers *ProviderDirectory) (ResultCollection, map[string][]string) {
	var wg sync.WaitGroup
	originMap := make(map[string][]string)
	resultCollectionLog := ResultCollection{}
	results_log := make(chan Result)

	for _, prov := range Providers.Providers {
		if !slices.Contains(prov.PClasses, requirements.ProductMatch) {
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			response, _ := inspectProvider(requirements, prov.Address)
			if response {
				results_log <- Result{
					ContactName: prov.Name,
					Matches:     []string{"origin"},
					Location:    prov.Location,
				}
			}
		}()
	}
	go func() {
		wg.Wait()
		close(results_log)
	}()
	for res := range results_log {
		resultCollectionLog.Results = append(resultCollectionLog.Results, res)
	}
	if len(resultCollectionLog.Results) == 0 {
		fmt.Println("No results received.")
	}
	for _, item := range resultCollectionLog.Results {
		originMap["origin"] = append(originMap["origin"], item.ContactName)
	}

	return resultCollectionLog, originMap
}

func inspectProvider(requirements PurchaseRequirements, filename string) (bool, error) {
	providerData, err := getProvider(filename)
	if err != nil {
		log.Printf("Unable to process provider data %v", err)
		return false, err
	}
	for _, motor := range providerData.Motors {
		if motor.TechnicalData.EfficiencyClass == requirements.ProductMatch {
			if motor.Stock >= requirements.Quantity {
				return true, nil
			}
		}
	}
	return false, nil
}
