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

	"github.com/google/uuid"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// New structs for input to be used for MVP:
type OptimizationInput struct {
	Batch               int                           `json:"Batch"`
	ProcessRequirements ProcessRequirements           `json:"process_requirements"`
	Routes              map[string]map[string]DAGNode `json:"routes"`
	StartingPoint       []float64                     `json:"starting_point"`
	OptimizationMode    string                        `json:"optimization_mode"`
}

type ProcessRequirements struct {
	ProductMatch string `json:"product_match"`
	Quantity     int    `json:"quantity"`
}

// DAGNode represents a single process step in the workflow DAG.
// Prerequisites lists the process names that must complete before this one starts.
type DAGNode struct {
	Prerequisites []string     `json:"prerequisites"`
	Requirements  Requirements `json:"requirements"`
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

// Assignment pairs a process name with the actor assigned to execute it.
type Assignment struct {
	ProcessName string `json:"processName"`
	ActorID     string `json:"actorID"`
}

type ProcessedSolution struct {
	SolNumber         int
	Assignments       []Assignment // in topological execution order, boundary nodes excluded
	TransportCost     float64
	ManufacturingCost float64
}

// OptimizationResults holds the processed results from the gRPC response.
// This struct is returned by the processOptimizationResponse function.
type OptimizationResults struct {
	Solutions []ProcessedSolution // A list of the processed solutions found
}

// OptimizationResultForSequence holds the result for a single optimization run.
type OptimizationResultForSequence struct {
	ProcessID        string              `json:"processID"`
	ProcessSequence  []string            `json:"processSequence"`
	DAGPrereqs       map[string][]string `json:"dagPrereqs"`
	OptimizationJSON OptimizationResults `json:"optimizationResultsJson,omitempty"`
	Error            string              `json:"error,omitempty"`
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

// MotorElement maps the fields we need from a Proposal returned by GET /purchase_proposals.
type MotorElement struct {
	CustomerID         string   `json:"customerId"`
	CustomerLocation   GeoPoint `json:"customerLocation"`
	Cardinality        int      `json:"cardinality"`
	ProductType        string   `json:"productType"`
	RequiredParameters struct {
		RatedPower      float64 `json:"ratedPower"`
		AxisHeight      int     `json:"axisHeight"`
		EfficiencyClass string  `json:"efficiencyClass"`
	} `json:"requiredParameters"`
}

type GeoPoint struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

type SolChoice = struct {
	ProcessID string `json:"processID"`
	SolNumber int    `json:"solNumber"`
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

	// Gate on external service availability before doing any work.
	if ok, msg := IsServiceAvailable(ServiceAASAPI); !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("AAS API unavailable: %s", msg)})
		return
	}
	if ok, msg := IsServiceAvailable(ServiceOptimizer); !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("optimizer service unavailable: %s", msg)})
		return
	}
	if request.OptimizationMode == "forward" {
		if ok, msg := IsServiceAvailable(ServiceMongoDB); !ok {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("MongoDB unavailable: %s", msg)})
			return
		}
	}

	processToFetch := GetDistinct(request)
	fmt.Println("DSM process initialized for processes: ", processToFetch)
	AClassesToFetch := IdentifyActorClasses(processToFetch, ActorProcesses)

	uniqueSteps := FlattenUniqueStepsWithData(request)
	TPResponses, err := techProviderSearch(AClassesToFetch, uniqueSteps, directory)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Unable to connect to the AAS %v", err)})
		return
	}

	//fmt.Println("tech providers  ", TPResponses)
	resultTP := ConvertTPToResultCollection(TPResponses)
	//fmt.Println("tech providers converted   ", resultTP)

	resultCollection := ResultCollection{}
	resultCollection.Results = append(resultCollection.Results, resultTP.Results...)
	// Changes for the different optimization modes
	purchaseRequirements := request.ProcessRequirements

	if request.OptimizationMode == "forward" {
		resCollectionCustomer, _, _ := customerSearch(purchaseRequirements)

		// If no customer proposals matched, fall back to the starting point as the delivery destination.
		// This guarantees at least one actor capable of the "end" boundary step.
		if len(resCollectionCustomer.Results) == 0 {
			log.Printf("[forward] no customer proposals found — using starting point as end placeholder")
			resCollectionCustomer.Results = append(resCollectionCustomer.Results, Result{
				ContactName: "end_destination",
				Matches:     []string{"end"},
				Location:    request.StartingPoint,
			})
		}

		resultCollection.Results = append(resultCollection.Results, resCollectionCustomer.Results...)
		resultCollection.Results = append(resultCollection.Results, Result{
			ContactName: "origin",
			Matches:     []string{"origin"},
			Location:    request.StartingPoint,
		})

	} else if request.OptimizationMode == "reverse" {
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

func SelectOption(c *gin.Context) {
	var request SolChoice

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid JSON format: %v", err)})
		return
	}
	UserChoice, err := GetOptimizationResult(request.ProcessID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch: %v", err)})
	}
	for _, match := range UserChoice.OptimizationJSON.Solutions {
		if match.SolNumber == request.SolNumber {
			var firstActor string
			if len(match.Assignments) > 0 {
				firstActor = match.Assignments[0].ActorID
			}
			sample, err := GetTPData("TPRes_" + firstActor)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch: %v", err)})
				return
			}

			graph, err := BuildExecutionGraph(request.ProcessID, UserChoice.ProcessSequence, match, UserChoice.DAGPrereqs)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to build execution graph: %v", err)})
				return
			}
			if err := execdb.SaveExecutionGraph(graph); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save execution graph: %v", err)})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message":        "User Selected their choice!",
				"ProcessID":      UserChoice.ProcessSequence,
				"sample":         sample,
				"Choice":         match,
				"executionGraph": graph,
			})
			return
		}
	}

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
			assignments := make([]Assignment, 0, len(sol.Assignments))
			for _, a := range sol.Assignments {
				assignments = append(assignments, Assignment{
					ProcessName: a.ProcessName,
					ActorID:     a.UserId,
				})
			}
			results.Solutions[i] = ProcessedSolution{
				SolNumber:         i,
				Assignments:       assignments,
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

func customerSearch(requirements ProcessRequirements) (ResultCollection, map[string][]string, error) {
	originMap := make(map[string][]string)
	resultCollectionLog := ResultCollection{}
	matches, err := retrieveProposals("http://localhost:8080", requirements.ProductMatch)
	if err != nil {
		log.Printf("Retrieval from local API did not work: %s", err)
		return resultCollectionLog, originMap, err
	}
	fmt.Println("Matches local", matches)
	for _, match := range matches {
		// Customers are the delivery destination in forward mode → "end" actors.
		res := Result{ContactName: match.CustomerID, Matches: []string{"end"}, Location: match.CustomerLocation.Coordinates}
		resultCollectionLog.Results = append(resultCollectionLog.Results, res)
	}
	if len(resultCollectionLog.Results) == 0 {
		fmt.Println("No results received.")
	}
	for _, item := range resultCollectionLog.Results {
		originMap["end"] = append(originMap["end"], item.ContactName)
	}

	return resultCollectionLog, originMap, nil
}

// Change this function to be able to query the dummy reverse data for DSM (shall extra not provide it)
func providerSearch(requirements ProcessRequirements) (ResultCollection, map[string][]string, error) {
	originMap := make(map[string][]string)
	resultCollectionLog := ResultCollection{}
	batchEfficiencies := make(map[string][]string)

	for batch, path := range HoloFiles {
		batchX, err := getBatch(path)
		if err != nil {
			fmt.Printf("Error reading %s: %v\n", path, err)
			continue
		}

		// Collect efficiency classes for this batch
		classSet := make(map[string]bool)
		for _, solution := range batchX.FeasibleSolutions {
			for _, prod := range solution.OutputProducts {
				if prod.EfficiencyClass != "" {
					classSet[prod.EfficiencyClass] = true
				}
			}
		}

		// Convert to slice
		for class := range classSet {
			batchEfficiencies[batch] = append(batchEfficiencies[batch], class)
		}
	}
	fmt.Println(batchEfficiencies)
	for b_Id, batch := range batchEfficiencies {
		if contains(batch, requirements.ProductMatch) {
			res := Result{ContactName: b_Id, Matches: []string{"origin"}, Location: HoloLocations[b_Id]}
			resultCollectionLog.Results = append(resultCollectionLog.Results, res)
		}
	}
	if len(resultCollectionLog.Results) == 0 {
		fmt.Println("No results received.")
	}
	for _, item := range resultCollectionLog.Results {
		originMap["origin"] = append(originMap["origin"], item.ContactName)
	}

	return resultCollectionLog, originMap, nil
}

func retrieveProposals(baseURL, efficiencyClass string) ([]MotorElement, error) {
	params := url.Values{}
	params.Set("efficiencyClass", efficiencyClass)
	fullURL := fmt.Sprintf("%s/purchase_proposals?%s", baseURL, params.Encode())

	resp, err := http.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server error: %s - %s", resp.Status, string(body))
	}

	var proposals []MotorElement
	if err := json.NewDecoder(resp.Body).Decode(&proposals); err != nil {
		return nil, fmt.Errorf("error decoding JSON: %v", err)
	}

	return proposals, nil
}

func ExtractStepNamesFromRoutes(r OptimizationInput) []string {
	stepNameSet := make(map[string]struct{})
	for _, route := range r.Routes {
		for name := range route {
			stepNameSet[name] = struct{}{}
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

	clientOpt := pb.NewSubmissionServiceClient(conn)
	optimizationResultsChan := make(chan OptimizationResultForSequence, len(r.Routes))
	var wgOptimization sync.WaitGroup

	for routeName, dag := range r.Routes {
		prDAG := DAGToPrerequisites(dag)
		sequence, err := TopologicalSort(dag)
		if err != nil {
			log.Printf("Route %s has an invalid DAG (cycle?): %v", routeName, err)
			continue
		}

		wgOptimization.Add(1)
		go func(routeName string, sequence []string, prDAG map[string][]string) {
			defer wgOptimization.Done()

			optimizationProcesses := make([]string, 0, len(sequence)+2)
			optimizationProcesses = append(optimizationProcesses, "origin")
			optimizationProcesses = append(optimizationProcesses, sequence...)
			optimizationProcesses = append(optimizationProcesses, "end")

			fmt.Println("route:", routeName, "processes ordered:", optimizationProcesses)
			jsonData := exportToJson(optimizationProcesses, costMatrix, distMatrix, augmentWithBoundaryNodes(prDAG))
			fmt.Println("data used for opt:", jsonData)

			res, err := sendOptimize(jsonData, clientOpt)
			if err != nil {
				log.Printf("Failed to send optimization request for route %s: %v", routeName, err)
				optimizationResultsChan <- OptimizationResultForSequence{
					ProcessSequence: sequence,
					DAGPrereqs:      prDAG,
					Error:           fmt.Sprintf("Optimization request failed: %v", err),
				}
				return
			}

			optResultsJSON, err := processOptimizationResponse(res)
			if err != nil {
				log.Printf("Error processing optimization response for route %s: %v", routeName, err)
				optimizationResultsChan <- OptimizationResultForSequence{
					ProcessSequence: sequence,
					DAGPrereqs:      prDAG,
					Error:           fmt.Sprintf("Failed to process optimization results: %v", err),
				}
				return
			}

			optimizationResultsChan <- OptimizationResultForSequence{
				ProcessSequence:  sequence,
				DAGPrereqs:       prDAG,
				OptimizationJSON: optResultsJSON,
			}
		}(routeName, sequence, prDAG)
	}
	wgOptimization.Wait()
	close(optimizationResultsChan)

	for result := range optimizationResultsChan {
		result.ProcessID = uuid.NewString()
		err := SaveOptimizationResult(result.ProcessID, result, OptimizationTTL)
		if err != nil {
			log.Printf("Optimization was not stored in cache")
		}
		finalResponse.Results = append(finalResponse.Results, result)
		resultOpt, err := GetOptimizationResult(result.ProcessID)
		if err != nil {
			log.Printf("Optimization was not saved")
		}
		fmt.Println("Result for this opt run ", *resultOpt)
	}
	return finalResponse, nil

}

func techProviderSearch(AClasses []string, steps []StepEntry, dir *Directory) (responsesTP, error) {
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
	fmt.Println(finalResponse.Responses)
	if len(finalResponse.Responses) == 0 {
		fmt.Println("Error? ")
		return finalResponse, fmt.Errorf("No technology providers matched any processes.: %v")
	}

	return finalResponse, nil
}

// This function is called once per technology provider to retrieve their data
// defaultCostPerClass holds fallback cost_per_hour values used when the AAS API is unreachable.
var defaultCostPerClass = map[string]struct{ cost, co2 float64 }{
	"A": {50.0, 10.0},
	"C": {80.0, 15.0},
	"D": {70.0, 12.0},
}

func buildFallbackResponse(class, name string, loc []float64, steps []StepEntry) techProviderResponse {
	response := techProviderResponse{uClass: class, uid: name, location: loc}
	defaults := defaultCostPerClass[class]

	allowed := make(map[string]bool)
	for _, p := range ActorProcesses[class] {
		allowed[p] = true
	}
	for _, step := range steps {
		if allowed[step.StepName] {
			response.Processes = append(response.Processes, process{
				ProcessID:   fmt.Sprintf("%s-%s", name, step.StepName),
				ProcessName: step.StepName,
				Cost:        defaults.cost,
				Co2:         defaults.co2,
			})
		}
	}
	return response
}

func callTechProvider(class string, steps []StepEntry, address string, name string, loc []float64) (techProviderResponse, error) {
	response := techProviderResponse{uClass: class, uid: name, location: loc}
	providerData, err := fetchAAS(address)
	if err != nil {
		log.Printf("[fallback] AAS unavailable for %s (%s): %v — using dummy data", name, class, err)
		response = buildFallbackResponse(class, name, loc, steps)
		if saveErr := SaveTPResponse("TPRes_"+name, response, OptimizationTTL); saveErr != nil {
			log.Printf("Fallback response not stored in cache: %v", saveErr)
		}
		return response, nil
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
	err = SaveTPResponse("TPRes_"+name, response, OptimizationTTL)
	if err != nil {
		log.Printf("Optimization was not stored in cache")
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

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AAS API returned %d for %s", resp.StatusCode, id)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read AAS response body: %v", err)
	}
	log.Printf("AAS raw response for %s: %s", id, string(body))

	var aas AASResponse
	if err := json.Unmarshal(body, &aas); err != nil {
		return nil, fmt.Errorf("failed to decode AAS response: %v", err)
	}
	if aas.Data.IDShort == "" {
		return nil, fmt.Errorf("AAS response decoded but contains no data for %s", id)
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

//HAD TO REMOVE THE RETURN FALSE IN EACH IF AS AAS NOT WORKING

func matchClassA(aas AASResponse, req Requirements) bool {
	axisHeight := getFloatFromAAS(aas, "axis_height")
	power := getFloatFromAAS(aas, "power")
	KHFT := getBoolFromAAS(aas, "know-how_formalization-tool")
	fmt.Println("to match ", "AH: ", *req.AxisHeight, "Power: ", *req.Power, " With: ", "AH: ", axisHeight, "Power: ", power)
	if req.AxisHeight != nil && axisHeight < *req.AxisHeight {
		fmt.Println("Axis height too high for class A for actor ", aas.Data.IDShort)

	}
	if req.Power != nil && power < *req.Power {
		fmt.Println("Power too high for class A for actor ", aas.Data.IDShort)
	}
	if req.KnowHowFormalizationTool != nil && KHFT != *req.KnowHowFormalizationTool {
		fmt.Println("Know-how-formalization-tool property not matching for class A")
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

	}
	if req.Length != nil && length < float64(*req.Length) {
		fmt.Println("Length too long for class C for actor ", aas.Data.IDShort)

	}
	if req.Diameter != nil && diameter < float64(*req.Diameter) {
		fmt.Println("Diameter too large for class C for actor ", aas.Data.IDShort)

	}
	return true
}

func matchClassD(aas AASResponse, req Requirements) bool {
	axisHeight := getFloatFromAAS(aas, "axis_height")
	fmt.Println("to match ", "AH: ", *req.AxisHeight, " With: ", "AH: ", axisHeight)
	if req.AxisHeight != nil && axisHeight < *req.AxisHeight {
		fmt.Println("Axis height too high for class D for actor ", aas.Data.IDShort)

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
		for name, node := range route {
			if _, exists := uniqueSteps[name]; !exists {
				uniqueSteps[name] = node.Requirements
			}
		}
	}
	result := make([]StepEntry, 0, len(uniqueSteps))
	for name, reqs := range uniqueSteps {
		result = append(result, StepEntry{StepName: name, Requirements: reqs})
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
