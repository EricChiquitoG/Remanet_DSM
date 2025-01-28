package src

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Product struct {
	Product_name string `json:"Product_name"`
	Stock        int    `json:"Stock"`
}

// Define the Contact struct
type Offering struct {
	ProcessID   string    `json:"ProcessID"`
	ProcessName string    `json:"ProcessName"`
	Products    []Product `json:"Products"`
}

// Define the Directory struct
type Offerings struct {
	Offerings []Offering `json:"Offerings"`
}

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

func DataToOffering(json_data []byte) (*Offerings, error) {
	var offerings Offerings
	err := json.Unmarshal(json_data, &offerings)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return &offerings, nil
}

func CompanyData(filename string) (*Offerings, error) {
	jsonbytes, err := ReadJSONFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	off, err := DataToOffering(jsonbytes)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	return off, nil
}

func InitializeData(company string) (*Offerings, error) {

	file_name := "./data/" + company + ".json"
	offerings, err := CompanyData(file_name)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	return offerings, nil
}
