package src

// Tests that the actual HTTP endpoint (not just the Go functions behind
// it) now accepts a real Holonix KBS batch output, via the
// {"starting_point", "optimization_mode", "kbs": {...}} envelope
// BindOptimizationInput understands, alongside the pre-existing
// hand-built OptimizationInput body it already accepted.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newOptimizeRouter() *gin.Engine {
	r := gin.New()
	r.POST("/get_options", ProcessDirectory)
	return r
}

func newGinContextForBody(t *testing.T, body string) *gin.Context {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/get_options", strings.NewReader(body))
	return c
}

func TestBindOptimizationInput_OldShapeUnchanged(t *testing.T) {
	body := `{
		"use_case": "electric-motor",
		"process_requirements": {"product_match": "IE4", "quantity": 5},
		"routes": {"r1": {"Disassembly": {"prerequisites": [], "requirements": {}}}},
		"starting_point": [1.5, 2.5],
		"optimization_mode": "forward"
	}`
	input, err := BindOptimizationInput(newGinContextForBody(t, body))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if input.UseCase != "electric-motor" {
		t.Errorf("UseCase = %q, want electric-motor", input.UseCase)
	}
	if len(input.Routes["r1"]) != 1 {
		t.Errorf("expected route r1 to have 1 step, got %d", len(input.Routes["r1"]))
	}
	if input.StartingPoint[0] != 1.5 || input.StartingPoint[1] != 2.5 {
		t.Errorf("StartingPoint = %v, want [1.5 2.5]", input.StartingPoint)
	}
}

func TestBindOptimizationInput_InvalidJSON(t *testing.T) {
	_, err := BindOptimizationInput(newGinContextForBody(t, "{not valid json"))
	if err == nil {
		t.Error("expected an error for invalid JSON, got nil")
	}
}

func TestBindOptimizationInput_KBSEnvelope_UnknownProductType(t *testing.T) {
	requireMongoForKBS(t)

	kbs, err := LoadKBSOutput(clutchKBSFile)
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

	_, err = BindOptimizationInput(newGinContextForBody(t, string(body)))
	if err == nil {
		t.Error("expected an error for productType \"Clutch\" (no use case declares it), got nil")
	}
}

func TestBindOptimizationInput_KBSEnvelope_RealElectricMotor(t *testing.T) {
	requireMongoForKBS(t)

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

	input, err := BindOptimizationInput(newGinContextForBody(t, string(body)))
	if err != nil {
		t.Fatalf("BindOptimizationInput: %v", err)
	}
	if input.UseCase != "electric-motor" {
		t.Errorf("UseCase = %q, want electric-motor", input.UseCase)
	}
	if _, ok := input.Routes["reconditionStrategy"]; !ok {
		t.Error("missing reconditionStrategy route")
	}
	if _, ok := input.Routes["Upgrade"]; !ok {
		t.Error("missing Upgrade route")
	}
	if input.OptimizationMode != "forward" {
		t.Errorf("OptimizationMode = %q, want forward", input.OptimizationMode)
	}
}

// TestProcessDirectory_KBSEnvelopeEndToEnd posts a real KBS batch output
// straight to POST /get_options (through the real router, the real
// handler, the real optimizer) and checks it comes back with actual
// solutions — proving the live HTTP endpoint accepts this shape, not just
// the Go functions behind it.
func TestProcessDirectory_KBSEnvelopeEndToEnd(t *testing.T) {
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

	testUseCaseID := "kbs-endpoint-motor"
	sentinelProductType := "Motor-endpoint-sentinel"
	uc := electricMotorUseCaseForKBS()
	uc.ID = testUseCaseID
	uc.KBSProductType = sentinelProductType
	if err := execdb.UpsertUseCase(uc); err != nil {
		t.Fatalf("seeding use case: %v", err)
	}
	providers, capabilities := loadClusteredProviders(t, testUseCaseID)
	for _, p := range providers {
		if err := execdb.UpsertProvider(p); err != nil {
			t.Fatalf("seeding provider %s: %v", p.ID, err)
		}
	}
	for _, c := range capabilities {
		if err := execdb.UpsertCapability(c); err != nil {
			t.Fatalf("seeding capability %s: %v", c.ID, err)
		}
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		execdb.UseCaseCollection.DeleteOne(ctx, bson.M{"_id": testUseCaseID})
		execdb.ProviderCollection.DeleteMany(ctx, bson.M{"useCase": testUseCaseID})
		execdb.CapabilityCollection.DeleteMany(ctx, bson.M{"useCase": testUseCaseID})
	}()

	kbs, err := LoadKBSOutput(motorKBSFile)
	if err != nil {
		t.Fatalf("LoadKBSOutput: %v", err)
	}
	kbs.ProductType = sentinelProductType

	envelope := KBSOptimizationRequest{
		StartingPoint:    []float64{53.0, -8.0},
		OptimizationMode: "forward",
		KBS:              *kbs,
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/get_options", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newOptimizeRouter().ServeHTTP(w, req)

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
