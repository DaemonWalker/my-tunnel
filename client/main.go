package main

import (
	"bytes"
	"context"
	"flag"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	ctr "my-tunnel/proto"
	"net/http"
)

const (
	defaultName = "world"
)

var (
	addr = flag.String("addr", "localhost:58081", "the address to connect to")
)

func main() {
	flag.Parse()
	conn, err := grpc.Dial(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := ctr.NewTunnelClient(conn)

	stream, err := c.Streaming(context.Background())
	if err != nil {
		log.Fatalf("error creating stream: %v", err)
	}

	for {
		log.Println("receiving..")
		request, err := stream.Recv()
		log.Println("received!", err, request)
		if err != nil && request == nil {
			log.Fatalln(err, "|", request == nil)
			continue
		}
		go func() {
			if err != nil {
				_ = stream.Send(&ctr.StreamingRequest{Id: request.Id})
			}
			clientRequest := ctr.StreamingRequest{
				Id: request.Id,
			}
			httpRequest, _ := http.NewRequest(request.Method, "http://localhost:8081"+request.Url, nil)
			for v, k := range request.Header {
				httpRequest.Header.Set(v, k.Header[0])
			}
			if len(request.Body) > 0 {
				bodyBytes := bytes.NewBuffer(request.Body)
				httpRequest.Body = io.NopCloser(bodyBytes)
			}

			httpClient := http.Client{}
			response, err := httpClient.Do(httpRequest)
			if err != nil {
				log.Println("error during send http request", err)
				_ = stream.Send(&clientRequest)
				return
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				_ = stream.Send(&clientRequest)
				log.Println("error during read response body", err)
				return
			}

			header := make(map[string]*ctr.StringArray, 0)
			for k, v := range response.Header {
				header[k] = &ctr.StringArray{Header: v}
			}
			clientRequest.Body = body
			clientRequest.Header = header
			clientRequest.Status = int32(response.StatusCode)

			_ = stream.Send(&clientRequest)
		}()
	}
}
