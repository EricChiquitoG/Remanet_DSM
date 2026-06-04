package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ProcessNode struct {
	ProcessName string   `json:"processName" bson:"processName"`
	ActorID     string   `json:"actorId" bson:"actorId"`
	Cost        float64  `json:"cost" bson:"cost"`
	Co2         float64  `json:"co2" bson:"co2"`
	Next        []string `json:"next" bson:"next"`
	Status      string   `json:"status" bson:"status"` // "pending", "in_progress", "completed"
}

type ExecutionGraph struct {
	GraphID                string                 `json:"graphId" bson:"graphId"`
	Nodes                  map[string]ProcessNode `json:"nodes" bson:"nodes"`
	CurrentSteps           []string               `json:"currentSteps" bson:"currentSteps"`
	ProcessSequence        []string               `json:"processSequence" bson:"processSequence"`
	TotalTransportCost     float64                `json:"totalTransportCost" bson:"totalTransportCost"`
	TotalManufacturingCost float64                `json:"totalManufacturingCost" bson:"totalManufacturingCost"`
}

func SaveExecutionGraph(graph ExecutionGraph) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Replace().SetUpsert(true)
	_, err := ExecutionCollection.ReplaceOne(ctx, bson.M{"graphId": graph.GraphID}, graph, opts)
	if err != nil {
		return fmt.Errorf("failed to save execution graph: %v", err)
	}
	return nil
}

func GetExecutionGraph(graphID string) (*ExecutionGraph, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var graph ExecutionGraph
	err := ExecutionCollection.FindOne(ctx, bson.M{"graphId": graphID}).Decode(&graph)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution graph: %v", err)
	}
	return &graph, nil
}

// AdvanceStep marks the current in-progress steps as completed and activates their successors.
// Returns the updated graph; CurrentSteps will be empty once the final step completes.
func AdvanceStep(graphID string) (*ExecutionGraph, error) {
	graph, err := GetExecutionGraph(graphID)
	if err != nil {
		return nil, err
	}
	if graph == nil {
		return nil, fmt.Errorf("execution graph %s not found", graphID)
	}

	nextSteps := make(map[string]bool)
	for _, stepName := range graph.CurrentSteps {
		node, ok := graph.Nodes[stepName]
		if !ok {
			continue
		}
		node.Status = "completed"
		graph.Nodes[stepName] = node
		for _, next := range node.Next {
			nextSteps[next] = true
		}
	}

	newCurrentSteps := make([]string, 0, len(nextSteps))
	for stepName := range nextSteps {
		node := graph.Nodes[stepName]
		node.Status = "in_progress"
		graph.Nodes[stepName] = node
		newCurrentSteps = append(newCurrentSteps, stepName)
	}
	graph.CurrentSteps = newCurrentSteps

	if err := SaveExecutionGraph(*graph); err != nil {
		return nil, err
	}
	return graph, nil
}
