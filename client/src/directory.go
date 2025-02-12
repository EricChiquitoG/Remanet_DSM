package src

import (
	"encoding/json"
	"fmt"
	"io"

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

// Define the Directory struct
type Directory struct {
	Contacts []Contact `json:"Contacts"`
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

func DataToDir(json_data []byte) (*Directory, error) {
	var directory Directory
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
