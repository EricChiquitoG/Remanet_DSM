package src

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

var (
	directory    *Directory
	providers    *ProviderData
	providersDir *ProviderDirectory
	locations    *LocAdd
	pclasses     *CustomerInterests
	costs        *CostData
)

// Define the Process struct
// Define the Contact struct
type Contact struct {
	Name      string    `json:"Name"`
	Address   string    `json:"Address"`
	Location  []float64 `json:"Location"`
	Offerings []string  `json:"Offerings"`
}

type LocationsAddresses struct {
	Name     string    `json:"Name"`
	Address  string    `json:"Address"`
	Location []float64 `json:"Location"`
}

// Define the Directory struct
type Directory struct {
	Contacts []Contact `json:"Contacts"`
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

// Structs for providers - Will change once I get input from partners
type ProviderData struct {
	Provider string  `json:"Provider"`
	Motors   []Motor `json:"Motors"`
}

type Motor struct {
	ID                string    `json:"ID"`
	Model             string    `json:"Model"`
	Description       string    `json:"Description"`
	Manufacturer      string    `json:"Manufacturer"`
	SerialNumber      string    `json:"SerialNumber"`
	DateOfManufacture string    `json:"DateOfManufacture"`
	CountryOfOrigin   string    `json:"CountryOfOrigin"`
	TechnicalData     Technical `json:"TechnicalData"`
	Materials         Materials `json:"Materials"`
	Stock             int       `json:"Stock"`
}

type Technical struct {
	Phase            int     `json:"Phase"`
	RatedPower_kW    float64 `json:"RatedPower_kW"`
	EfficiencyClass  string  `json:"EfficiencyClass"`
	AxisHeight_mm    int     `json:"AxisHeight_mm"`
	ShaftDiameter_mm int     `json:"ShaftDiameter_mm"`
	ShaftLength_mm   int     `json:"ShaftLength_mm"`
}

type Materials struct {
	Stator SubMaterial `json:"Stator"`
	Rotor  SubMaterial `json:"Rotor"`
}

type SubMaterial struct {
	Material     string  `json:"Material"`
	MassFraction float64 `json:"MassFraction"`
}

// End of provider structs

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

func getProvider(filename string) (*ProviderData, error) {
	return LoadFromFile[ProviderData](filename)
}

func provDir(filename string) (*ProviderDirectory, error) {
	return LoadFromFile[ProviderDirectory](filename)
}

func MyDir(filename string) (*Directory, error) {
	return LoadFromFile[Directory](filename)
}

func MyLocs(filename string) (*LocAdd, error) {
	return LoadFromFile[LocAdd](filename)
}

func UpdateJson(newContact Contact) error {
	dir, err := MyDir("./data/directory.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	dir.Contacts = append(dir.Contacts, newContact)

	return nil
}
func InitData() error {
	var err error

	directory, err = MyDir("./data/directory.json")
	if err != nil {
		return fmt.Errorf("failed to load directory.json: %v", err)
	}

	providersDir, err = provDir("./data/providers.json")
	if err != nil {
		return fmt.Errorf("failed to load providers.json: %v", err)
	}

	locations, err = MyLocs("./data/locationAdd.json")
	if err != nil {
		return fmt.Errorf("failed to load locationAdd.json: %v", err)
	}

	pclasses, err = getInterests("./data/interests.json")
	if err != nil {
		return fmt.Errorf("failed to load interests.json: %v", err)
	}

	costs, err = Costs("./data/cost.json")
	if err != nil {
		return fmt.Errorf("failed to load cost.json: %v", err)
	}

	return nil
}
