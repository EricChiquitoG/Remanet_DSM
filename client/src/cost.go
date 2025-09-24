package src

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
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
	Step    int
	Process string
}

var TransportEmissions = map[string]float64{
	"Truck": 0.1,
	"Train": 0.015,
	"Ship":  0.03,
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

func DistanceMatrixConstructor(resultCollection ResultCollection) (DistanceMatrix map[string]map[string]float64) {
	n := len(resultCollection.Results)
	distanceMatrix := make(map[string]map[string]float64)

	for i := 0; i < n; i++ {
		from := resultCollection.Results[i].ContactName
		distanceMatrix[from] = make(map[string]float64)
		for j := 0; j < n; j++ {
			to := resultCollection.Results[j].ContactName
			if i == j {
				distanceMatrix[from][to] = 0
				continue
			}
			p1 := []float64{resultCollection.Results[i].Location[0], resultCollection.Results[i].Location[1]}
			p2 := []float64{resultCollection.Results[j].Location[0], resultCollection.Results[j].Location[1]}
			d := Haversine(p1, p2)
			distanceMatrix[from][to] = d
		}
	}
	return distanceMatrix
}

// haversine function from chatgpt
func Haversine(Point1, Point2 []float64) float64 {
	const R = 6371

	lat1Rad := Point1[0] * (math.Pi / 180)
	lon1Rad := Point1[1] * (math.Pi / 180)
	lat2Rad := Point2[0] * (math.Pi / 180)
	lon2Rad := Point2[1] * (math.Pi / 180)

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func CreateCostMatrixFromResults(rc ResultCollection, costs *CostData) (map[string]map[string]float64, error) {

	// Initialize the cost matrix
	matrix := make(map[string]map[string]float64)
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
