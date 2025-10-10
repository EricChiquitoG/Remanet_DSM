package src

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// New structs for input to be used for MVP:
type OptimizationInput struct {
	Batch               int                 `json:"Batch"`
	ProcessRequirements ProcessRequirements `json:"process_requirements"`
	Routes              map[string][]Step   `json:"routes"`
	StartingPoint       []float64           `json:"starting_point"`
	OptimizationMode    string              `json:"optimization_mode"`
}

type ProcessRequirements struct {
	ProductMatch string `json:"product_match"`
	Quantity     int    `json:"quantity"`
}

type Step struct {
	Name         string       `json:"name"`
	Requirements Requirements `json:"Requirements"`
}

type Requirements struct {
	AxisHeight               *float64 `json:"axis_height,omitempty"`
	Power                    *float64 `json:"power,omitempty"`
	KnowHowFormalizationTool *bool    `json:"know-how_formalization-tool,omitempty"`
	Tolerance                *string  `json:"tolerance,omitempty"`
	Length                   *int     `json:"length,omitempty"`
	Diameter                 *int     `json:"diameter,omitempty"`
}

//End of new structs

type ResultCollection struct {
	Results []Result
}

type Result struct {
	ContactName string
	Matches     []string
	Location    []float64
	Response    string
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
	Error            string              `json:"error,omitempty"`                   // Any error encountered during this specific run<
}

// CombinedOptimizationResponse holds the results from all concurrent optimization runs.
type CombinedOptimizationResponse struct {
	Message string                          `json:"message"` // A general message for the overall request
	Results []OptimizationResultForSequence `json:"results"` // A list of results for each sequence
}

// This stuct contains the data required for tech provider matchmaking

type responsesTP struct {
	Responses []techProviderResponse
}

type techProviderResponse struct {
	uid       string
	uClass    string
	location  []float64
	Processes []process
}

type process struct {
	ProcessID   string
	ProcessName string
	Cost        float64
	Co2         float64
}

type StepEntry struct {
	StepName     string       `json:"step_name"`
	Requirements Requirements `json:"requirements"`
}

// Structs retrieve from extraDB
type APIResponse struct {
	Elements []MotorElement `json:"elements"`
	Count    int            `json:"count"`
}

type MotorElement struct {
	ID              int      `json:"id"`
	OrganizationID  string   `json:"organizationId"`
	UserName        string   `json:"userName"`
	NumberOfMotors  int      `json:"numberOfMotors"`
	AxisHeight      float64  `json:"axisHeight"`
	RatedPower      float64  `json:"ratedPower"`
	Point           GeoPoint `json:"point"`
	EfficiencyClass string   `json:"efficiencyClass"`
}

type GeoPoint struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

// End of extra structs

// ProcessDirectory using goroutines
func ProcessDirectory(c *gin.Context) {
	var request OptimizationInput

	// Bind JSON to the struct
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid JSON format: %v", err)})
		return
	}

	processToFetch := GetDistinct(request)
	fmt.Println("DSM process initialized for processes: ", processToFetch)
	AClassesToFetch := IdentifyActorClasses(processToFetch, ActorProcesses)

	uniqueSteps := FlattenUniqueStepsWithData(request)
	TPResponses := techProviderSearch(AClassesToFetch, uniqueSteps, directory)
	//fmt.Println("tech providers  ", TPResponses)
	resultTP := ConvertTPToResultCollection(TPResponses)
	//fmt.Println("tech providers converted   ", resultTP)

	resultCollection := ResultCollection{}
	resultCollection.Results = append(resultCollection.Results, resultTP.Results...)
	// Changes for the different optimization modes
	if request.OptimizationMode == "forward" {
		resCollectionCustomer, _ := customerSearch(request.ProcessRequirements.ProductMatch, pclasses)

		originProcess := ResultCollection{}
		originProcess.Results = append(originProcess.Results, Result{ContactName: "origin",
			Matches: []string{"origin"}, Location: request.StartingPoint})

		resultCollection.Results = append(resultCollection.Results, resCollectionCustomer.Results...) // Append customer results to the collection

		resultCollection.Results = append(resultCollection.Results, originProcess.Results...)

	} else if request.OptimizationMode == "reverse" {
		purchaseRequirements := request.ProcessRequirements
		resCollectionSeller, _, _ := providerSearch(purchaseRequirements)
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

	//Hardcoded information regarding the redesign process with dummy data
	if slices.Contains(processToFetch, "Redesign") {
		redesignProcess := ResultCollection{}
		redesignProcess.Results = append(redesignProcess.Results, Result{ContactName: "SPIN_REDESIGN",
			Matches: []string{"Redesign"}, Location: request.StartingPoint})
		resultCollection.Results = append(resultCollection.Results, redesignProcess.Results...)
	}

	// --- Create Cost Matrix and Distance Matrix ---
	CostMatrix, err := CreateCostMatrixFromResults(resultCollection, costs, &TPResponses)
	if err != nil {
		log.Printf("Error creating CostMatrix: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create cost matrix: %v", err)})
		return
	}
	//fmt.Println("results", resultCollection.Results)

	distMatrix := DistanceMatrixConstructor(resultCollection, directory)

	optimizationResponse, err := OptimizationHandler(request, distMatrix, CostMatrix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to connect to optimization service: %v", err)})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":             "Optimization completed successfully",
		"optimizationResults": optimizationResponse, // Embed the JSON string as raw JSON
	})
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

// Change this function to be able to query the dummy reverse data for DSM (shall extra not provide it)
func providerSearch(requirements ProcessRequirements) (ResultCollection, map[string][]string, error) {
	originMap := make(map[string][]string)
	resultCollectionLog := ResultCollection{}
	baseURL := "https://platform.remanet.extrared.it/backend/gateway/coordination/purchase-proposal/v1"
	matches, err := retrieveExternalAPI(baseURL, requirements.ProductMatch, "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJtN0hWNUMzRTE1WWQzc250QnROc0pqZkVtakxuMkFTc0FJM1FoVmNKSldFIn0.eyJleHAiOjE3NjAwODI4MTcsImlhdCI6MTc2MDA4MjUxNywiYXV0aF90aW1lIjoxNzYwMDgxOTMxLCJqdGkiOiI4MzNhODBjOS1lMWUzLTQ4NzgtYjNhNi0xYWFmODc1MDFjOWMiLCJpc3MiOiJodHRwczovL2tleWNsb2FrLnJlbWFuZXQuZXh0cmFyZWQuaXQvcmVhbG1zL3JlbWFuZXRQbGF0Zm9ybVdlYlVJIiwiYXVkIjoiYWNjb3VudCIsInN1YiI6ImU5NzllNjhiLWUyMDYtNDJmNS1hZmY4LTZmM2JiNjNmN2FlOSIsInR5cCI6IkJlYXJlciIsImF6cCI6InJlbWFuZXQtZmUiLCJzaWQiOiIzMzk0ODMxNC0yM2U2LTQ4NGEtYjdhMi1iODAwNTQwNjgzZjYiLCJhY3IiOiIwIiwiYWxsb3dlZC1vcmlnaW5zIjpbImh0dHBzOi8vcGxhdGZvcm0ucmVtYW5ldC5leHRyYXJlZC5pdCIsImh0dHA6Ly9sb2NhbGhvc3Q6NDIwMCIsImh0dHBzOi8vcmVtYW5ldC1mZSIsImh0dHA6Ly9wbGF0Zm9ybS5yZW1hbmV0LmV4dHJhcmVkLml0Il0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJvZmZsaW5lX2FjY2VzcyIsImRlZmF1bHQtcm9sZXMtcmVtYW5ldHBsYXRmb3Jtd2VidWkiLCJ1bWFfYXV0aG9yaXphdGlvbiJdfSwicmVzb3VyY2VfYWNjZXNzIjp7ImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInNjb3BlIjoib3BlbmlkIHByb2ZpbGUgb3JnYW5pemF0aW9uIGVtYWlsIiwiZW1haWxfdmVyaWZpZWQiOnRydWUsIm9yZ2FuaXphdGlvbiI6WyJleHRyYXJlZCJdLCJuYW1lIjoiRXJpYyBDaGlxdWl0byIsImdyb3VwcyI6WyIvZWxlY3RyaWNhbC1tb3RvcnMiXSwicHJlZmVycmVkX3VzZXJuYW1lIjoiZWNoaXF1aXRvIiwiZ2l2ZW5fbmFtZSI6IkVyaWMiLCJmYW1pbHlfbmFtZSI6IkNoaXF1aXRvIiwiZW1haWwiOiJlcmljLmNoaXF1aXRvQGx0dS5zZSJ9.VB1UFRAAA6vcEN_YWrd0yWtiGWrl3Q_oI0gq04c1EAvt7_eQTDOYCk34hLsXCaivaKlIgTpS5DFk5O8u5r0aNzvO-9qIsgiHYnnDctHjQRFJq7K_sGRXPkgWRMoHmljvWmjUUmZlwYUpN-4qEkS-2rK9ASwBxSvzccIUdmyGOhy2uQwoxjW5dt6ntun4-2O2Ni9UeXfwegO_6iL2q1bvd9j5_OFe2hOL1lzYdowpS4DQn0vPY6RBwnB6VTDhy8BIe99G5ZTo7q-0LPE2Ux8HvWUuk2E9IRjX0CxZxYmpUrkhFg9gm8iMWnvZMXlRGHCI2ggxAf9wowUoZalVAw4uuA")
	if err != nil {
		log.Printf("Retrieval from external API did not work: %s", err)
		return resultCollectionLog, originMap, err
	}
	fmt.Println("MAtches extra", matches)
	for _, match := range matches {
		res := Result{ContactName: match.OrganizationID, Matches: []string{"origin"}, Location: []float64{match.Point.Coordinates[0], match.Point.Coordinates[1]}}
		resultCollectionLog.Results = append(resultCollectionLog.Results, res)
	}
	if len(resultCollectionLog.Results) == 0 {
		fmt.Println("No results received.")
	}
	for _, item := range resultCollectionLog.Results {
		originMap["origin"] = append(originMap["origin"], item.ContactName)
	}

	return resultCollectionLog, originMap, nil
}

// Generalized to call API but hardcoded for extra, TODO, generalize
func retrieveExternalAPI(baseURL, PClass, jwt string) ([]MotorElement, error) {
	params := url.Values{}
	params.Set("efficiencyClass", PClass)
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error: %s - %s", resp.Status, string(body))
	}

	// Read and decode the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	return apiResp.Elements, nil
}

func ExtractStepNamesFromRoutes(r OptimizationInput) []string {
	stepNameSet := make(map[string]struct{})

	for _, route := range r.Routes {
		for _, stepName := range route {
			stepNameSet[stepName.Name] = struct{}{}
		}
	}

	stepNames := make([]string, 0, len(stepNameSet))
	for name := range stepNameSet {
		stepNames = append(stepNames, name)
	}

	return stepNames
}

func OptimizationHandler(r OptimizationInput, distMatrix map[string]map[string]float64, costMatrix map[string]map[string]float64) (CombinedOptimizationResponse, error) {

	// Optimizer connection (running on docker as well)

	finalResponse := CombinedOptimizationResponse{
		Message: "Optimization process initiated for multiple sequences.",
		Results: []OptimizationResultForSequence{},
	}
	conn, err := connectToServer("optimizer:50060")
	if err != nil {
		log.Printf("Failed to connect to pymoo-runner: %v", err)
		return finalResponse, err
	}
	defer conn.Close()

	// Create the optimization client
	clientOpt := pb.NewSubmissionServiceClient(conn)
	numSequences := len(r.Routes)
	optimizationResultsChan := make(chan OptimizationResultForSequence, numSequences)
	var wgOptimization sync.WaitGroup
	fmt.Print("rutas ", r.Routes)
	// Optimization process to be optimized
	for _, processSequence := range r.Routes {
		sequence := make([]string, 0, len(processSequence))
		for _, stepName := range processSequence {
			sequence = append(sequence, stepName.Name)
		}

		wgOptimization.Add(1)
		go func(sequence []string) {
			defer wgOptimization.Done()
			var i int // only needed to move the redesign step, will most likely delete later
			optimizationProcesses := make([]string, 0, len(sequence)+2)
			optimizationProcesses = append(optimizationProcesses, "origin")
			optimizationProcesses = append(optimizationProcesses, sequence...)
			optimizationProcesses = append(optimizationProcesses, "end")
			if contains(optimizationProcesses, "Redesign") {

				optimizationProcesses, i = moveFlagToEnd(optimizationProcesses, "Redesign")
			}
			fmt.Println("processes ordered? ", optimizationProcesses)
			jsonData := exportToJson(optimizationProcesses, costMatrix, distMatrix) // Pass the specific sequence

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
			if contains(optimizationProcesses, "Redesign") {
				for _, solution := range optResultsJSON.Solutions {
					solution.UserIDs = moveEndToIndex(solution.UserIDs, i)
				}
			}

			// --- Send the result for THIS sequence to the channel ---
			optimizationResultsChan <- OptimizationResultForSequence{
				ProcessSequence:  sequence,
				OptimizationJSON: optResultsJSON,
				Error:            "",
			}

		}(sequence)
	}
	wgOptimization.Wait()
	close(optimizationResultsChan)

	for result := range optimizationResultsChan {
		finalResponse.Results = append(finalResponse.Results, result)
	}
	return finalResponse, nil

}

func techProviderSearch(AClasses []string, steps []StepEntry, dir *Directory) responsesTP {
	var wg sync.WaitGroup
	resultsChan := make(chan techProviderResponse)
	for class, contacts := range dir.Contacts {
		if !slices.Contains(AClasses, class) {
			continue
		}

		for _, contact := range contacts {
			class := class     // shadow variable to avoid closure capture
			contact := contact // same here

			wg.Add(1)
			go func(class string, contact Contact) {
				defer wg.Done()

				resp, err := callTechProvider(class, steps, contact.Address, contact.Name, contact.Location)
				if err != nil {
					log.Printf("Error calling tech provider %s (%s): %v", contact.Name, class, err)
					return
				}

				if len(resp.Processes) > 0 {
					resultsChan <- resp
				}
			}(class, contact) // pass variables here explicitly
		}
	}
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	finalResponse := responsesTP{}

	for result := range resultsChan {
		finalResponse.Responses = append(finalResponse.Responses, result)
	}

	if len(finalResponse.Responses) == 0 {
		fmt.Println("⚠️ No technology providers matched any processes.")
	}

	return finalResponse
}

// This function is called once per technology provider to retrieve their data
func callTechProvider(class string, steps []StepEntry, address string, name string, loc []float64) (techProviderResponse, error) {
	response := techProviderResponse{uClass: class, uid: name, location: loc}
	providerData, err := fetchAAS(address)

	if err != nil {
		log.Printf("Unable to process provider data %v", err)
		return response, err
	}
	processLookup := make(map[string]bool)
	if allowed, ok := ActorProcesses[class]; ok {
		for _, p := range allowed {
			processLookup[p] = true
		}
	}
	for _, step := range steps {
		stepName := step.StepName
		if !processLookup[stepName] {
			continue
		}
		reqMatched := MatchesStepRequirements(class, *providerData, step.Requirements)
		if reqMatched {
			cost := getFloatFromAAS(*providerData, "cost_per_hour")
			co2 := getFloatFromAAS(*providerData, "co2_static_per_hour")
			//loc1 := getFloatFromAAS(*providerData, "longitude") * 100
			//loc2 := getFloatFromAAS(*providerData, "latitude") * 100

			newProcess := process{
				ProcessID:   fmt.Sprintf("%s-%s", name, stepName),
				ProcessName: stepName,
				Cost:        cost,
				Co2:         co2,
			}
			//response.location = append(response.location, loc1, loc2)
			response.Processes = append(response.Processes, newProcess)
		}
	}

	return response, nil
}

func fetchAAS(id string) (*AASResponse, error) {
	baseURL := "https://ds.platform.remanet.extrared.it/ds4/retrieve_by_id"

	// Encode the path variable in case it has special characters like ":" or "/"
	encodedID := url.PathEscape(id)

	// Construct the full URL
	fullURL := fmt.Sprintf("%s/%s", baseURL, encodedID)

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var aas AASResponse
	if err := json.NewDecoder(resp.Body).Decode(&aas); err != nil {
		return nil, err
	}
	return &aas, nil
}

func MatchesStepRequirements(actorClass string, aas AASResponse, req Requirements) bool {
	switch actorClass {
	case "A":
		fmt.Println("Class A Matching:")
		return matchClassA(aas, req)
	case "C":
		fmt.Println("Class C Matching:")
		return matchClassC(aas, req)
	case "D":
		fmt.Println("Class D Matching:")

		return matchClassD(aas, req)
	default:
		fmt.Println("Unknown actor class:", actorClass)
		return false
	}
}

// ---------- Matching per Class ----------

func matchClassA(aas AASResponse, req Requirements) bool {
	axisHeight := getFloatFromAAS(aas, "axis_height")
	power := getFloatFromAAS(aas, "power")
	KHFT := getBoolFromAAS(aas, "know-how_formalization-tool")
	fmt.Println("to match ", "AH: ", *req.AxisHeight, "Power: ", *req.Power, " With: ", "AH: ", axisHeight, "Power: ", power)
	if req.AxisHeight != nil && axisHeight < *req.AxisHeight {
		fmt.Println("Axis height too high for class A for actor ", aas.Data.IDShort)
		return false
	}
	if req.Power != nil && power < *req.Power {
		fmt.Println("Power too high for class A for actor ", aas.Data.IDShort)
		return false
	}
	if req.KnowHowFormalizationTool != nil && KHFT != *req.KnowHowFormalizationTool {
		fmt.Println("Know-how-formalization-tool property not matching for class A")
		return false
	}
	return true
}

func matchClassC(aas AASResponse, req Requirements) bool {
	tolerance := getStringFromAAS(aas, "tolerance")
	length := getFloatFromAAS(aas, "length")
	diameter := getFloatFromAAS(aas, "diameter")

	toleranceMapped := ProcessMap[tolerance]
	fmt.Println("to match ", "Tolerance: ", *req.Tolerance, "Length: ", *req.Length, "Diameter: ", *req.Diameter, " With: ", "Tolerance: ", toleranceMapped, "Length: ", length, "Diameter: ", diameter)
	if req.Tolerance != nil && ProcessMap[tolerance] < ProcessMap[*req.Tolerance] {
		fmt.Println("Tolerance mismatch for class C for actor ", aas.Data.IDShort)
		return false
	}
	if req.Length != nil && length < float64(*req.Length) {
		fmt.Println("Length too long for class C for actor ", aas.Data.IDShort)
		return false
	}
	if req.Diameter != nil && diameter < float64(*req.Diameter) {
		fmt.Println("Diameter too large for class C for actor ", aas.Data.IDShort)
		return false
	}
	return true
}

func matchClassD(aas AASResponse, req Requirements) bool {
	axisHeight := getFloatFromAAS(aas, "axis_height")
	fmt.Println("to match ", "AH: ", *req.AxisHeight, " With: ", "AH: ", axisHeight)
	if req.AxisHeight != nil && axisHeight < *req.AxisHeight {
		fmt.Println("Axis height too high for class D for actor ", aas.Data.IDShort)
		return false
	}
	return true
}

// Finds and parses a float64 property directly from AAS by name
func getFloatFromAAS(aas AASResponse, name string) float64 {
	name = strings.ToLower(name)
	for _, sm := range aas.Data.Submodels {
		for _, elem := range sm.SubmodelElements {
			if strings.EqualFold(elem.IDShort, name) {
				var s string
				if err := json.Unmarshal(elem.Value, &s); err == nil {
					if f, err := strconv.ParseFloat(s, 64); err == nil {
						return f
					}
				}
				var f float64
				if err := json.Unmarshal(elem.Value, &f); err == nil {
					return f
				}
			}
		}
	}
	return 0 // property not found
}

// Finds and parses a string property directly from AAS by name
func getStringFromAAS(aas AASResponse, name string) string {
	name = strings.ToLower(name)
	for _, sm := range aas.Data.Submodels {
		for _, elem := range sm.SubmodelElements {
			if strings.EqualFold(elem.IDShort, name) {
				var s string
				_ = json.Unmarshal(elem.Value, &s)
				return s
			}
		}
	}
	return ""
}

func getBoolFromAAS(aas AASResponse, name string) bool {
	name = strings.ToLower(name)
	for _, sm := range aas.Data.Submodels {
		for _, elem := range sm.SubmodelElements {
			if strings.EqualFold(elem.IDShort, name) {
				var s bool
				_ = json.Unmarshal(elem.Value, &s)
				return s
			}
		}
	}
	return false
}

func FlattenUniqueStepsWithData(input OptimizationInput) []StepEntry {
	uniqueSteps := make(map[string]Requirements)

	for _, route := range input.Routes {
		for _, step := range route {
			// If step not already seen, store it
			if _, exists := uniqueSteps[step.Name]; !exists {
				uniqueSteps[step.Name] = step.Requirements
			}
		}
	}

	result := make([]StepEntry, 0, len(uniqueSteps))
	for name, reqs := range uniqueSteps {
		result = append(result, StepEntry{
			StepName:     name,
			Requirements: reqs,
		})
	}
	return result
}

func ConvertTPToResultCollection(tp responsesTP) ResultCollection {
	resultCollection := ResultCollection{}

	for _, resp := range tp.Responses {
		var matches []string
		for _, p := range resp.Processes {
			matches = append(matches, p.ProcessName)
		}

		newResult := Result{
			ContactName: resp.uid,
			Matches:     matches,
			Location:    resp.location, // you can fill this later if needed
			Response:    resp.uClass,   // optional, using class as response label
		}

		resultCollection.Results = append(resultCollection.Results, newResult)
	}

	return resultCollection
}
