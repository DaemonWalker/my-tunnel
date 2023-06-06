package main

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	ctr "my-tunnel/proto"
	"net/http"
)

type server struct {
	ctr.UnimplementedTunnelServer
}

type redirectMessage struct {
	inbound   *ctr.StreamingReply
	replyChan chan *ctr.StreamingRequest
}

var requests = make(chan *redirectMessage)
var chanMap = make(map[string]*chan *ctr.StreamingRequest, 0)

func IncomingRequest(writer http.ResponseWriter, request *http.Request) {
	log.Println("incoming request..")
	header := make(map[string]*ctr.StringArray, 0)
	for k, v := range request.Header {
		header[k] = &ctr.StringArray{Header: v}
	}

	bytes, err := io.ReadAll(request.Body)
	if err != nil {
		log.Println(err)
		return
	}

	reply := ctr.StreamingReply{
		Url:    request.RequestURI,
		Header: header,
		Body:   bytes,
		Method: request.Method,
		Id:     uuid.New().String(),
	}
	req := redirectMessage{
		inbound:   &reply,
		replyChan: make(chan *ctr.StreamingRequest),
	}
	requests <- &req

	response := <-req.replyChan
	for k, v := range response.Header {
		writer.Header().Set(k, v.Header[0])
	}
	writer.Write(response.Body)
}

func (s *server) CreateTunnel(ctx context.Context, in *ctr.TunnelRequest) (*ctr.TunnelReply, error) {
	return &ctr.TunnelReply{Url: "http://localhost:80" + in.GetMessage()}, nil
}

func (s *server) Streaming(stream ctr.Tunnel_StreamingServer) error {
	go func() {
		for {
			rep, err := stream.Recv()
			log.Println("receive from stream", rep)
			if err != nil || rep == nil {
				fmt.Printf("server: error receiving from stream: %v\n", err)
				if err == io.EOF {
					return
				}
				if rep != nil {
					*chanMap[rep.Id] <- nil
				}
				return
			}
			*chanMap[rep.Id] <- rep
		}
	}()
	for {
		req := <-requests
		log.Println("get request from chan", req)
		go func() {
			chanMap[req.inbound.Id] = &req.replyChan
			err := stream.Send(req.inbound)
			if err != nil {
				fmt.Printf("server: error send %v\n", err)
				chanMap[req.inbound.Id] = nil
			}
		}()
	}
	return nil
}
