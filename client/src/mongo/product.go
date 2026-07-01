package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Valid component condition values.
const (
	ComponentGood     = "Good"
	ComponentWorn     = "Worn"
	ComponentDegraded = "Degraded"
	ComponentFailed   = "Failed"
)

var validComponentStatuses = map[string]bool{
	ComponentGood:     true,
	ComponentWorn:     true,
	ComponentDegraded: true,
	ComponentFailed:   true,
}

func IsValidComponentStatus(s string) bool {
	return validComponentStatuses[s]
}

// Components holds the condition of each fixed motor component.
type Components struct {
	Bearings string `json:"bearings" bson:"bearings"`
	Stator   string `json:"stator"   bson:"stator"`
	Rotor    string `json:"rotor"    bson:"rotor"`
	Housing  string `json:"housing"  bson:"housing"`
	Windings string `json:"windings" bson:"windings"`
	Shaft    string `json:"shaft"    bson:"shaft"`
}

// Product is the core motor asset record.
type Product struct {
	ID              string      `json:"id"              bson:"id"`
	Class           string      `json:"cls"             bson:"cls"`
	AxisHeight      float64     `json:"axis"            bson:"axis"`
	Power           float64     `json:"pow"             bson:"pow"`
	Batch           *string     `json:"batch"           bson:"batch"`           // null when not assigned to a batch
	Components      *Components `json:"components"      bson:"components"`      // null until first inspection
	OwnerHistory    []string    `json:"ownerHistory"    bson:"ownerHistory"`    // null until first inspection
	InspectionNotes *string     `json:"inspectionNotes" bson:"inspectionNotes"` // null until first inspection
	CreatedAt       time.Time   `json:"createdAt"       bson:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"       bson:"updatedAt"`
}

// ProductCreate is the body accepted by POST /products.
type ProductCreate struct {
	ID              string      `json:"id"              bson:"id"              binding:"required"`
	Class           string      `json:"cls"             bson:"cls"             binding:"required"`
	AxisHeight      float64     `json:"axis"            bson:"axis"            binding:"required"`
	Power           float64     `json:"pow"             bson:"pow"             binding:"required"`
	Batch           *string     `json:"batch"           bson:"batch"`
	Components      *Components `json:"components"      bson:"components"`
	OwnerHistory    []string    `json:"ownerHistory"    bson:"ownerHistory"`
	InspectionNotes *string     `json:"inspectionNotes" bson:"inspectionNotes"`
}

// HistoricalRecord stores the service history for a product.
// productId is stored in MongoDB for lookup but omitted from the JSON response
// (the caller already knows it from the URL).
type HistoricalRecord struct {
	ProductID   string   `json:"-"            bson:"productId"`
	Owners      []string `json:"owners"       bson:"owners"`
	Age         int      `json:"age"          bson:"age"`
	LastService string   `json:"lastService"  bson:"lastService"`
	Notes       string   `json:"notes"        bson:"notes"`
}

// --- DB functions -----------------------------------------------------------

func GetAllProducts() ([]Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := ProductCollection.Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"id": 1}))
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %v", err)
	}
	defer cursor.Close(ctx)

	var products []Product
	if err := cursor.All(ctx, &products); err != nil {
		return nil, fmt.Errorf("failed to decode products: %v", err)
	}
	if products == nil {
		products = []Product{}
	}
	return products, nil
}

func GetProductByID(id string) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var product Product
	err := ProductCollection.FindOne(ctx, bson.M{"id": id}).Decode(&product)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %v", err)
	}
	return &product, nil
}

func CreateProduct(pc ProductCreate) (*Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Reject duplicate IDs.
	var existing Product
	if err := ProductCollection.FindOne(ctx, bson.M{"id": pc.ID}).Decode(&existing); err == nil {
		return nil, fmt.Errorf("product with ID '%s' already exists", pc.ID)
	}

	now := time.Now()
	product := Product{
		ID:              pc.ID,
		Class:           pc.Class,
		AxisHeight:      pc.AxisHeight,
		Power:           pc.Power,
		Batch:           pc.Batch,
		Components:      pc.Components,
		OwnerHistory:    pc.OwnerHistory,
		InspectionNotes: pc.InspectionNotes,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if _, err := ProductCollection.InsertOne(ctx, product); err != nil {
		return nil, fmt.Errorf("failed to insert product: %v", err)
	}
	return &product, nil
}

func GetProductHistory(productID string) (*HistoricalRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var record HistoricalRecord
	err := HistoryCollection.FindOne(ctx, bson.M{"productId": productID}).Decode(&record)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get history: %v", err)
	}
	return &record, nil
}
