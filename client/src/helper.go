package src

import (
	"fmt"
	"slices"
	"sort"
)

func FindCommon(list1, list2 []string) []string {
	var common []string
	elementMap := make(map[string]bool)

	for _, item := range list2 {
		elementMap[item] = true
	}

	for _, item := range list1 {
		if elementMap[item] {
			common = append(common, item)
		}
	}
	return common
}

func CreateMap(Activities []string, resultCollection ResultCollection) map[string][]string {
	activityMap := make(map[string][]string)

	for _, task := range Activities {
		for _, result := range resultCollection.Results {
			if slices.Contains(result.Matches, task) {
				activityMap[task] = append(activityMap[task], result.ContactName)
			}
		}
	}
	return activityMap
}

func GetDistinct(input OptimizationInput) []string {
	unique := make(map[string]bool)
	for _, route := range input.Routes {
		for name := range route {
			unique[name] = true
		}
	}
	processes := make([]string, 0, len(unique))
	for name := range unique {
		processes = append(processes, name)
	}
	return processes
}

// DAGToPrerequisites converts a route DAG into the prerequisites map expected by the optimizer.
func DAGToPrerequisites(dag map[string]DAGNode) map[string][]string {
	prereqs := make(map[string][]string, len(dag))
	for name, node := range dag {
		prereqs[name] = node.Prerequisites
	}
	return prereqs
}

// TopologicalSort returns the process names in a valid execution order (Kahn's algorithm).
// Returns an error if the DAG contains a cycle.
func TopologicalSort(dag map[string]DAGNode) ([]string, error) {
	inDegree := make(map[string]int, len(dag))
	successors := make(map[string][]string, len(dag))
	for name := range dag {
		if _, ok := inDegree[name]; !ok {
			inDegree[name] = 0
		}
	}
	for name, node := range dag {
		for _, prereq := range node.Prerequisites {
			successors[prereq] = append(successors[prereq], name)
			inDegree[name]++
		}
	}

	queue := make([]string, 0)
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	result := make([]string, 0, len(dag))
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)
		next := append([]string{}, successors[node]...)
		sort.Strings(next)
		for _, succ := range next {
			inDegree[succ]--
			if inDegree[succ] == 0 {
				queue = append(queue, succ)
			}
		}
	}

	if len(result) != len(dag) {
		return nil, fmt.Errorf("DAG contains a cycle")
	}
	return result, nil
}

// augmentWithBoundaryNodes adds "origin" and "end" boundary nodes to a prerequisites map.
// "origin" becomes a prerequisite of every start node (those with no prerequisites).
// "end" depends on every terminal node (those that no other node depends on).
// This lets the optimizer include transport from the pickup location to the first step
// and from the last step to the delivery destination.
func augmentWithBoundaryNodes(prDAG map[string][]string) map[string][]string {
	isPrereq := make(map[string]bool)
	for _, prereqs := range prDAG {
		for _, p := range prereqs {
			isPrereq[p] = true
		}
	}

	augmented := make(map[string][]string, len(prDAG)+2)
	augmented["origin"] = []string{}
	var terminalNodes []string

	for node, prereqs := range prDAG {
		if len(prereqs) == 0 {
			augmented[node] = []string{"origin"}
		} else {
			augmented[node] = prereqs
		}
		if !isPrereq[node] {
			terminalNodes = append(terminalNodes, node)
		}
	}

	sort.Strings(terminalNodes)
	augmented["end"] = terminalNodes
	return augmented
}

func exportToJson(processes []string, CostMatrix map[string]map[string]float64, EnergyMatrix map[string]map[string]float64, distM map[string]map[string]float64, prDAG map[string][]string) map[string]interface{} {
	return map[string]interface{}{
		"processes":    prDAG,
		"CostMatrix":   CostMatrix,
		"EnergyMatrix": EnergyMatrix,
		"matrix":       distM,
	}
}
func IdentifyActorClasses(processes []string, actorProcesses map[string][]string) []string {
	requiredClasses := make(map[string]bool)

	for _, process := range processes {
		for class, classProcesses := range actorProcesses {
			for _, p := range classProcesses {
				if p == process {
					requiredClasses[class] = true
					break
				}
			}
		}
	}

	result := make([]string, 0, len(requiredClasses))
	for class := range requiredClasses {
		result = append(result, class)
	}
	return result
}
