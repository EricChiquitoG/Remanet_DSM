package src

import (
	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
)

func Match(pr *pb.Purchase, interests *InterestsData) bool {

	var answer = false
	for _, interest := range interests.Interests {
		if interest.Product == pr.ProductType {
			answer = true
		}

		//matchMap[offering.ProcessID] = false
	}

	return answer
}
