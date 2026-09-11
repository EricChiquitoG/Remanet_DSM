package src

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	ServiceMongoDB    = "mongodb"
	ServiceAASAPI     = "aas_api"
	ServiceOptimizer  = "optimizer"
	healthCheckPeriod = 10 * time.Minute

	defaultAASBaseURL = "https://ds.platform.remanet.extrared.it/ds4/retrieve_by_id"
)

// AASBaseURL returns the AAS API base URL from the AAS_BASE_URL env var,
// falling back to the platform default.
func AASBaseURL() string {
	return getEnvOrDefault("AAS_BASE_URL", defaultAASBaseURL)
}

type ServiceStatus struct {
	Available   bool
	LastChecked time.Time
	Error       string
}

var (
	healthMu sync.RWMutex
	health   = map[string]ServiceStatus{
		ServiceMongoDB:   {},
		ServiceAASAPI:    {},
		ServiceOptimizer: {},
	}
)

// IsServiceAvailable returns whether the named service passed its last health check.
func IsServiceAvailable(service string) (bool, string) {
	healthMu.RLock()
	defer healthMu.RUnlock()
	s, ok := health[service]
	if !ok {
		return false, "unknown service"
	}
	return s.Available, s.Error
}

func checkHTTP(rawURL string) ServiceStatus {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return ServiceStatus{Available: false, LastChecked: time.Now(), Error: err.Error()}
	}
	resp.Body.Close()
	// Any HTTP response (including 401/403) means the server is reachable.
	return ServiceStatus{Available: true, LastChecked: time.Now()}
}

func checkTCP(addr string) ServiceStatus {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return ServiceStatus{Available: false, LastChecked: time.Now(), Error: fmt.Sprintf("unreachable: %v", err)}
	}
	conn.Close()
	return ServiceStatus{Available: true, LastChecked: time.Now()}
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func runHealthChecks() {
	mongoAddr := getEnvOrDefault("HEALTH_MONGO_ADDR", "db:27017")
	optimizerAddr := getEnvOrDefault("HEALTH_OPTIMIZER_ADDR", "optimizer:50060")
	results := map[string]ServiceStatus{
		ServiceMongoDB:   checkTCP(mongoAddr),
		ServiceAASAPI:    checkHTTP(AASBaseURL()),
		ServiceOptimizer: checkTCP(optimizerAddr),
	}

	healthMu.Lock()
	for k, v := range results {
		health[k] = v
	}
	healthMu.Unlock()

	for svc, s := range results {
		if s.Available {
			log.Printf("[health] %s: available", svc)
		} else {
			log.Printf("[health] %s: UNAVAILABLE — %s", svc, s.Error)
		}
	}
}

// StartHealthChecker runs an immediate health check and then repeats every 10 minutes.
// Call this once during application startup.
func StartHealthChecker() {
	runHealthChecks()
	go func() {
		ticker := time.NewTicker(healthCheckPeriod)
		defer ticker.Stop()
		for range ticker.C {
			runHealthChecks()
		}
	}()
}
