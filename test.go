package main

import (
	"fmt"
	"log"

	"gonum.org/v1/gonum/graph/path"
	"gonum.org/v1/gonum/graph/simple"
)

func main() {
	fullGraph := simple.NewWeightedDirectedGraph(0, 0)

	// Define edges with weights and colors
	edges := []struct {
		from, to int64
		weight   float64
		color    string
	}{
		// Red paths (3 steps): 1 -> 2 -> 5 -> 4 and 1 -> 3 -> 6 -> 4
		{1, 2, 1.0, "red"},
		{2, 5, 1.0, "red"},
		{5, 4, 1.0, "red"},

		{1, 3, 1.0, "red"},
		{3, 6, 1.0, "red"},
		{6, 4, 1.0, "red"},

		// Blue paths (2 steps): 1 -> 7 -> 4 and 1 -> 8 -> 4
		{1, 7, 1.5, "blue"},
		{7, 4, 1.5, "blue"},

		{1, 8, 1.2, "blue"},
		{8, 4, 1.3, "blue"},
	}

	// Map to store edge color
	edgeColors := make(map[[2]int64]string)

	// Add edges and color mappings
	for _, e := range edges {
		if fullGraph.Node(e.from) == nil {
			fullGraph.AddNode(simple.Node(e.from))
		}
		if fullGraph.Node(e.to) == nil {
			fullGraph.AddNode(simple.Node(e.to))
		}
		fullGraph.SetWeightedEdge(fullGraph.NewWeightedEdge(
			fullGraph.Node(e.from),
			fullGraph.Node(e.to),
			e.weight,
		))
		edgeColors[[2]int64{e.from, e.to}] = e.color
	}

	// You can change this to "blue" or "red"
	targetColor := "blue"

	filtered := simple.NewWeightedDirectedGraph(0, 0)

	// Correct iteration over Nodes
	nodes := fullGraph.Nodes()
	for nodes.Next() {
		filtered.AddNode(nodes.Node())
	}

	// Correct iteration over Edges
	edgesx := fullGraph.WeightedEdges()
	for edgesx.Next() {
		e := edgesx.WeightedEdge()
		from, to := e.From().ID(), e.To().ID()
		if edgeColors[[2]int64{from, to}] == targetColor {
			filtered.SetWeightedEdge(e)
		}
	}

	// Compute shortest path
	source := filtered.Node(1)
	target := filtered.Node(4)

	if source == nil || target == nil {
		log.Fatal("Source or target node missing")
	}

	shortest := path.DijkstraFrom(source, filtered)
	pathNodes, weight := shortest.To(target.ID())

	fmt.Printf("Shortest path (%s): ", targetColor)
	for _, n := range pathNodes {
		fmt.Printf("%d ", n.ID())
	}
	fmt.Printf("\nTotal weight: %.2f\n", weight)
}
