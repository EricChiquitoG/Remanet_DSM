package src

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/daveroberts0321/distancecalculator"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

type Cost struct {
	ProcessID   string  `json:"ProcessID"`
	ProcessName string  `json:"ProcessName"`
	Co2Em       float64 `json:"Co2_em"`
	Energy      float64 `json:"Energy"`
	Cost        float64 `json:"Cost"`
}

// ProcessData holds a list of processes
type CostData struct {
	Processes []Cost `json:"Processes"`
}

type OptionCost struct {
	RouteID   string   `json:"route_id"`
	Route     []string `json:"route"`
	Option    []string `json:"option"`
	Logistics float64  `json:"logistics"`
	CO2Em     float64  `json:"co2_em"`
	Energy    float64  `json:"energy"`
	CostEUR   float64  `json:"cost_eur"`
}

type AllCost struct {
	Options []OptionCost `json:"options"`
}

type Node struct {
	ID      int64
	Company string
	Step    int    // step associated with the process, same company can be in different or all steps
	Process string //?
}

// TransportCosts holds predefined values for different transport mechanisms

var TransportEmissions = map[string]float64{
	"Truck": 0.1,
	"Train": 0.015,
	"Ship":  0.03,
}

// Creates a map with the locations of each company
func locationMapper(dir *Directory, r *RouteRequest) map[string]distancecalculator.Coord {
	companyLocations := make(map[string]distancecalculator.Coord)

	for _, contact := range dir.Contacts {
		companyLocations[contact.Name] = distancecalculator.Coord{
			Lat:  contact.Location[0],
			Long: contact.Location[1],
		}
	}
	companyLocations["origin"] = distancecalculator.Coord{
		Lat:  r.StartingPoint[0],
		Long: r.StartingPoint[1],
	}
	return companyLocations
}

// Calculates the distance between two companies
func logistics(locations map[string]distancecalculator.Coord, compA, compB string) (float64, error) {
	destinations := []distancecalculator.Coord{locations[compB]}
	distances, err := distancecalculator.Kilometers(locations[compA], destinations)
	if err != nil {
		fmt.Println("Error calculating distances:", err)
		return 0, err
	}
	return distances[0], nil
}

func costCalculator(dir *Directory, possibleRoutes map[string][][]string, cost *CostData, allCost *AllCost, request *RouteRequest) *AllCost {
	var co2, energy, cost_eur float64
	companyLocations := locationMapper(dir, request)
	fmt.Println(companyLocations)
	for routeID, route := range possibleRoutes {
		co2, energy, cost_eur = 0, 0, 0
		//Iterate over the different tasks associated with that route
		for _, task := range request.Routes[routeID] {
			//Iterate over the processes in the cost JSON
			for _, process := range cost.Processes {
				//Match if the process is in the cost json
				if task == process.ProcessID {
					co2 += process.Co2Em
					energy += process.Energy
					cost_eur += process.Cost
					break
				}
			}
		}
		for _, option := range route {
			currentCost := OptionCost{
				RouteID: routeID,
				Route:   request.Routes[routeID],
				Option:  option,
				CO2Em:   co2,
				Energy:  energy,
				CostEUR: cost_eur,
			}
			for i := range option {
				if i != 0 {
					if option[i] != option[i-1] {
						distance, err := logistics(companyLocations, option[i], option[i-1])
						if err != nil {
							fmt.Println("Calculating logistics:", err)
							return allCost
						}
						currentCost.Logistics += distance * (TransportEmissions["Truck"] * 10)
					}
				} else {
					distance, err := logistics(companyLocations, option[i], "origin")
					if err != nil {
						fmt.Println("Calculating logistics:", err)
						return allCost
					}
					currentCost.Logistics += distance * (TransportEmissions["Truck"] * 10)
				}
			}
			allCost.Options = append(allCost.Options, currentCost)

		}

	}
	return Sort_Options(allCost)
}
func DatatoCost(json_data []byte) (*CostData, error) {
	var cost CostData
	err := json.Unmarshal(json_data, &cost)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return &cost, nil
}

func Sort_Options(allcost *AllCost) *AllCost {
	sort.Slice(allcost.Options, func(i, j int) bool {
		if allcost.Options[i].CostEUR != allcost.Options[j].CostEUR {
			return allcost.Options[i].CostEUR < allcost.Options[j].CostEUR // Primary: Lower cost first
		}
		if allcost.Options[i].Logistics != allcost.Options[j].Logistics {
			return allcost.Options[i].Logistics < allcost.Options[j].Logistics // Secondary: Lower logistics cost
		}
		if allcost.Options[i].CO2Em != allcost.Options[j].CO2Em {
			return allcost.Options[i].CO2Em < allcost.Options[j].CO2Em // Tertiary: Lower CO₂ emissions
		}
		return allcost.Options[i].Energy < allcost.Options[j].Energy // Finally: Lower energy consumption
	})
	return allcost
}

func Costs(filename string) (*CostData, error) {
	jsonbytes, err := ReadJSONFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	dir, err := DatatoCost(jsonbytes)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	return dir, err
}

func DistanceMatrixConstructor(LocAddresses *LocAdd) (DistanceMatrix map[string]map[string]float64) {
	n := len(LocAddresses.Contacts)
	distanceMatrix := make(map[string]map[string]float64)
	fmt.Println(LocAddresses.Contacts)
	for i := 0; i < n; i++ {
		from := LocAddresses.Contacts[i].Name
		distanceMatrix[from] = make(map[string]float64)
		for j := 0; j < n; j++ {
			to := LocAddresses.Contacts[j].Name
			if i == j {
				distanceMatrix[from][to] = 0
				continue
			}
			p1 := []float64{LocAddresses.Contacts[i].Location[0], LocAddresses.Contacts[i].Location[1]}
			p2 := []float64{LocAddresses.Contacts[j].Location[0], LocAddresses.Contacts[j].Location[1]}
			d := Haversine(p1, p2)
			distanceMatrix[from][to] = d
		}
	}
	return distanceMatrix
}

// haversine calculates the distance between two points (lat1, lon1) and (lat2, lon2) in kilometers
func Haversine(Point1, Point2 []float64) float64 {
	const R = 6371 // Earth radius in km

	// Convert degrees to radians
	lat1Rad := Point1[0] * (math.Pi / 180)
	lon1Rad := Point1[1] * (math.Pi / 180)
	lat2Rad := Point2[0] * (math.Pi / 180)
	lon2Rad := Point2[1] * (math.Pi / 180)

	// Differences
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	// Haversine formula
	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Distance in km
	return R * c
}

func layerMap(dir *Directory, capMap map[string][]string, processList []string, originNode Node, endNode Node) (map[int][]Node, []string) {
	// map[step][]Node
	layers := make(map[int][]Node)
	processList = append([]string{"P0"}, processList...) // prepend
	processList = append(processList, "PN")
	idCounter := int64(0)
	for stepIdx, process := range processList { // e.g., ["P1", "P2", "P3"]
		if stepIdx == 0 {
			layers[stepIdx] = append(layers[stepIdx], originNode)
			idCounter++
		}

		for _, company := range dir.Contacts {

			if hasProcess(process, company.Name, capMap) {
				node := Node{
					ID:      idCounter,
					Company: company.Name,
					Step:    stepIdx,
					Process: process,
				}
				layers[stepIdx] = append(layers[stepIdx], node)
				idCounter++
			}
		}
		//fmt.Println(stepIdx, len(processList))
		if stepIdx == len(processList)-1 {
			layers[stepIdx] = append(layers[stepIdx], endNode)
			return layers, processList
		}
	}
	return layers, processList
}
func hasProcess(process string, company string, klMap map[string][]string) bool {
	if companies, ok := klMap[process]; ok {
		for _, c := range companies {
			if c == company {
				return true
			}
		}
	}
	return false
}

func graphConstructor(layerMap map[int][]Node, costMatrix map[string]map[string]float64, routeSteps []string) graph.WeightedDirected {
	g := simple.NewWeightedDirectedGraph(0, 0)

	for _, nodes := range layerMap {
		for _, node := range nodes {
			g.AddNode(simple.Node(node.ID))
		}
	}

	// Assume distanceMatrix[companyA][companyB] is precomputed
	for i := 0; i < len(routeSteps)-1; i++ {
		for _, from := range layerMap[i] {
			for _, to := range layerMap[i+1] {
				if from.Company != to.Company {
					cost := costMatrix[from.Company][to.Company]
					g.SetWeightedEdge(g.NewWeightedEdge(
						simple.Node(from.ID),
						simple.Node(to.ID),
						cost,
					))
				} else {
					// same company does two steps, no transport needed
					g.SetWeightedEdge(g.NewWeightedEdge(
						simple.Node(from.ID),
						simple.Node(to.ID),
						0,
					))
				}
			}
		}
	}
	return g
}

func CreateCostMatrixFromResults(rc ResultCollection, costs *CostData) (map[string]map[string]float64, error) {

	// Initialize the cost matrix
	matrix := make(map[string]map[string]float64)
	// Create a lookup for ProcessID -> Cost
	costLookup := make(map[string]float64)
	for _, p := range costs.Processes {
		costLookup[p.ProcessID] = p.Cost
	}
	for _, res := range rc.Results {
		company := res.ContactName
		matrix[company] = make(map[string]float64)

		for _, p := range costs.Processes {
			if contains(res.Matches, p.ProcessID) {
				matrix[company][p.ProcessID] = costLookup[p.ProcessID]
			} else {
				matrix[company][p.ProcessID] = 0
			}
		}
	}

	return matrix, nil
}

func contains(list []string, val string) bool {
	for _, v := range list {
		if v == val {
			return true
		}
	}
	return false
}
