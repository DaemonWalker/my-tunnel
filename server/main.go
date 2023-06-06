package main

import (
	"google.golang.org/grpc"
	"log"
	controller "my-tunnel/proto"
	"net"
	"net/http"
)

func main() {
	go func() {
		http.HandleFunc("/", IncomingRequest)
		log.Println("start http...")
		log.Fatalln(http.ListenAndServe(":58080", nil))
	}()
	lis, err := net.Listen("tcp", ":58081")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Println("start grpc...")
	grpcServer := grpc.NewServer()
	// 注册服务
	controller.RegisterTunnelServer(grpcServer, &server{})
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalln(err)
	}

}
