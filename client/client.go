package main

import (
	"github.com/EricChiquitoG/Remanet_DSM/client/src"
	"github.com/gin-gonic/gin"
)

func main() {
	// Create a new grpc client

	// give us a context that we can cancel, but also a timeout just to illustrate a point

	r := gin.Default()
	r.POST("/get_options", src.ProcessDirectory)

	r.Run(":8080")

}
