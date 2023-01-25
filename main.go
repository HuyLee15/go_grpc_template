package main

import (
	"ahiho/todo_list/proto/todogrpc"
	"context"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type server struct {
	todogrpc.UnimplementedHelloWorldServiceServer
	DB *gorm.DB
}

func (s *server) HelloWorld(_ context.Context, req *todogrpc.HelloWorldRequest) (*todogrpc.HelloWorldResponse, error) {
	if req.Req == "" {
		return &todogrpc.HelloWorldResponse{
			Res: "hello world",
		}, nil
	}

	return &todogrpc.HelloWorldResponse{
		Res: "hello " + req.Req,
	}, nil
}

func NewServer(db *gorm.DB) *server {
	return &server{
		DB: db,
	}
}

func main() {
	db, err := gorm.Open(mysql.Open("root:akashi@tcp(127.0.0.1:3306)/go_grpc?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	// Create a listener on TCP port
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	// Create a gRPC server object
	s := grpc.NewServer()

	// Attach the Greeter service to the server
	todogrpc.RegisterHelloWorldServiceServer(s, NewServer(db))

	// Serve gRPC server
	log.Println("Serving gRPC on 0.0.0.0:8080")
	go func() {
		log.Fatalln(s.Serve(lis))
	}()

	// Create a client connection to the gRPC server we just started
	// This is where the gRPC-Gateway proxies the requests
	conn, err := grpc.DialContext(
		context.Background(),
		"0.0.0.0:8080",
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}

	gwmux := runtime.NewServeMux()
	// Register Greeter
	err = todogrpc.RegisterHelloWorldServiceHandler(context.Background(), gwmux, conn)
	if err != nil {
		log.Fatalln("Failed to register gateway:", err)
	}

	gwServer := &http.Server{
		Addr:    ":8090",
		Handler: gwmux,
	}

	log.Println("Serving gRPC-Gateway on http://0.0.0.0:8090")
	log.Fatalln(gwServer.ListenAndServe())

}
