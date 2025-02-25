package src

import (
	"encoding/json"
	"fmt"

	"github.com/daveroberts0321/distancecalculator"
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

// TransportCosts holds predefined values for different transport mechanisms
var TransportEmissions = map[string]float64{
	"Truck": 0.1,   // Example values
	"Train": 0.015, // Lower emissions per km
	"Ship":  0.03,  // Even lower emissions
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
	return allCost
}
func DatatoCost(json_data []byte) (*CostData, error) {
	var cost CostData
	err := json.Unmarshal(json_data, &cost)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return &cost, nil
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
