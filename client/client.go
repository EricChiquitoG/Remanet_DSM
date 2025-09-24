package main

import (
	"context"
	"log"
	"net"
	"os"

	pb "github.com/EricChiquitoG/Remanet_DSM/DSM_protos"
	"github.com/EricChiquitoG/Remanet_DSM/client/src"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedSubmissionServiceServer
}

func main() {
	// REST API setup
	r := gin.Default()

	if err := src.InitData(); err != nil {
		log.Fatalf("Initialization error: %v", err)
	}

	r.POST("/get_options", src.ProcessDirectory)

	// Start REST API server in a goroutine
	go func() {
		if err := r.Run(":8080"); err != nil {
			log.Fatalf("failed to run REST server: %v", err)
		}
	}()

	// gRPC server setup
	port := os.Getenv("GRPC_PORT")
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterSubmissionServiceServer(grpcServer, &server{})

	// Start gRPC server (blocking)
	log.Printf("Starting gRPC server on port %s", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve gRPC: %s", err)
	}
}

func (s *server) EnrollServer(context context.Context, pr *pb.Enroll) (*pb.EnrollResponse, error) {

	newContact := src.Contact{
		Name:      pr.Name,
		Address:   pr.Address,
		Location:  pr.Location,
		Offerings: pr.Offerings,
	}
	err := src.UpdateJson(newContact)
	if err != nil {
		return &pb.EnrollResponse{
			Status:  "Unable to nroll user",
			Message: "Unable to set up data in server",
		}, nil
	}
	return &pb.EnrollResponse{
		Status:  "Data Setup succesfullyr",
		Message: "Ok",
	}, nil
}
