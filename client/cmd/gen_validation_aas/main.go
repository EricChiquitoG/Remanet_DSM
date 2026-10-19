// Command gen_validation_aas transforms
// client/data/dsm_validation_dataset/motor_providers_clustered.csv into
// one IDTA 02020 Capability Description AAS JSON file per provider,
// following the exact template of the project's 5 existing hand-written
// samples (client/data/NewData/electric-motors/V2/aas-instancies/
// CapabilityDescription-ElectricMotorProvider-{000..004}.json).
//
// Usage (from the repo root):
//
//	go run ./client/cmd/gen_validation_aas
package main

import (
	"flag"
	"log"

	"github.com/EricChiquitoG/Remanet_DSM/client/src"
)

func main() {
	csvPath := flag.String("csv", "client/data/dsm_validation_dataset/motor_providers_clustered.csv", "clustered providers CSV to read")
	outDir := flag.String("out", "client/data/dsm_validation_dataset/aas-instancies", "directory to write one CapabilityDescription AAS JSON file per provider into")
	flag.Parse()

	n, err := src.GenerateValidationAASFiles(*csvPath, *outDir)
	if err != nil {
		log.Fatalf("generating validation AAS files: %v", err)
	}
	log.Printf("wrote %d AAS capability files to %s", n, *outDir)
}
