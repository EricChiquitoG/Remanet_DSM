package src

import (
	"encoding/json"
	"fmt"
	"os"
)

var (
	directory      *Directory
	pclasses       *CustomerInterests
	costs          *CostData
	ActorProcesses map[string][]string
	ProcessMap     map[string]int
)

type Contact struct {
	Name     string    `json:"Name"`
	Address  string    `json:"Address"`
	Location []float64 `json:"Location"`
}

// Root JSON
type Directory struct {
	Contacts map[string][]Contact `json:"Contacts"`
}

type LocationsAddresses struct {
	Name     string    `json:"Name"`
	Address  string    `json:"Address"`
	Location []float64 `json:"Location"`
}

type LocAdd struct {
	Contacts []LocationsAddresses `json:"Contacts"`
}

type Customer struct {
	UID      string    `json:"uid"`
	Address  string    `json:"Address"`
	Location []float64 `json:"location"`
}

// PClassEntry represents each "PClass" with its list of users
type PClassEntry struct {
	PClass string     `json:"PClass"`
	Users  []Customer `json:"users"`
}

// Structs for interests
type CustomerInterests struct {
	Customers []CustomerInterested `json:"Customers"`
}

type CustomerInterested struct {
	Name     string    `json:"Name"`
	Address  string    `json:"Address"`
	Location []float64 `json:"location"`
	PClasses []string  `json:"Pclasses"`
}

// End of structs for interests

// Structs for provider directory
type ProviderDirectory struct {
	Providers []CustomerInterested `json:"Providers"`
}

//_________ Structs to ingest data from AAS_________

type AASResponse struct {
	Status string  `json:"status"`
	Data   AASData `json:"data"`
}

// Core AAS data
type AASData struct {
	ModelType      ModelType      `json:"modelType"`
	IDShort        string         `json:"idShort"`
	Identification Identification `json:"identification"`
	Submodels      []Submodel     `json:"submodels"`
	Asset          Asset          `json:"asset"`
}

type ModelType struct {
	Name string `json:"name"`
}

type Identification struct {
	IDType string `json:"idType"`
	ID     string `json:"id"`
}

// Each submodel (e.g., CapabilityDescription)
type Submodel struct {
	IDShort          string            `json:"idShort"`
	Identification   Identification    `json:"identification"`
	ModelType        ModelType         `json:"modelType"`
	Kind             string            `json:"kind"`
	SubmodelElements []SubmodelElement `json:"submodelElements"`
}

// Each property or capability inside the submodel
type SubmodelElement struct {
	IDShort    string          `json:"idShort"`
	ModelType  ModelType       `json:"modelType"`
	ValueType  string          `json:"valueType,omitempty"`
	Value      json.RawMessage `json:"value,omitempty"` // flexible: string, number, object
	Qualifiers []Qualifier     `json:"qualifiers,omitempty"`
}

// Qualifier structure (true/false roles like Offered, Required)
type Qualifier struct {
	Type      string `json:"type"`
	ValueType string `json:"valueType"`
	Value     string `json:"value"`
}

// Asset metadata
type Asset struct {
	IDShort        string         `json:"idShort"`
	Identification Identification `json:"identification"`
	Kind           string         `json:"kind"`
	Category       string         `json:"category"`
}

//_______ Ends of AAS structs

//___ Dummy Intrasoft data -  Need to change it to match Holonix input

// Generic type for processes
type Process struct {
	Co2Static float64 `json:"Co2Static"`
	CostEUR   float64 `json:"CostEUR"`
}

// Shaft Manufacturer
type ShaftManufacturer struct {
	ActorType          string    `json:"ActorType"`
	CustomerClass      string    `json:"CustomerClass"`
	Location           []float64 `json:"Location"`
	AvailableProcesses struct {
		Grinding Process `json:"Grinding"`
		Turning  Process `json:"Turning"`
	} `json:"AvailableProcesses"`
	Requirements struct {
		MaximumTolerance int `json:"MaximumTolerance"`
		MaximumLength    int `json:"MaximumLength"`
		MaximumDiameter  int `json:"MaximumDiameter"`
	} `json:"Requirements"`
}

// Disassembler
type Disassembler struct {
	ActorType          string    `json:"ActorType"`
	CustomerClass      string    `json:"CustomerClass"`
	Location           []float64 `json:"Location"`
	AvailableProcesses struct {
		Cleaning      Process `json:"Cleaning"`
		Inspection    Process `json:"Inspection"`
		Disassembly   Process `json:"Disassembly"`
		Reassembly    Process `json:"Reassembly"`
		Certification Process `json:"Certification"`
	} `json:"AvailableProcesses"`
	Requirements struct {
		MaximumPower             int  `json:"MaximumPower"`
		MaximumAxisHeight        int  `json:"MaximumAxisHeight"`
		KnowHowFormalizationTool bool `json:"KnowHowFormalizationTool"`
	} `json:"Requirements"`
}

// Lamination
type Lamination struct {
	ActorType          string    `json:"ActorType"`
	CustomerClass      string    `json:"CustomerClass"`
	Location           []float64 `json:"Location"`
	AvailableProcesses struct {
		Punching Process `json:"Punching"`
	} `json:"AvailableProcesses"`
	Requirements struct {
		MaximumAxisHeight int `json:"MaximumAxisHeight"`
	} `json:"Requirements"`
}

// End of dummy intrasoft data

var Port = os.Getenv("GRPC_PORT")

func ReadJSONFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

func ParseJSON[T any](data []byte) (*T, error) {
	var obj T
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}
	return &obj, nil
}

func LoadFromFile[T any](filename string) (*T, error) {
	data, err := ReadJSONFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %v", filename, err)
	}
	return ParseJSON[T](data)
}

func getInterests(filename string) (*CustomerInterests, error) {
	return LoadFromFile[CustomerInterests](filename)
}

func MyDir(filename string) (*Directory, error) {
	return LoadFromFile[Directory](filename)
}

func MyLocs(filename string) (*LocAdd, error) {
	return LoadFromFile[LocAdd](filename)
}

// Process of AAS data function

func ProcessAAS(filename string) (*AASData, error) {
	return LoadFromFile[AASData](filename)
}

func InitData() error {
	var err error

	directory, err = MyDir("./data/directory.json")
	if err != nil {
		return fmt.Errorf("failed to load directory.json: %v", err)
	}

	pclasses, err = getInterests("./data/interests.json")
	if err != nil {
		return fmt.Errorf("failed to load interests.json: %v", err)
	}

	costs, err = Costs("./data/cost.json")
	if err != nil {
		return fmt.Errorf("failed to load cost.json: %v", err)
	}

	ActorProcesses = map[string][]string{
		"C": {"Grinding", "Turning"},
		"A": {"Cleaning", "Inspection", "Disassembly", "BearingSubstitution", "Reassembly", "Certification"},
		"D": {"Punching"},
	}

	ProcessMap = map[string]int{
		"good": 3,
		"high": 3,
		"mid":  2,
		"low":  1,
	}

	return nil
}
