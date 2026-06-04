package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type GeoPoint struct {
	Type        string    `json:"type"`
	Coordinates []float64 `json:"coordinates"`
}

type RequiredParameters struct {
	RatedPower      float64 `json:"ratedPower"`
	AxisHeight      int     `json:"axisHeight"`
	EfficiencyClass string  `json:"efficiencyClass"`
	PhaseN          *int    `json:"phaseN,omitempty"`
}

type ProposalCreate struct {
	ProposalID         string             `json:"proposalId"`
	CustomerID         string             `json:"customerId"`
	CustomerLocation   GeoPoint           `json:"customerLocation"`
	Cardinality        int                `json:"cardinality"`
	ProductType        string             `json:"productType"`
	RequiredParameters RequiredParameters `json:"requiredParameters"`
}

// parseLocalization parses "[lon, lat]" string into a GeoJSON Point.
func parseLocalization(s string) (GeoPoint, error) {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "[]")
	parts := strings.SplitN(s, ",", 2)
	if len(parts) != 2 {
		return GeoPoint{}, fmt.Errorf("unexpected localization format: %q", s)
	}
	lon, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return GeoPoint{}, err
	}
	lat, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return GeoPoint{}, err
	}
	return GeoPoint{Type: "Point", Coordinates: []float64{lon, lat}}, nil
}

func rowToProposal(record []string, header map[string]int) (ProposalCreate, error) {
	get := func(col string) string { return strings.TrimSpace(record[header[col]]) }

	customerID := get("id_customer")

	loc, err := parseLocalization(get("localization"))
	if err != nil {
		return ProposalCreate{}, fmt.Errorf("localization: %v", err)
	}

	motorN, err := strconv.Atoi(get("motor_n"))
	if err != nil {
		return ProposalCreate{}, fmt.Errorf("motor_n: %v", err)
	}

	phaseN, err := strconv.Atoi(get("phase_n"))
	if err != nil {
		return ProposalCreate{}, fmt.Errorf("phase_n: %v", err)
	}

	ratedPower, err := strconv.ParseFloat(get("rated_power"), 64)
	if err != nil {
		return ProposalCreate{}, fmt.Errorf("rated_power: %v", err)
	}

	axisHeight, err := strconv.Atoi(get("axis_height"))
	if err != nil {
		return ProposalCreate{}, fmt.Errorf("axis_height: %v", err)
	}

	return ProposalCreate{
		ProposalID:       "proposal-" + customerID,
		CustomerID:       customerID,
		CustomerLocation: loc,
		Cardinality:      motorN,
		ProductType:      "electric_motor",
		RequiredParameters: RequiredParameters{
			RatedPower:      ratedPower,
			AxisHeight:      axisHeight,
			EfficiencyClass: get("efficiency_class"),
			PhaseN:          &phaseN,
		},
	}, nil
}

func postProposal(apiURL string, p ProposalCreate) (int, string, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return 0, "", err
	}
	resp, err := http.Post(apiURL+"/new_proposal", "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	return resp.StatusCode, buf.String(), nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run csv_ingestor.go <csv_file> [api_url]")
		fmt.Println("Example: go run csv_ingestor.go dataset.csv http://localhost:8080")
		os.Exit(1)
	}

	csvPath := os.Args[1]
	apiURL := "http://localhost:8080"
	if len(os.Args) > 2 {
		apiURL = os.Args[2]
	}

	f, err := os.Open(csvPath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	headers, err := reader.Read()
	if err != nil {
		fmt.Printf("Error reading header: %v\n", err)
		os.Exit(1)
	}

	headerIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		headerIdx[strings.TrimSpace(h)] = i
	}

	fmt.Printf("API: %s\nFile: %s\n\n", apiURL, csvPath)

	success, failed := 0, 0
	for lineNum := 2; ; lineNum++ {
		record, err := reader.Read()
		if err != nil {
			break
		}

		proposal, err := rowToProposal(record, headerIdx)
		if err != nil {
			fmt.Printf("[line %d] parse error: %v\n", lineNum, err)
			failed++
			continue
		}

		status, body, err := postProposal(apiURL, proposal)
		if err != nil {
			fmt.Printf("[line %d] request error: %v\n", lineNum, err)
			failed++
			continue
		}

		if status == 201 || status == 200 {
			fmt.Printf("[line %d] OK  customer=%s\n", lineNum, proposal.CustomerID)
			success++
		} else {
			fmt.Printf("[line %d] FAIL status=%d customer=%s body=%s\n", lineNum, status, proposal.CustomerID, body)
			failed++
		}
	}

	fmt.Printf("\nDone: %d inserted, %d failed\n", success, failed)
}
