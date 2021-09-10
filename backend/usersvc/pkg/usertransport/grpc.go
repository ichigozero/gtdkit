package usertransport

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/ichigozero/gtdkit/backend/usersvc/pb"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userendpoint"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userservice"
	"google.golang.org/grpc"
)

type grpcServer struct {
	userID grpctransport.Handler
	pb.UnimplementedUserServer
}

func NewGRPCServer(endpoints userendpoint.Set, logger log.Logger) pb.UserServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
	}

	return &grpcServer{
		userID: grpctransport.NewServer(
			endpoints.UserIDEndpoint,
			decodeGRPCUserIDRequest,
			encodeGRPCUserIDResponse,
			options...,
		),
	}
}

func (s *grpcServer) UserID(ctx context.Context, req *pb.UserIDRequest) (*pb.UserIDReply, error) {
	_, rep, err := s.userID.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.UserIDReply), nil
}

func NewGRPCClient(conn *grpc.ClientConn, logger log.Logger) userservice.Service {
	var options []grpctransport.ClientOption

	var userEndpoint endpoint.Endpoint
	{
		userEndpoint = grpctransport.NewClient(
			conn,
			"pb.User",
			"UserID",
			encodeGRPCUserIDRequest,
			decodeGRPCUserIDResponse,
			pb.UserIDReply{},
			options...,
		).Endpoint()
	}

	return userendpoint.Set{
		UserIDEndpoint: userEndpoint,
	}
}

func decodeGRPCUserIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.UserIDRequest)
	return userendpoint.UserIDRequest{
		Name:     string(req.Name),
		Password: string(req.Password),
	}, nil
}

func encodeGRPCUserIDResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(userendpoint.UserIDResponse)
	return &pb.UserIDReply{Id: int64(resp.ID), Err: err2str(resp.Err)}, nil
}

func encodeGRPCUserIDRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(userendpoint.UserIDRequest)
	return &pb.UserIDRequest{Name: req.Name, Password: req.Password}, nil
}

func decodeGRPCUserIDResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.UserIDReply)
	return userendpoint.UserIDResponse{ID: int(reply.Id), Err: str2err(reply.Err)}, nil
}

func str2err(s string) error {
	if s == "" {
		return nil
	}
	return errors.New(s)
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
