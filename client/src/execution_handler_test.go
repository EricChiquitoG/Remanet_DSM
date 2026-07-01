package src

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// newRouter wires the two handlers under test to a fresh Gin engine.
func newTestRouter() *gin.Engine {
	r := gin.New()
	r.GET("/execution_graphs", GetAllExecutionGraphsHandler)
	r.POST("/advance_step", AdvanceStepHandler)
	return r
}

// sampleGraph returns a predictable ExecutionGraph for assertions.
func sampleGraph(id string) execdb.ExecutionGraph {
	return execdb.ExecutionGraph{
		GraphID:         id,
		ProcessSequence: []string{"Cleaning", "Reassembly"},
		CurrentSteps:    []string{"Cleaning"},
		Nodes: map[string]execdb.ProcessNode{
			"Cleaning": {
				ProcessName: "Cleaning",
				ActorID:     "ActorA",
				Cost:        50,
				Co2:         5,
				Next:        []string{"Reassembly"},
				Status:      "in_progress",
			},
			"Reassembly": {
				ProcessName: "Reassembly",
				ActorID:     "ActorA",
				Cost:        80,
				Co2:         8,
				Next:        []string{},
				Status:      "pending",
			},
		},
		TotalTransportCost:     100,
		TotalManufacturingCost: 130,
	}
}

// ---- GET /execution_graphs -------------------------------------------------

func TestGetAllExecutionGraphs_Empty(t *testing.T) {
	getAllExecutionGraphsFn = func() ([]execdb.ExecutionGraph, error) {
		return []execdb.ExecutionGraph{}, nil
	}
	defer func() { getAllExecutionGraphsFn = execdb.GetAllExecutionGraphs }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/execution_graphs", nil)
	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body []execdb.ExecutionGraph
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(body) != 0 {
		t.Errorf("expected empty array, got %d graphs", len(body))
	}
}

func TestGetAllExecutionGraphs_ReturnsAll(t *testing.T) {
	graphs := []execdb.ExecutionGraph{
		sampleGraph("graph-1"),
		sampleGraph("graph-2"),
	}
	getAllExecutionGraphsFn = func() ([]execdb.ExecutionGraph, error) {
		return graphs, nil
	}
	defer func() { getAllExecutionGraphsFn = execdb.GetAllExecutionGraphs }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/execution_graphs", nil)
	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body []execdb.ExecutionGraph
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("expected 2 graphs, got %d", len(body))
	}
	if body[0].GraphID != "graph-1" || body[1].GraphID != "graph-2" {
		t.Errorf("unexpected graph IDs: %s, %s", body[0].GraphID, body[1].GraphID)
	}
}

func TestGetAllExecutionGraphs_GraphFields(t *testing.T) {
	getAllExecutionGraphsFn = func() ([]execdb.ExecutionGraph, error) {
		return []execdb.ExecutionGraph{sampleGraph("g1")}, nil
	}
	defer func() { getAllExecutionGraphsFn = execdb.GetAllExecutionGraphs }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/execution_graphs", nil)
	newTestRouter().ServeHTTP(w, req)

	var body []execdb.ExecutionGraph
	json.NewDecoder(w.Body).Decode(&body)
	g := body[0]

	if g.TotalTransportCost != 100 {
		t.Errorf("TotalTransportCost: expected 100, got %f", g.TotalTransportCost)
	}
	if g.TotalManufacturingCost != 130 {
		t.Errorf("TotalManufacturingCost: expected 130, got %f", g.TotalManufacturingCost)
	}
	if len(g.ProcessSequence) != 2 {
		t.Errorf("ProcessSequence length: expected 2, got %d", len(g.ProcessSequence))
	}
	if _, ok := g.Nodes["Cleaning"]; !ok {
		t.Error("Nodes should contain Cleaning")
	}
}

func TestGetAllExecutionGraphs_DBError(t *testing.T) {
	getAllExecutionGraphsFn = func() ([]execdb.ExecutionGraph, error) {
		return nil, errors.New("connection refused")
	}
	defer func() { getAllExecutionGraphsFn = execdb.GetAllExecutionGraphs }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/execution_graphs", nil)
	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

// ---- POST /advance_step ----------------------------------------------------

func advanceBody(graphID string) *bytes.Buffer {
	b, _ := json.Marshal(map[string]string{"graphId": graphID})
	return bytes.NewBuffer(b)
}

func TestAdvanceStep_Success(t *testing.T) {
	advanced := sampleGraph("g1")
	advanced.CurrentSteps = []string{"Reassembly"}
	advanced.Nodes["Cleaning"] = execdb.ProcessNode{Status: "completed", Next: []string{"Reassembly"}}
	advanced.Nodes["Reassembly"] = execdb.ProcessNode{Status: "in_progress"}

	advanceStepFn = func(id string) (*execdb.ExecutionGraph, error) {
		if id != "g1" {
			t.Errorf("unexpected graphId: %s", id)
		}
		return &advanced, nil
	}
	defer func() { advanceStepFn = execdb.AdvanceStep }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/advance_step", advanceBody("g1"))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	var body execdb.ExecutionGraph
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(body.CurrentSteps) != 1 || body.CurrentSteps[0] != "Reassembly" {
		t.Errorf("expected CurrentSteps=[Reassembly], got %v", body.CurrentSteps)
	}
}

func TestAdvanceStep_MissingGraphID(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/advance_step", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdvanceStep_MalformedJSON(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/advance_step", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAdvanceStep_GraphNotFound(t *testing.T) {
	advanceStepFn = func(id string) (*execdb.ExecutionGraph, error) {
		return nil, fmt.Errorf("execution graph %s not found", id)
	}
	defer func() { advanceStepFn = execdb.AdvanceStep }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/advance_step", advanceBody("missing-id"))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] == "" {
		t.Error("expected non-empty error message")
	}
}

func TestAdvanceStep_CompletedGraph(t *testing.T) {
	// When all steps are done, CurrentSteps should be empty.
	done := sampleGraph("g-done")
	done.CurrentSteps = []string{}

	advanceStepFn = func(id string) (*execdb.ExecutionGraph, error) {
		return &done, nil
	}
	defer func() { advanceStepFn = execdb.AdvanceStep }()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/advance_step", advanceBody("g-done"))
	req.Header.Set("Content-Type", "application/json")
	newTestRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body execdb.ExecutionGraph
	json.NewDecoder(w.Body).Decode(&body)
	if len(body.CurrentSteps) != 0 {
		t.Errorf("expected empty CurrentSteps for completed graph, got %v", body.CurrentSteps)
	}
}
