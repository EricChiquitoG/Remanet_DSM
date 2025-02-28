package src

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Define the struct to match the JSON structure
type InterestsData struct {
	Interests []Interest `json:"interests"`
}

type Interest struct {
	Product    string   `json:"product"`
	Capacity   int      `json:"capacity"`
	Usage      int      `json:"usage"`
	PhaseN     int      `json:"phase_n"`
	RatedSpeed *float64 `json:"rated_speed"` // Nullable field
	RatedPower float64  `json:"rated_power"`
	AxisHeight int      `json:"axis_height"`
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

func DataToOffering(json_data []byte) (*InterestsData, error) {
	var data InterestsData
	err := json.Unmarshal(json_data, &data)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %v", err)
	}

	return &data, nil
}

func CompanyData(filename string) (*InterestsData, error) {
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

func InitializeData(company string) (*InterestsData, error) {

	file_name := "./data/" + company + ".json"
	offerings, err := CompanyData(file_name)
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	return offerings, nil
}
