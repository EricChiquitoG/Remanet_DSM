package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	Client              *mongo.Client
	Database            *mongo.Database
	Collection          *mongo.Collection
	ExecutionCollection *mongo.Collection
	ProductCollection   *mongo.Collection
	HistoryCollection   *mongo.Collection
)

const (
	DatabaseName            = "dsm_db"
	CollectionName          = "proposals"
	ExecutionCollectionName = "execution_graphs"
	ProductCollectionName   = "products"
	HistoryCollectionName   = "product_history"
)

// InitMongoDB initializes the MongoDB connection and creates indexes
func InitMongoDB() error {
	// Get MongoDB URL from environment or use default
	mongoURL := os.Getenv("MONGODB_URL")
	if mongoURL == "" {
		mongoURL = "mongodb://localhost:27017"
	}

	// Set client options
	clientOptions := options.Client().ApplyURI(mongoURL)

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	Client = client
	Database = client.Database(DatabaseName)
	Collection = Database.Collection(CollectionName)
	ExecutionCollection = Database.Collection(ExecutionCollectionName)
	ProductCollection = Database.Collection(ProductCollectionName)
	HistoryCollection = Database.Collection(HistoryCollectionName)

	// Create indexes
	if err := createIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create indexes: %v", err)
	}

	log.Printf("✓ Connected to MongoDB: %s", mongoURL)
	log.Printf("✓ Database: %s", DatabaseName)
	log.Printf("✓ Collections: %s, %s, %s, %s", CollectionName, ExecutionCollectionName, ProductCollectionName, HistoryCollectionName)
	log.Printf("✓ Indexes created")

	return nil
}

// createIndexes creates necessary indexes for the proposals collection
func createIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "proposalId", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{Keys: bson.D{{Key: "customerId", Value: 1}}},
		{Keys: bson.D{{Key: "customerLocation", Value: "2dsphere"}}},
		{Keys: bson.D{{Key: "productType", Value: 1}}},
		{Keys: bson.D{{Key: "requiredParameters.efficiencyClass", Value: 1}}},
	}
	if _, err := Collection.Indexes().CreateMany(ctx, indexes); err != nil {
		return err
	}

	if _, err := ExecutionCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "graphId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	if _, err := ProductCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "id", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return err
	}

	_, err := HistoryCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "productId", Value: 1}},
	})
	return err
}

// Close closes the MongoDB connection
func Close() error {
	if Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return Client.Disconnect(ctx)
	}
	return nil
}

// Ping checks if MongoDB connection is alive
func Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return Client.Ping(ctx, nil)
}
