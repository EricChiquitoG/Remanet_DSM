// Command importdata loads the demonstrator datasets under
// client/data/NewData into the generalized, use-case-agnostic Mongo
// collections defined in client/src/mongo/catalog.go.
//
// It is schema-driven: every use case it knows about is described by a
// client/data/usecases/*.json config (operations, requirement attributes,
// condition model, identity columns, source CSV paths). Adding a new
// product family means adding one such file — this program has no
// per-domain branches beyond the two known ConditionModel shapes.
//
// Usage (from the repo root):
//
//	go run ./client/cmd/importdata                 # parse + seed MongoDB
//	go run ./client/cmd/importdata -dry-run         # parse + print counts only
//	go run ./client/cmd/importdata -usecases-dir ./client/data/usecases
package main

import (
	"archive/zip"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
)

// ── Config shape (mirrors client/data/usecases/*.json) ──────────────────

type datasetConfig struct {
	Archive      string `json:"archive,omitempty"`
	ProvidersCsv string `json:"providersCsv"`
	OwnersCsv    string `json:"ownersCsv"`
	// ProductsDir holds one batch_NNN.csv file per matchmaking individual
	// (batch). Every row inside one file is identical on every
	// matchmaking-relevant column — row count is just how many physical
	// units that batch contains — so each file becomes exactly one
	// CatalogProduct, not one per row.
	ProductsDir string `json:"productsDir"`
}

type useCaseConfig struct {
	ID                      string                          `json:"id"`
	DisplayName             string                          `json:"displayName"`
	Operations              []string                        `json:"operations"`
	RequirementFields       []string                        `json:"requirementFields"`
	Condition               execdb.ConditionModel           `json:"condition"`
	Identity                execdb.IdentityColumns          `json:"identity"`
	ExistenceOnlyOperations []string                        `json:"existenceOnlyOperations,omitempty"`
	AttributeRules          map[string]execdb.AttributeRule `json:"attributeRules,omitempty"`
	KBSProductType          string                          `json:"kbsProductType,omitempty"`
	KBSAttributeAliases     map[string]string               `json:"kbsAttributeAliases,omitempty"`
	Dataset                 datasetConfig                   `json:"dataset"`
}

func (c useCaseConfig) toUseCase() execdb.UseCase {
	return execdb.UseCase{
		ID:                      c.ID,
		DisplayName:             c.DisplayName,
		Operations:              c.Operations,
		RequirementFields:       c.RequirementFields,
		AttributeRules:          c.AttributeRules,
		Condition:               c.Condition,
		Identity:                c.Identity,
		ExistenceOnlyOperations: c.ExistenceOnlyOperations,
		KBSProductType:          c.KBSProductType,
		KBSAttributeAliases:     c.KBSAttributeAliases,
	}
}

// ── main ─────────────────────────────────────────────────────────────────

func main() {
	repoRoot := flag.String("repo-root", ".", "repo root, used to resolve dataset paths from the use-case configs")
	usecasesDir := flag.String("usecases-dir", "client/data/usecases", "directory of use-case *.json configs, relative to repo-root")
	dryRun := flag.Bool("dry-run", false, "parse and report counts without connecting to MongoDB")
	flag.Parse()

	configPaths, err := filepath.Glob(filepath.Join(*repoRoot, *usecasesDir, "*.json"))
	if err != nil || len(configPaths) == 0 {
		log.Fatalf("no use-case configs found in %s: %v", *usecasesDir, err)
	}

	if !*dryRun {
		if err := execdb.InitMongoDB(); err != nil {
			log.Fatalf("failed to connect to MongoDB: %v", err)
		}
		defer execdb.Close()
	}

	for _, path := range configPaths {
		cfg, err := loadUseCaseConfig(path)
		if err != nil {
			log.Fatalf("%s: %v", path, err)
		}
		if err := importUseCase(*repoRoot, cfg, *dryRun); err != nil {
			log.Fatalf("[%s] import failed: %v", cfg.ID, err)
		}
	}
}

func loadUseCaseConfig(path string) (useCaseConfig, error) {
	var cfg useCaseConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("invalid config: %w", err)
	}
	return cfg, nil
}

func importUseCase(repoRoot string, cfg useCaseConfig, dryRun bool) error {
	log.Printf("[%s] %s", cfg.ID, cfg.DisplayName)

	providerRows, err := readCSVRows(repoRoot, cfg.Dataset, cfg.Dataset.ProvidersCsv)
	if err != nil {
		return fmt.Errorf("providers csv: %w", err)
	}
	ownerRows, err := readCSVRows(repoRoot, cfg.Dataset, cfg.Dataset.OwnersCsv)
	if err != nil {
		return fmt.Errorf("owners csv: %w", err)
	}
	batchFiles, err := listBatchFiles(repoRoot, cfg.Dataset)
	if err != nil {
		return fmt.Errorf("listing batch files: %w", err)
	}

	providers, capabilities, err := buildProviders(cfg, providerRows)
	if err != nil {
		return fmt.Errorf("build providers: %w", err)
	}
	owners := buildOwners(cfg, ownerRows)
	products, err := buildProducts(repoRoot, cfg, batchFiles)
	if err != nil {
		return fmt.Errorf("build products: %w", err)
	}

	log.Printf("[%s] parsed: %d providers, %d capabilities, %d owners, %d products",
		cfg.ID, len(providers), len(capabilities), len(owners), len(products))

	if dryRun {
		if len(capabilities) > 0 {
			sample, _ := json.MarshalIndent(capabilities[0], "", "  ")
			log.Printf("[%s] sample capability:\n%s", cfg.ID, sample)
		}
		if len(products) > 0 {
			sample, _ := json.MarshalIndent(products[0], "", "  ")
			log.Printf("[%s] sample product:\n%s", cfg.ID, sample)
		}
		return nil
	}

	if err := execdb.UpsertUseCase(cfg.toUseCase()); err != nil {
		return fmt.Errorf("upsert use case: %w", err)
	}
	for _, p := range providers {
		if err := execdb.UpsertProvider(p); err != nil {
			return fmt.Errorf("upsert provider %s: %w", p.ID, err)
		}
	}
	for _, c := range capabilities {
		if err := execdb.UpsertCapability(c); err != nil {
			return fmt.Errorf("upsert capability %s: %w", c.ID, err)
		}
	}
	for _, o := range owners {
		if err := execdb.UpsertOwner(o); err != nil {
			return fmt.Errorf("upsert owner %s: %w", o.ID, err)
		}
	}
	for _, pr := range products {
		if err := execdb.UpsertCatalogProduct(pr); err != nil {
			return fmt.Errorf("upsert product %s: %w", pr.ID, err)
		}
	}
	log.Printf("[%s] seeded", cfg.ID)
	return nil
}

// ── CSV / archive reading ────────────────────────────────────────────────

// readCSVRows opens relPath (either a plain repo-relative file, or an entry
// inside ds.Archive when set) and returns one map[header]value per row.
func readCSVRows(repoRoot string, ds datasetConfig, relPath string) ([]map[string]string, error) {
	rc, err := openDatasetFile(repoRoot, ds, relPath)
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	r := csv.NewReader(rc)
	r.FieldsPerRecord = -1
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("empty csv: %s", relPath)
	}
	header := records[0]
	rows := make([]map[string]string, 0, len(records)-1)
	for _, rec := range records[1:] {
		row := make(map[string]string, len(header))
		for i, col := range header {
			if i < len(rec) {
				row[col] = rec[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func openDatasetFile(repoRoot string, ds datasetConfig, relPath string) (io.ReadCloser, error) {
	if ds.Archive == "" {
		return os.Open(filepath.Join(repoRoot, relPath))
	}
	zr, err := zip.OpenReader(filepath.Join(repoRoot, ds.Archive))
	if err != nil {
		return nil, fmt.Errorf("open archive %s: %w", ds.Archive, err)
	}
	for _, f := range zr.File {
		if f.Name == relPath {
			inner, err := f.Open()
			if err != nil {
				zr.Close()
				return nil, err
			}
			return &zipEntryReader{inner: inner, archive: zr}, nil
		}
	}
	zr.Close()
	return nil, fmt.Errorf("%s not found in %s", relPath, ds.Archive)
}

// listBatchFiles returns the relative paths (in the same form readCSVRows
// expects) of every batch_*.csv file under ds.ProductsDir, sorted so import
// order is deterministic. Handles both a plain directory and an entry
// inside ds.Archive.
func listBatchFiles(repoRoot string, ds datasetConfig) ([]string, error) {
	if ds.Archive == "" {
		entries, err := os.ReadDir(filepath.Join(repoRoot, ds.ProductsDir))
		if err != nil {
			return nil, err
		}
		var files []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".csv") {
				files = append(files, filepath.Join(ds.ProductsDir, e.Name()))
			}
		}
		sort.Strings(files)
		return files, nil
	}

	zr, err := zip.OpenReader(filepath.Join(repoRoot, ds.Archive))
	if err != nil {
		return nil, fmt.Errorf("open archive %s: %w", ds.Archive, err)
	}
	defer zr.Close()

	prefix := ds.ProductsDir + "/"
	var files []string
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, prefix) && strings.HasSuffix(f.Name, ".csv") {
			files = append(files, f.Name)
		}
	}
	sort.Strings(files)
	return files, nil
}

// zipEntryReader closes both the entry and the archive it came from.
type zipEntryReader struct {
	inner   io.ReadCloser
	archive *zip.ReadCloser
}

func (z *zipEntryReader) Read(p []byte) (int, error) { return z.inner.Read(p) }
func (z *zipEntryReader) Close() error {
	z.inner.Close()
	return z.archive.Close()
}

// ── value helpers ────────────────────────────────────────────────────────

func isBlank(v string) bool { return v == "" || v == "N/A" }

func parseFloatPtr(raw string) *float64 {
	if isBlank(raw) {
		return nil
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return nil
	}
	return &f
}

// attrValueToList renders one equipment-attribute value (string, list, or
// nested structure — the source data is a Python dict literal, so this is
// deliberately permissive) as a list of scalar values. A plain string
// becomes a single-element list; a source list (e.g. MaterialCategory:
// ['Ceramic', 'Carbide']) is kept as multiple elements so matching can check
// "does this equipment support every requested value" against the actual
// values, not a pre-joined display string.
func attrValueToList(v any) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			out = append(out, attrValueToList(e)...)
		}
		return out
	case nil:
		return nil
	default:
		b, _ := json.Marshal(t)
		return []string{string(b)}
	}
}

// parseEquipmentDict turns a source cell like
// "{'RoboticAssemblyService': {'PayloadCapacity': '10 kg'}, 'OperatorGuidingTool': {}}"
// into a list of Equipment. The source is a Python dict literal; every
// sampled cell in both datasets converts cleanly to JSON via a plain
// quote swap, so that's all this does — anything that fails to parse is
// reported, not silently dropped.
func parseEquipmentDict(raw string) ([]execdb.Equipment, error) {
	if isBlank(raw) {
		return nil, nil
	}
	jsonish := strings.ReplaceAll(raw, "'", "\"")
	var parsed map[string]map[string]any
	if err := json.Unmarshal([]byte(jsonish), &parsed); err != nil {
		return nil, fmt.Errorf("equipment cell %q: %w", raw, err)
	}
	equipment := make([]execdb.Equipment, 0, len(parsed))
	for name, attrs := range parsed {
		if name == "" {
			// {'': {}} marks "offered, equipment unspecified" — not an
			// equipment entry in its own right.
			continue
		}
		attributes := make(map[string][]string, len(attrs))
		for k, v := range attrs {
			attributes[k] = attrValueToList(v)
		}
		equipment = append(equipment, execdb.Equipment{Name: name, Attributes: attributes})
	}
	return equipment, nil
}

// ── record builders ──────────────────────────────────────────────────────

func buildProviders(cfg useCaseConfig, rows []map[string]string) ([]execdb.ProviderRecord, []execdb.CapabilityRecord, error) {
	var providers []execdb.ProviderRecord
	var capabilities []execdb.CapabilityRecord

	for _, row := range rows {
		providerID := row["IdProvider"]
		lat, _ := strconv.ParseFloat(row["Latitude"], 64)
		lon, _ := strconv.ParseFloat(row["Longitude"], 64)

		var offered []string
		for _, op := range cfg.Operations {
			eqRaw := row[op+"Equipments"]
			if isBlank(eqRaw) {
				continue // operation not offered by this provider
			}
			equipment, err := parseEquipmentDict(eqRaw)
			if err != nil {
				return nil, nil, fmt.Errorf("provider %s op %s: %w", providerID, op, err)
			}
			offered = append(offered, op)
			capabilities = append(capabilities, execdb.CapabilityRecord{
				ID:         cfg.ID + ":" + providerID + ":" + op,
				UseCase:    cfg.ID,
				ProviderID: providerID,
				Operation:  op,
				Energy:     parseFloatPtr(row[op+"Energy"]),
				Cost:       parseFloatPtr(row[op+"Cost"]),
				Equipment:  equipment,
			})
		}

		providers = append(providers, execdb.ProviderRecord{
			ID:         cfg.ID + ":" + providerID,
			UseCase:    cfg.ID,
			ProviderID: providerID,
			Name:       row["Name"],
			Country:    row["Country"],
			Location:   []float64{lat, lon},
			Operations: offered,
		})
	}
	return providers, capabilities, nil
}

func buildOwners(cfg useCaseConfig, rows []map[string]string) []execdb.OwnerRecord {
	owners := make([]execdb.OwnerRecord, 0, len(rows))
	for _, row := range rows {
		ownerID := row["IdOwner"]
		lat, _ := strconv.ParseFloat(row["Latitude"], 64)
		lon, _ := strconv.ParseFloat(row["Longitude"], 64)
		owners = append(owners, execdb.OwnerRecord{
			ID:       cfg.ID + ":" + ownerID,
			UseCase:  cfg.ID,
			OwnerID:  ownerID,
			Name:     row["Name"],
			Country:  row["Country"],
			Location: []float64{lat, lon},
		})
	}
	return owners
}

// buildProducts turns each batch_NNN.csv file into exactly one
// CatalogProduct. Every row within a file is identical on every
// matchmaking-relevant column (verified against the real dataset — this is
// not an assumption), so the first row supplies Attributes/Condition; the
// row count becomes Quantity, and every row's serial is kept in
// SerialNumbers for traceability without duplicating the batch record.
func buildProducts(repoRoot string, cfg useCaseConfig, batchFiles []string) ([]execdb.CatalogProduct, error) {
	products := make([]execdb.CatalogProduct, 0, len(batchFiles))
	now := time.Now()

	for _, relPath := range batchFiles {
		rows, err := readCSVRows(repoRoot, cfg.Dataset, relPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", relPath, err)
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("%s: no rows", relPath)
		}
		batchID := strings.TrimSuffix(filepath.Base(relPath), ".csv")
		row := rows[0]

		serials := make([]string, 0, len(rows))
		for _, r := range rows {
			if s := r[cfg.Identity.Serial]; s != "" {
				serials = append(serials, s)
			}
		}

		attributes := make(map[string]string)
		for _, field := range cfg.RequirementFields {
			if v := row[field]; !isBlank(v) {
				attributes[field] = v
			}
		}

		product := execdb.CatalogProduct{
			ID:              cfg.ID + ":" + batchID,
			UseCase:         cfg.ID,
			Batch:           batchID,
			Quantity:        len(rows),
			SerialNumbers:   serials,
			OwnerID:         valueOrEmpty(row, cfg.Identity.OwnerID),
			CurrentHolderID: valueOrEmpty(row, cfg.Identity.CurrentHolderID),
			Manufacturer:    valueOrEmpty(row, cfg.Identity.Manufacturer),
			Designation:     valueOrEmpty(row, cfg.Identity.Designation),
			Attributes:      attributes,
			Condition:       map[string]map[string]string{},
			CreatedAt:       now,
		}

		switch cfg.Condition.Type {
		case "perComponent":
			for _, component := range cfg.Condition.Components {
				entry := map[string]string{}
				if v := row[component+"SelfAssessment"]; !isBlank(v) {
					entry["self"] = v
				}
				if v := row[component+"RealAssessment"]; !isBlank(v) {
					entry["real"] = v
				}
				if v := row[component+"SerialInformation"]; !isBlank(v) {
					entry["serial"] = v
				}
				if len(entry) > 0 {
					product.Condition[component] = entry
				}
			}
		case "overallWithDefects":
			product.OverallSelfAssessment = row[cfg.Condition.OverallField]
			for _, defect := range cfg.Condition.DefectFields {
				if v := row[defect]; !isBlank(v) {
					product.Condition[defect] = map[string]string{"value": v}
				}
			}
		default:
			return nil, fmt.Errorf("unknown condition model type %q", cfg.Condition.Type)
		}

		products = append(products, product)
	}
	return products, nil
}

func valueOrEmpty(row map[string]string, col string) string {
	if col == "" {
		return ""
	}
	if v := row[col]; !isBlank(v) {
		return v
	}
	return ""
}
