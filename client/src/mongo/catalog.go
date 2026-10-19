package database

// Generalized, use-case-agnostic matchmaking data model.
//
// These collections replace the per-domain hardcoding that used to live in
// Go structs (ShaftManufacturer/Disassembler/Lamination in directory.go,
// the fixed Product/Components shape in product.go) and in static files
// (directory.json, the mongosh fixture scripts). A product family (electric
// motor, cutting tool, ...) is described once as a UseCase document; every
// provider, capability and catalog product is tagged with a UseCase id and
// carries its domain-specific fields in an open map rather than named Go
// fields. Adding a new product family is a new UseCase + a new import run,
// not a new struct or a new branch in the matching code.
//
// Populated by client/cmd/importdata from client/data/usecases/*.json and
// the corresponding CSV datasets under client/data/NewData. Nothing in the
// live request/matchmaking path reads these yet — that's a follow-up once
// Requirements/Product are generalized to match.

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	UseCaseCollection        *mongo.Collection
	ProviderCollection       *mongo.Collection
	CapabilityCollection     *mongo.Collection
	ProductCatalogCollection *mongo.Collection
	OwnerCollection          *mongo.Collection
)

const (
	UseCaseCollectionName        = "use_cases"
	ProviderCollectionName       = "providers"
	CapabilityCollectionName     = "capabilities"
	ProductCatalogCollectionName = "product_catalog"
	OwnerCollectionName          = "owners"
)

// initCatalogCollections wires up the collection handles. Called from
// InitMongoDB once the database handle exists.
func initCatalogCollections() {
	UseCaseCollection = Database.Collection(UseCaseCollectionName)
	ProviderCollection = Database.Collection(ProviderCollectionName)
	CapabilityCollection = Database.Collection(CapabilityCollectionName)
	ProductCatalogCollection = Database.Collection(ProductCatalogCollectionName)
	OwnerCollection = Database.Collection(OwnerCollectionName)
}

// createCatalogIndexes creates secondary indexes for the generalized
// collections. _id is already uniquely indexed by MongoDB, so only the
// lookup-pattern indexes need creating explicitly.
// Called from createIndexes alongside the existing collections.
func createCatalogIndexes(ctx context.Context) error {
	if _, err := ProviderCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "useCase", Value: 1}},
	}); err != nil {
		return err
	}
	if _, err := CapabilityCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "useCase", Value: 1}, {Key: "operation", Value: 1}}},
		{Keys: bson.D{{Key: "providerId", Value: 1}}},
	}); err != nil {
		return err
	}
	if _, err := ProductCatalogCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "useCase", Value: 1}}},
		{Keys: bson.D{{Key: "batch", Value: 1}}},
	}); err != nil {
		return err
	}
	if _, err := OwnerCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "useCase", Value: 1}},
	}); err != nil {
		return err
	}
	return nil
}

// ── Models ───────────────────────────────────────────────────────────────

// ConditionModel describes how "current condition" is represented for a
// product family. Two shapes are known so far:
//   - "perComponent": a fixed set of named components, each with a predicted
//     (self) and ground-truth (real) assessment — electric motors.
//   - "overallWithDefects": one overall self-assessment plus a set of named
//     defect measurements — cutting tools.
//
// A third product family can reuse either shape (config-only change) or,
// if it genuinely needs a new one, add a case the importer switches on —
// still one place, not one per call site.
type ConditionModel struct {
	Type         string   `bson:"type" json:"type"`
	Components   []string `bson:"components,omitempty" json:"components,omitempty"`
	OverallField string   `bson:"overallField,omitempty" json:"overallField,omitempty"`
	DefectFields []string `bson:"defectFields,omitempty" json:"defectFields,omitempty"`
}

// AttributeRule declares how one requirement attribute should be compared,
// for attributes where plain exact-value-set membership (the default when
// an attribute has no rule) is the wrong semantics:
//
//   - "numeric_min": the attribute is a physical capacity (AxisHeight,
//     RatedPower, ShaftLength, ShaftDiameter, ...). Equipment satisfies a
//     requirement if ANY of its values, read as a leading number (units
//     and formatting ignored — "355", "355 mm", "315.000 kW" all parse the
//     same), is >= the requirement's number. A 400 mm-rated machine can
//     obviously also take a 355 mm job; exact string equality can't express
//     that at all.
//   - "ordinal": the attribute is a ranked category (ShaftToleranceClass:
//     bad < mid < good). Equipment satisfies a requirement if any of its
//     values ranks at or above the requirement's value in Order (worst to
//     best). A "good"-precision machine should also satisfy a "mid" ask.
//
// Attributes with no rule keep the original exact-match-with-all-requested-
// values behavior (correct for nominal categories like CoatingType).
type AttributeRule struct {
	Type  string   `bson:"type" json:"type"`
	Order []string `bson:"order,omitempty" json:"order,omitempty"` // ordinal only, worst to best
}

// IdentityColumns maps the generalized identity fields every product family
// carries to that family's CSV/AAS column names. There is deliberately no
// "batch" column here: the batch identifier comes from the source filename
// (batch_000.csv -> "batch_000"), not from a CSV column. The dataset's own
// BatchInformation-style column is a decorative per-row label, not a stable
// grouping key — verified empirically: every row in one batch_NNN.csv file
// is identical on every matchmaking-relevant column, and that per-row label
// varies within the file without meaning anything.
type IdentityColumns struct {
	Serial          string `bson:"serial" json:"serial"`
	OwnerID         string `bson:"ownerId" json:"ownerId"`
	CurrentHolderID string `bson:"currentHolderId" json:"currentHolderId"`
	Manufacturer    string `bson:"manufacturer" json:"manufacturer"`
	Designation     string `bson:"designation" json:"designation"`
}

// UseCase is the schema for one product family: which operations its
// providers offer, which attributes describe a product/requirement, and how
// its condition data is shaped. Mirrors client/data/usecases/*.json.
type UseCase struct {
	ID                string          `bson:"_id" json:"id"`
	DisplayName       string          `bson:"displayName" json:"displayName"`
	Operations        []string        `bson:"operations" json:"operations"`
	RequirementFields []string        `bson:"requirementFields" json:"requirementFields"`
	Condition         ConditionModel  `bson:"condition" json:"condition"`
	Identity          IdentityColumns `bson:"identity" json:"identity"`
	// ExistenceOnlyOperations lists operations for which the source data
	// never carries equipment attributes (every provider's equipment for
	// them is either absent or the "offered, unspecified" sentinel). A
	// requirement is matched against these by offering alone — attribute
	// matching would either be vacuous or would wrongly reject every
	// provider. Electric motors: AssemblyTesting, Reuse, Replace.
	ExistenceOnlyOperations []string `bson:"existenceOnlyOperations,omitempty" json:"existenceOnlyOperations,omitempty"`
	// AttributeRules declares non-default comparison semantics per
	// requirement attribute name — see AttributeRule. Attributes not listed
	// here keep exact-value-set matching.
	AttributeRules map[string]AttributeRule `bson:"attributeRules,omitempty" json:"attributeRules,omitempty"`
	// KBSProductType is the Holonix KBS's own productType string for this
	// use case (e.g. "Motor" for electric-motor) — the join key used to
	// pick a use case from a kbs_output.json file's productType field.
	KBSProductType string `bson:"kbsProductType,omitempty" json:"kbsProductType,omitempty"`
	// KBSAttributeAliases maps a normalized KBS requirementName (lowercased,
	// trailing instance digits stripped — "axisHeightMotor001" ->
	// "axisheightmotor") to the canonical entry in RequirementFields
	// ("AxisHeight"). The KBS qualifies every requirement name with the
	// artifact it belongs to and an instance number; this table is the only
	// place that per-use-case translation needs to live.
	KBSAttributeAliases map[string]string `bson:"kbsAttributeAliases,omitempty" json:"kbsAttributeAliases,omitempty"`
}

// ProviderRecord is one remanufacturing actor within one use case. Replaces
// the hand-maintained Contact entries in directory.json.
type ProviderRecord struct {
	ID         string    `bson:"_id" json:"id"`
	UseCase    string    `bson:"useCase" json:"useCase"`
	ProviderID string    `bson:"providerId" json:"providerId"`
	Name       string    `bson:"name" json:"name"`
	Country    string    `bson:"country" json:"country"`
	Location   []float64 `bson:"location" json:"location"` // [lat, lon]
	Operations []string  `bson:"operations" json:"operations"`
}

// Equipment is one piece of equipment a provider uses to perform an
// operation, with whatever attributes the source data specified for it
// (e.g. ShaftToleranceClass, CoatingType). Each attribute is a list of
// values rather than a single string because the source data itself can be
// list-valued (e.g. MaterialCategory: ['Ceramic', 'Carbide']) — matching
// needs to check "does this equipment support every requested value",
// which needs the individual values, not a pre-joined display string.
// Units, where present, are embedded in the value string as-is.
type Equipment struct {
	Name       string              `bson:"name" json:"name"`
	Attributes map[string][]string `bson:"attributes" json:"attributes"`
}

// CapabilityRecord is one (provider, operation) pair — the normalized,
// query-friendly projection of an IDTA CapabilityContainer. The raw AAS JSON
// a real capability response came from (if any) stays in AASActorCollection;
// this is what matching should actually read.
type CapabilityRecord struct {
	ID         string      `bson:"_id" json:"id"`
	UseCase    string      `bson:"useCase" json:"useCase"`
	ProviderID string      `bson:"providerId" json:"providerId"`
	Operation  string      `bson:"operation" json:"operation"`
	Energy     *float64    `bson:"energy,omitempty" json:"energy,omitempty"`
	Cost       *float64    `bson:"cost,omitempty" json:"cost,omitempty"`
	Equipment  []Equipment `bson:"equipment" json:"equipment"`
}

// CatalogProduct is one *batch* — the real matchmaking individual — not one
// physical unit. Each source batch_NNN.csv file holds one or more rows that
// are identical on every matchmaking-relevant column (verified empirically:
// Attributes and Condition never vary within a file); the row count is
// simply how many physical units exist in that batch. So one CatalogProduct
// is built from a batch file's first row, Quantity records how many rows
// (units) it stood for, and SerialNumbers keeps every individual unit's
// serial for traceability without duplicating the batch record per unit.
// Attributes holds the family-specific technical fields (AxisHeight/
// RatedPower or ConnectionDiameter/CoatingMaterial, ...); Condition holds
// per-component or per-defect state, keyed by whatever ConditionModel the
// use case declares. This is also the eventual landing shape for data
// pulled from the Product-DT AAS instead of CSV.
type CatalogProduct struct {
	ID                    string                       `bson:"_id" json:"id"`
	UseCase               string                       `bson:"useCase" json:"useCase"`
	Batch                 string                       `bson:"batch" json:"batch"` // from the filename, e.g. "batch_000"
	Quantity              int                          `bson:"quantity" json:"quantity"`
	SerialNumbers         []string                     `bson:"serialNumbers,omitempty" json:"serialNumbers,omitempty"`
	OwnerID               string                       `bson:"ownerId,omitempty" json:"ownerId,omitempty"`
	CurrentHolderID       string                       `bson:"currentHolderId,omitempty" json:"currentHolderId,omitempty"`
	Manufacturer          string                       `bson:"manufacturer,omitempty" json:"manufacturer,omitempty"`
	Designation           string                       `bson:"designation,omitempty" json:"designation,omitempty"`
	Attributes            map[string]string            `bson:"attributes" json:"attributes"`
	OverallSelfAssessment string                       `bson:"overallSelfAssessment,omitempty" json:"overallSelfAssessment,omitempty"`
	Condition             map[string]map[string]string `bson:"condition" json:"condition"`
	CreatedAt             time.Time                    `bson:"createdAt" json:"createdAt"`
}

// OwnerRecord is the legal owner of a product unit, scoped to one use case
// (owner IDs are not shared across product families in the source data).
type OwnerRecord struct {
	ID       string    `bson:"_id" json:"id"`
	UseCase  string    `bson:"useCase" json:"useCase"`
	OwnerID  string    `bson:"ownerId" json:"ownerId"`
	Name     string    `bson:"name" json:"name"`
	Country  string    `bson:"country" json:"country"`
	Location []float64 `bson:"location" json:"location"`
}

// ── Upserts (used by client/cmd/importdata) ─────────────────────────────

func UpsertUseCase(uc UseCase) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := UseCaseCollection.ReplaceOne(ctx, bson.M{"_id": uc.ID}, uc, options.Replace().SetUpsert(true))
	return err
}

func UpsertProvider(p ProviderRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ProviderCollection.ReplaceOne(ctx, bson.M{"_id": p.ID}, p, options.Replace().SetUpsert(true))
	return err
}

func UpsertCapability(c CapabilityRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := CapabilityCollection.ReplaceOne(ctx, bson.M{"_id": c.ID}, c, options.Replace().SetUpsert(true))
	return err
}

func UpsertCatalogProduct(p CatalogProduct) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := ProductCatalogCollection.ReplaceOne(ctx, bson.M{"_id": p.ID}, p, options.Replace().SetUpsert(true))
	return err
}

func UpsertOwner(o OwnerRecord) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := OwnerCollection.ReplaceOne(ctx, bson.M{"_id": o.ID}, o, options.Replace().SetUpsert(true))
	return err
}

// ── Reads (used by matching) ────────────────────────────────────────────

func GetUseCase(id string) (UseCase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var uc UseCase
	err := UseCaseCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&uc)
	return uc, err
}

// GetUseCaseByKBSProductType resolves a Holonix KBS productType string
// (e.g. "Motor") to the use case that declares it via kbsProductType. There
// is deliberately no fallback guess when no use case claims a given
// productType — an unmapped product family is a real gap to surface, not
// something to default past.
func GetUseCaseByKBSProductType(productType string) (UseCase, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var uc UseCase
	err := UseCaseCollection.FindOne(ctx, bson.M{"kbsProductType": productType}).Decode(&uc)
	return uc, err
}

// ListProvidersByUseCase returns every provider registered for one use
// case — the candidate pool techProviderSearch filters by operation.
func ListProvidersByUseCase(useCase string) ([]ProviderRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cur, err := ProviderCollection.Find(ctx, bson.M{"useCase": useCase})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []ProviderRecord
	err = cur.All(ctx, &out)
	return out, err
}

// GetCapability returns the (providerID, operation) capability within a use
// case. Returns mongo.ErrNoDocuments when the provider doesn't offer it.
func GetCapability(useCase, providerID, operation string) (CapabilityRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var c CapabilityRecord
	err := CapabilityCollection.FindOne(ctx, bson.M{"_id": useCase + ":" + providerID + ":" + operation}).Decode(&c)
	return c, err
}

// ListCapabilitiesByOperation returns every provider's capability for one
// operation within a use case — the candidate pool a matcher filters.
func ListCapabilitiesByOperation(useCase, operation string) ([]CapabilityRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cur, err := CapabilityCollection.Find(ctx, bson.M{"useCase": useCase, "operation": operation})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []CapabilityRecord
	err = cur.All(ctx, &out)
	return out, err
}
