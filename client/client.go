package main

import (
	"github.com/EricChiquitoG/Remanet_DSM/client/src"
)

func main() {
	// Create a new grpc client

	// give us a context that we can cancel, but also a timeout just to illustrate a point
	src.ProcessDirectory(src.MyDir)

}
