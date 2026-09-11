package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// AASActorRecord stores the raw AAS JSON response for an actor, keyed by its URN address.
type AASActorRecord struct {
	ActorID     string `bson:"actorId"`
	AASResponse string `bson:"aasResponse"`
}

// GetMockAASJSON returns the raw AAS JSON string for the given actor address.
// Returns mongo.ErrNoDocuments when no record exists (caller can check with errors.Is).
func GetMockAASJSON(actorID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var record AASActorRecord
	if err := AASActorCollection.FindOne(ctx, bson.M{"actorId": actorID}).Decode(&record); err != nil {
		if err == mongo.ErrNoDocuments {
			return "", fmt.Errorf("no proxy record for %s", actorID)
		}
		return "", fmt.Errorf("proxy lookup failed for %s: %v", actorID, err)
	}
	return record.AASResponse, nil
}
