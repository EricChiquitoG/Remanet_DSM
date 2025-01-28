package src

import (
	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
	"google.golang.org/protobuf/types/known/structpb"
)

func Match(pr *pb.Process, offerings *Offerings) []string {

	var matchMap []string

	for _, offering := range offerings.Offerings {
		for _, product := range offering.Products {
			if product.Product_name == pr.ProductType {
				matchMap = append(matchMap, offering.ProcessID)
			}

		}
		//matchMap[offering.ProcessID] = false
	}

	return matchMap
}

func MapToStruct(input map[string]bool) (*structpb.Struct, error) {
	// Create a map[string]interface{} for compatibility
	convertedMap := make(map[string]interface{})
	for key, value := range input {
		convertedMap[key] = value
	}

	// Create the structpb.Struct
	return structpb.NewStruct(convertedMap)
}
