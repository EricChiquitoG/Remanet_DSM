package src

import "fmt"

type ProcessData struct {
	Company      string
	Process      string
	ProductTypes []string
}

type AvailableProcesses struct {
	Activities []ProcessData
}

var DataICA = ProcessData{
	Company:      "ICA",
	Process:      "Cleaning",
	ProductTypes: []string{"SawbladesX", "SawbladesY", "SawBladesZ"},
}

var DataCoop = ProcessData{
	Company:      "Coop",
	Process:      "Inspection",
	ProductTypes: []string{"SawbladesX", "SawbladesY", "SawBladesZ"},
}

var DataWillys = ProcessData{
	Company:      "Willys",
	Process:      "Resharpen",
	ProductTypes: []string{"SawbladesX", "SawbladesY"},
}

var ExampleData = AvailableProcesses{
	Activities: []ProcessData{DataCoop, DataICA, DataWillys},
}

type Avilability struct {
	Process           string
	ProductType       string
	economic_operator string
}

type Processes struct {
	P []Avilability
}

var ProcessExample1 = Avilability{
	Process:     "Dump",
	ProductType: "SawbladesX",
}
var ProcessExample2 = Avilability{
	Process:     "Inspection",
	ProductType: "SawbladesY",
}

var AvailabilityExample = Processes{
	P: []Avilability{ProcessExample1, ProcessExample2},
}

func InitializeData(company string) (*ProcessData, error) {
	for _, PData := range ExampleData.Activities {
		if PData.Company == company {
			return &PData, nil
		}
	}
	return nil, fmt.Errorf("company '%s' not found", company)
}
