package src

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"sort"
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
	PClasses, err := getInterests("./data/interests.json")
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

	processToFetch := GetDistinct(request.Routes)

	resultCollection := ResultCollection{}
	resultCollectionLog := ResultCollection{}
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
		resultCollection.Results = append(resultCollection.Results, <-results)
	}
	resultMap := CreateMap(processToFetch, resultCollection)
	routeIndex := indexBuilder(resultMap)
	for index, route := range request.Routes {

		route_c := []string{}
		taskN = 0
		pathList := pathMaker(route_c, taskN, resultMap, routeIndex, PossibleRoutesI, route)
		PossibleRoutes[index] = pathList
	}
	fmt.Println(PossibleRoutes)
	distances := costCalculator(dir, PossibleRoutes, costs, &allCost, &request)
	var wg sync.WaitGroup
	results_log := make(chan Result)
	for _, PClass := range PClasses.PClasses {
		if PClass.PClass == product_match {
			for _, cost := range distances.Options {
				for _, customer := range PClass.Users {
					wg.Add(1)
					go func(customer Customer, cost OptionCost) {
						defer wg.Done() // Ensure goroutine is marked as done

						distanceMap := make(map[string]float64)
						for _, contact := range dir.Contacts {
							distanceMap[contact.Name] = Haversine(contact.Location, customer.Location)
						}
						fmt.Println(distanceMap)
						// Establish a gRPC connection to the contact's server
						conn, err := grpc.NewClient(customer.Address, grpc.WithTransportCredentials(insecure.NewCredentials()))
						if err != nil {
							return
						}
						defer conn.Close()

						// Create a client
						client := pb.NewSubmissionServiceClient(conn)
						lastItem_user := cost.Option[len(cost.Option)-1]

						response := CheckCustomer(client, cost, product_match, distanceMap[lastItem_user])

						if response.Capability {
							final_logistics := cost.Logistics + distanceMap[lastItem_user]
							results_log <- Result{
								ContactName:    customer.UID,
								Capability:     true,
								Response:       cost.RouteID,
								TotalLogistics: final_logistics,
							}
							return

						}
					}(customer, cost)

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
	}

	// Collect results

	sort.Slice(resultCollectionLog.Results, func(i, j int) bool {
		return resultCollectionLog.Results[i].TotalLogistics < resultCollectionLog.Results[j].TotalLogistics
	})
	fmt.Println(resultCollectionLog.Results)

	c.JSON(http.StatusOK, gin.H{
		"message":         "Data received successfully",
		"customer":        resultCollectionLog.Results[0].ContactName,
		"final_logistics": resultCollectionLog.Results[0].TotalLogistics,
		"selected_route":  resultCollectionLog.Results[0].Response,
		//"Options":         distances,
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

func CheckCustomer(c pb.SubmissionServiceClient, cost OptionCost, product string, logistics float64) *pb.PurchaseResponse {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)

	defer cancel()
	process := &pb.Purchase{
		ProductType: product,
		Logistics:   logistics,
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
