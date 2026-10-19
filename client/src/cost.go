package src

import (
	"math"
	"sort"
)

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

func DistanceMatrixConstructor(resultCollection ResultCollection, dir *Directory) (DistanceMatrix map[string]map[string]float64) {
	n := len(resultCollection.Results)
	distanceMatrix := make(map[string]map[string]float64)

	for i := 0; i < n; i++ {
		from := resultCollection.Results[i].ContactName
		distanceMatrix[from] = make(map[string]float64)
		for j := 0; j < n; j++ {
			to := resultCollection.Results[j].ContactName
			if to == "SPIN_REDESIGN" || from == "SPIN_REDESIGN" {
				distanceMatrix[from][to] = 0
				continue
			}
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

func CreateCostMatrixFromResults(rc ResultCollection, techProv *responsesTP) (map[string]map[string]float64, error) {

	// Initialize the cost matrix
	matrix := make(map[string]map[string]float64)
	baseCosts := []string{"origin", "end"}
	for _, res := range rc.Results {
		company := res.ContactName
		matrix[company] = make(map[string]float64)
		if len(res.Matches) > 0 {
			if contains(baseCosts, res.Matches[0]) {
				matrix[res.ContactName][res.Matches[0]] = 1
			}
			// Hardcoded Cost for redesign
			if res.Matches[0] == "Redesign" {
				matrix[res.ContactName][res.Matches[0]] = 100
			}
		}
		for _, tp := range techProv.Responses {
			if tp.uid == company {
				for _, process := range tp.Processes {
					if contains(res.Matches, process.ProcessName) {
						matrix[company][process.ProcessName] = process.Cost
					} else {
						matrix[company][process.ProcessName] = 0
					}
				}
			}
		}
	}
	return matrix, nil
}

// CreateEnergyMatrixFromResults mirrors CreateCostMatrixFromResults, but for
// energy: matrix[actor][processName] = energy consumed by that actor
// performing that process. Unlike cost, boundary/Redesign entries don't need
// a nonzero sentinel — the optimizer's capability gating reads CostMatrix
// only, so a 0 here just means "no energy tracked," never "infeasible."
func CreateEnergyMatrixFromResults(rc ResultCollection, techProv *responsesTP) (map[string]map[string]float64, error) {
	matrix := make(map[string]map[string]float64)
	for _, res := range rc.Results {
		company := res.ContactName
		matrix[company] = make(map[string]float64)
		if len(res.Matches) > 0 && contains([]string{"origin", "end", "Redesign"}, res.Matches[0]) {
			matrix[res.ContactName][res.Matches[0]] = 0
		}
		for _, tp := range techProv.Responses {
			if tp.uid == company {
				for _, process := range tp.Processes {
					if contains(res.Matches, process.ProcessName) {
						matrix[company][process.ProcessName] = process.Energy
					} else {
						matrix[company][process.ProcessName] = 0
					}
				}
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
