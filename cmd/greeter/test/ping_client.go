package main


import (
	 pb "github.com/sky4access/grpc-gateway-example/pkg/greeter"

	"google.golang.org/grpc"
	"os"
	"time"
	"log"
	"context"
)


const (
	address     = "localhost:50051"
	defaultName = "hello"
)


func main() {

	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.Ping(ctx, &pb.TestRequest{Name: name})
	if err != nil {
		log.Fatalf("could not ping: %v", err)
	}
	log.Printf("Greeting: %s", r.Msg)
}