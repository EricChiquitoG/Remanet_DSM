package main

import (
	"log"

	"github.com/EricChiquitoG/Remanet_DSM/client/src"
)

func main() {
	// Create a new grpc client

	// give us a context that we can cancel, but also a timeout just to illustrate a point
	dir, err := src.MyDir("./data/directory.json")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	allCosts, err := src.Costs("./data/cost.json")
	if err != nil {
		log.Fatalf("Error: %v", err)

	}
	src.ProcessDirectory(dir, allCosts)

}
