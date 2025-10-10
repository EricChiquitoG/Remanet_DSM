package src

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
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
		for _, stepName := range route {
			unique[stepName.Name] = true
		}
	}

	// Convert map keys back into a slice
	processes := make([]string, 0, len(unique))
	for name := range unique {
		processes = append(processes, name)
	}
	return processes
}

func exportToJson(processes []string, CostMatrix map[string]map[string]float64, distM map[string]map[string]float64) map[string]interface{} {
	data := map[string]interface{}{
		"processes":  processes,
		"CostMatrix": CostMatrix,
		"matrix":     distM,
	}

	file, err := os.Create("output.json")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return nil
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		fmt.Println("Error writing JSON:", err)
	} else {
		fmt.Println("Data written to output.json")
	}
	return data
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
func StepNamesFromRoute(steps []Step) []string {
	stepNames := make([]string, 0, len(steps))
	for _, step := range steps {
		stepNames = append(stepNames, step.Name)
	}
	return stepNames
}

// For logical processes that the DSM shouldnt are about logistics
func moveFlagToEnd(s []string, value string) ([]string, int) {
	flagIndex := -1

	for i, v := range s {
		if v == value {
			flagIndex = i
			break
		}
	}

	if flagIndex == -1 {
		return s, -1
	}

	s = append(s[:flagIndex], s[flagIndex+1:]...)

	s = append(s, value)

	return s, flagIndex
}
func moveEndToIndex(s []string, i int) []string {
	if len(s) == 0 || i < 0 || i >= len(s) {
		return s
	}
	last := s[len(s)-1]
	s = s[:len(s)-1]
	s = append(s[:i], append([]string{last}, s[i:]...)...)

	return s
}
