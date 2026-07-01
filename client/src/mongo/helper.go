package database

import (
	"context"
	//"encoding/csv"
	//"encoding/json"
	"fmt"
	//"io"
	"net/http"
	"strconv"

	//"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	//"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type GeoPoint struct {
	Type        string    `json:"type" bson:"type"`
	Coordinates []float64 `json:"coordinates" bson:"coordinates"` // [longitude, latitude]
}

// RequiredParameters contains the motor specifications
type RequiredParameters struct {
	RatedPower      float64 `json:"ratedPower" bson:"ratedPower"`
	AxisHeight      int     `json:"axisHeight" bson:"axisHeight"`
	EfficiencyClass string  `json:"efficiencyClass" bson:"efficiencyClass"`
	PhaseN          *int    `json:"phaseN,omitempty" bson:"phaseN,omitempty"`
}

// Proposal represents a purchase proposal document
type Proposal struct {
	ID                 primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	ProposalID         string             `json:"proposalId" bson:"proposalId"`
	CustomerID         string             `json:"customerId" bson:"customerId"`
	CustomerLocation   GeoPoint           `json:"customerLocation" bson:"customerLocation"`
	Cardinality        int                `json:"cardinality" bson:"cardinality"`
	ProductType        string             `json:"productType" bson:"productType"`
	RequiredParameters RequiredParameters `json:"requiredParameters" bson:"requiredParameters"`
	CreatedAt          time.Time          `json:"createdAt" bson:"createdAt"`
	UpdatedAt          time.Time          `json:"updatedAt" bson:"updatedAt"`
}

// ProposalCreate is used for creating new proposals (no timestamps)
type ProposalCreate struct {
	ProposalID         string             `json:"proposalId" bson:"proposalId" binding:"required"`
	CustomerID         string             `json:"customerId" bson:"customerId" binding:"required"`
	CustomerLocation   GeoPoint           `json:"customerLocation" bson:"customerLocation" binding:"required"`
	Cardinality        int                `json:"cardinality" bson:"cardinality" binding:"required,gt=0"`
	ProductType        string             `json:"productType" bson:"productType" binding:"required"`
	RequiredParameters RequiredParameters `json:"requiredParameters" bson:"requiredParameters" binding:"required"`
}

// ProposalUpdate is used for updating proposals (all fields optional)
type ProposalUpdate struct {
	CustomerID         *string             `json:"customerId,omitempty" bson:"customerId,omitempty"`
	CustomerLocation   *GeoPoint           `json:"customerLocation,omitempty" bson:"customerLocation,omitempty"`
	Cardinality        *int                `json:"cardinality,omitempty" bson:"cardinality,omitempty"`
	ProductType        *string             `json:"productType,omitempty" bson:"productType,omitempty"`
	RequiredParameters *RequiredParameters `json:"requiredParameters,omitempty" bson:"requiredParameters,omitempty"`
}

// BulkInsertResponse represents the response from bulk insert
type BulkInsertResponse struct {
	Success       bool     `json:"success"`
	InsertedCount int      `json:"insertedCount"`
	FailedCount   int      `json:"failedCount"`
	Errors        []string `json:"errors"`
}

// StatsResponse represents database statistics
type StatsResponse struct {
	TotalProposals     int64            `json:"totalProposals"`
	ByProductType      map[string]int64 `json:"byProductType"`
	ByEfficiencyClass  map[string]int64 `json:"byEfficiencyClass"`
	AverageCardinality float64          `json:"averageCardinality"`
}

// CSVRow represents a row from the customer dataset CSV
type CSVRow struct {
	IDCustomer      string  `csv:"id_customer"`
	Localization    string  `csv:"localization"`
	MotorN          int     `csv:"motor_n"`
	PhaseN          int     `csv:"phase_n"`
	RatedPower      float64 `csv:"rated_power"`
	EfficiencyClass string  `csv:"efficiency_class"`
	AxisHeight      int     `csv:"axis_height"`
}

// NewGeoPoint creates a GeoJSON Point
func NewGeoPoint(longitude, latitude float64) GeoPoint {
	return GeoPoint{
		Type:        "Point",
		Coordinates: []float64{longitude, latitude},
	}
}

// ToProposal converts ProposalCreate to Proposal with timestamps
func (pc *ProposalCreate) ToProposal() *Proposal {
	now := time.Now()
	return &Proposal{
		ProposalID:         pc.ProposalID,
		CustomerID:         pc.CustomerID,
		CustomerLocation:   pc.CustomerLocation,
		Cardinality:        pc.Cardinality,
		ProductType:        pc.ProductType,
		RequiredParameters: pc.RequiredParameters,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}
func GetBatchData(c *gin.Context) {
	graphID := c.Query("graphId")
	if graphID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "graphId query parameter is required"})
		return
	}
	graph, err := GetExecutionGraph(graphID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if graph == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("execution graph %s not found", graphID)})
		return
	}

	c.JSON(http.StatusOK, graph)
}

// CreateProposal creates a new purchase proposal
func AddProposal(c *gin.Context) {
	var proposalCreate ProposalCreate

	if err := c.ShouldBindJSON(&proposalCreate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if proposalId already exists
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existing Proposal
	err := Collection.FindOne(ctx, bson.M{"proposalId": proposalCreate.ProposalID}).Decode(&existing)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Proposal with ID '%s' already exists", proposalCreate.ProposalID)})
		return
	}

	// Convert to Proposal with timestamps
	proposal := proposalCreate.ToProposal()

	// Insert into database
	result, err := Collection.InsertOne(ctx, proposal)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	proposal.ID = result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusCreated, proposal)
}

// GetProposals lists proposals with filtering and pagination
func GetProposals(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Pagination parameters
	skip, _ := strconv.Atoi(c.DefaultQuery("skip", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if limit > 1000 {
		limit = 1000
	}

	// Build filter
	filter := bson.M{}

	if customerID := c.Query("customerId"); customerID != "" {
		filter["customerId"] = customerID
	}

	if productType := c.Query("productType"); productType != "" {
		filter["productType"] = productType
	}

	if efficiencyClass := c.Query("efficiencyClass"); efficiencyClass != "" {
		filter["requiredParameters.efficiencyClass"] = efficiencyClass
	}

	// Query options
	opts := options.Find().
		SetSkip(int64(skip)).
		SetLimit(int64(limit))

	// Execute query
	cursor, err := Collection.Find(ctx, filter, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)

	// Decode results
	var proposals []Proposal
	if err := cursor.All(ctx, &proposals); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if proposals == nil {
		proposals = []Proposal{}
	}

	c.JSON(http.StatusOK, proposals)
}
