package src

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
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

// Root structure that holds all PClass entries
type PClasses struct {
	PClasses []PClassEntry `json:"PClasses"`
}

var Port = os.Getenv("GRPC_PORT")

func ReadJSONFile(filename string) ([]byte, error) {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Read the file contents
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}
	return bytes, nil
}

func getInterestBytes(json_data []byte) (*PClasses, error) {
	var products PClasses
	err := json.Unmarshal(json_data, &products)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return &products, nil
}

func getInterests(filename string) (*PClasses, error) {
	jsonbytes, err := ReadJSONFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	dir, err := getInterestBytes(jsonbytes)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	return dir, err
}

func DataToDir(json_data []byte) (*Directory, error) {
	var directory Directory
	err := json.Unmarshal(json_data, &directory)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return &directory, nil
}
func DataToLoc(json_data []byte) (*LocAdd, error) {
	var directory LocAdd
	err := json.Unmarshal(json_data, &directory)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return &directory, nil
}

func MyDir(filename string) (*Directory, error) {
	jsonbytes, err := ReadJSONFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	dir, err := DataToDir(jsonbytes)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	return dir, err
}

func MyLocs(filename string) (*LocAdd, error) {
	jsonbytes, err := ReadJSONFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	dir, err := DataToLoc(jsonbytes)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	fmt.Println(dir)
	return dir, err
}

func UpdateJson(newContact Contact) error {
	dir, err := MyDir("./data/directory.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	dir.Contacts = append(dir.Contacts, newContact)
	// Step 4: Marshal back to JSON
	//updatedJSON, err := json.MarshalIndent(dir, "", "  ")
	//if err != nil {
	//	return fmt.Errorf("error marshaling updated data: %v", err)
	//}

	// Step 5: Write back to the file
	//if err := os.WriteFile("./data/directory.json", updatedJSON, 0644); err != nil {
	//	return fmt.Errorf("error writing updated JSON to file: %v", err)
	//}
	return nil
}
