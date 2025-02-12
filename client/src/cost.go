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

type optionCost struct {
	route     []string
	option    []string
	logistics float64
	co2_em    float64
	energy    float64
	cost_eur  float64
}

type AllCost struct {
	options []optionCost
}

// TransportCosts holds predefined values for different transport mechanisms
var TransportEmissions = map[string]float64{
	"Truck": 0.1,   // Example values
	"Train": 0.015, // Lower emissions per km
	"Ship":  0.03,  // Even lower emissions
}

// Creates a map with the locations of each company
func locationMapper(dir *Directory) map[string]distancecalculator.Coord {
	companyLocations := make(map[string]distancecalculator.Coord)

	for _, contact := range dir.Contacts {
		companyLocations[contact.Name] = distancecalculator.Coord{
			Lat:  contact.Location[0],
			Long: contact.Location[1],
		}
	}
	return companyLocations
}

// Calculates the distance between 2 cities
func logistics(locations map[string]distancecalculator.Coord, compA string, compB string) (float64, error) {
	destinations := []distancecalculator.Coord{}
	destinations = append(destinations, locations[compB])

	distancesInKilometers, err := distancecalculator.Kilometers(locations[compA], destinations)
	if err != nil {
		fmt.Println("Error calculating distances:", err)
		return 0, err
	}
	return distancesInKilometers[0], nil
}

func costCalculator(dir *Directory, possibleRoutes map[string][][]string, routes map[string][]string, cost *CostData, allCost *AllCost) *AllCost {
	currentCost := optionCost{}
	var co2 float64
	var energy float64
	var cost_eur float64
	companyLocations := locationMapper(dir)
	for routeID, route := range possibleRoutes {
		//Iterate over the different tasks associated with that route
		for _, task := range routes[routeID] {
			//Iterate over the processes in the cost JSON
			for _, process := range cost.Processes {
				//Match if the process is in the cost json
				if task == process.ProcessID {
					co2 += process.Co2Em
					energy += process.Energy
					cost_eur += process.Cost
				}
			}
		}
		for _, option := range route {

			currentCost.route = routes[routeID]
			currentCost.option = option
			for i, _ := range option {
				if i != 0 {
					if option[i] != option[i-1] {
						distance, err := logistics(companyLocations, option[i], option[i-1])
						if err != nil {
							fmt.Println("Calculating logistics:", err)
							return allCost
						}
						currentCost.logistics += distance * (TransportEmissions["Truck"] * 10)
					}
				}
			}
			currentCost.co2_em = co2
			currentCost.cost_eur = cost_eur
			currentCost.energy = energy
			allCost.options = append(allCost.options, currentCost)
			currentCost = optionCost{}
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
