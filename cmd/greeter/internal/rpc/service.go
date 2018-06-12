package rpc

import (
	pb "github.com/sky4access/grpc-gateway-example/pkg/greeter"

	"context"
)

var _ pb.GreeterServer = &Service{}


func NewService() *Service {
	return &Service{

	}
}

type Service struct {

}


func (s *Service) Ping(ctx context.Context, in *pb.TestRequest) (*pb.TestReply, error){

	return &pb.TestReply{Msg: "Pong"}, nil
}