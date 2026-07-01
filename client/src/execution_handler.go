package src

import (
	"fmt"
	"net/http"

	execdb "github.com/EricChiquitoG/Remanet_DSM/client/src/mongo"
	"github.com/gin-gonic/gin"
)

// BuildExecutionGraph assembles the execution graph from the chosen solution.
// dagPrereqs is the prerequisites map for the route (inverted to derive successor edges).
func BuildExecutionGraph(graphID string, processSequence []string, choice ProcessedSolution, dagPrereqs map[string][]string) (execdb.ExecutionGraph, error) {
	nextMap := make(map[string][]string)
	for process, prereqs := range dagPrereqs {
		for _, prereq := range prereqs {
			nextMap[prereq] = append(nextMap[prereq], process)
		}
	}

	assignmentMap := make(map[string]string, len(choice.Assignments))
	for _, a := range choice.Assignments {
		assignmentMap[a.ProcessName] = a.ActorID
	}

	nodes := make(map[string]execdb.ProcessNode)
	for i, processName := range processSequence {
		actorID := assignmentMap[processName]
		var cost, co2 float64

		tp, err := GetTPData("TPRes_" + actorID)
		if err != nil {
			return execdb.ExecutionGraph{}, fmt.Errorf("failed to fetch TP data for actor %s: %v", actorID, err)
		}
		if tp != nil {
			for _, p := range tp.Processes {
				if p.ProcessName == processName {
					cost = p.Cost
					co2 = p.Co2
					break
				}
			}
		}

		status := "pending"
		if i == 0 {
			status = "in_progress"
		}

		nodes[processName] = execdb.ProcessNode{
			ProcessName: processName,
			ActorID:     actorID,
			Cost:        cost,
			Co2:         co2,
			Next:        nextMap[processName],
			Status:      status,
		}
	}

	return execdb.ExecutionGraph{
		GraphID:                graphID,
		Nodes:                  nodes,
		CurrentSteps:           []string{processSequence[0]},
		ProcessSequence:        processSequence,
		TotalTransportCost:     choice.TransportCost,
		TotalManufacturingCost: choice.ManufacturingCost,
	}, nil
}

// These vars default to the real DB implementations and can be replaced in tests.
var (
	getAllExecutionGraphsFn = execdb.GetAllExecutionGraphs
	advanceStepFn          = execdb.AdvanceStep
)

func GetAllExecutionGraphsHandler(c *gin.Context) {
	graphs, err := getAllExecutionGraphsFn()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, graphs)
}

func AdvanceStepHandler(c *gin.Context) {
	var req struct {
		GraphID string `json:"graphId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	graph, err := advanceStepFn(req.GraphID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, graph)
}
