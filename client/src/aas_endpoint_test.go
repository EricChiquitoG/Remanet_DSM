package src

// Proves the real HTTP endpoint POST /get_options_validate works end to
// end: real router, real handler, real AAS-file-backed actor
// identification, real optimizer — not just the Go functions behind it.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func newValidateRouter() *gin.Engine {
	r := gin.New()
	r.POST("/get_options_validate", ProcessDirectoryValidate)
	return r
}

func TestProcessDirectoryValidate_RejectsNonMotorUseCase(t *testing.T) {
	body := `{
		"use_case": "cutting-tool",
		"process_requirements": {"product_match": "x", "quantity": 1},
		"routes": {"r1": {"Coating": {"prerequisites": [], "requirements": {}}}},
		"starting_point": [1, 2],
		"optimization_mode": "forward"
	}`
	req := httptest.NewRequest(http.MethodPost, "/get_options_validate", bytes.NewReader([]byte(body)))
	w := httptest.NewRecorder()
	newValidateRouter().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400. body: %s", w.Code, w.Body.String())
	}
}

func TestProcessDirectoryValidate_KBSEnvelopeEndToEnd(t *testing.T) {
	requireMongoForKBS(t)
	optimizerAddr := os.Getenv("OPTIMIZER_ADDR")
	if optimizerAddr == "" {
		t.Skip("OPTIMIZER_ADDR not set — skipping live optimizer run")
	}
	if os.Getenv("HEALTH_MONGO_ADDR") == "" {
		os.Setenv("HEALTH_MONGO_ADDR", "localhost:27017")
	}
	os.Setenv("HEALTH_OPTIMIZER_ADDR", optimizerAddr)
	runHealthChecks()
	if ok, msg := IsServiceAvailable(ServiceMongoDB); !ok {
		t.Skipf("MongoDB health check failed: %s", msg)
	}
	if ok, msg := IsServiceAvailable(ServiceOptimizer); !ok {
		t.Skipf("optimizer health check failed: %s", msg)
	}

	kbs, err := LoadKBSOutput(motorKBSFile)
	if err != nil {
		t.Fatalf("LoadKBSOutput: %v", err)
	}
	envelope := KBSOptimizationRequest{
		StartingPoint:    []float64{53.0, -8.0},
		OptimizationMode: "forward",
		KBS:              *kbs,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/get_options_validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newValidateRouter().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200. body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		OptimizationResults struct {
			Results []struct {
				ProcessSequence       []string `json:"processSequence"`
				OptimizationResultsJS struct {
					Solutions []struct {
						SolNumber int `json:"SolNumber"`
					} `json:"Solutions"`
				} `json:"optimizationResultsJson"`
				Error string `json:"error"`
			} `json:"results"`
		} `json:"optimizationResults"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decoding response: %v\nbody: %s", err, w.Body.String())
	}
	if len(resp.OptimizationResults.Results) != 2 {
		t.Fatalf("got %d route results, want 2 (reconditionStrategy + Upgrade). body: %s",
			len(resp.OptimizationResults.Results), w.Body.String())
	}
	for _, route := range resp.OptimizationResults.Results {
		if route.Error != "" {
			t.Errorf("route (sequence len %d) failed: %s", len(route.ProcessSequence), route.Error)
			continue
		}
		if len(route.OptimizationResultsJS.Solutions) == 0 {
			t.Errorf("route (sequence len %d) produced zero solutions", len(route.ProcessSequence))
		}
	}
}
