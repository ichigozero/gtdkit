package usertransport

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/ichigozero/gtdkit/backend/usersvc"
	"github.com/ichigozero/gtdkit/backend/usersvc/pb"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userendpoint"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userservice"
	"google.golang.org/grpc"
)

type grpcServer struct {
	userID   grpctransport.Handler
	isExists grpctransport.Handler
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
		isExists: grpctransport.NewServer(
			endpoints.IsExistsEndpoint,
			decodeGRPCIsExistsRequest,
			encodeGRPCIsExistsResponse,
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

func (s *grpcServer) IsExists(ctx context.Context, req *pb.IsExistsRequest) (*pb.IsExistsReply, error) {
	_, rep, err := s.isExists.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.IsExistsReply), nil
}

func NewGRPCClient(conn *grpc.ClientConn, logger log.Logger) userservice.Service {
	var options []grpctransport.ClientOption

	var userIDEndpoint endpoint.Endpoint
	{
		userIDEndpoint = grpctransport.NewClient(
			conn,
			"pb.User",
			"UserID",
			encodeGRPCUserIDRequest,
			decodeGRPCUserIDResponse,
			pb.UserIDReply{},
			options...,
		).Endpoint()
	}

	var isExistsEndpoint endpoint.Endpoint
	{
		isExistsEndpoint = grpctransport.NewClient(
			conn,
			"pb.User",
			"IsExists",
			encodeGRPCIsExistsRequest,
			decodeGRPCIsExistsResponse,
			pb.IsExistsReply{},
			options...,
		).Endpoint()
	}

	return userendpoint.Set{
		UserIDEndpoint:   userIDEndpoint,
		IsExistsEndpoint: isExistsEndpoint,
	}
}

func decodeGRPCUserIDRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.UserIDRequest)
	return userendpoint.UserIDRequest{Name: string(req.Name), Password: string(req.Password)}, nil
}

func encodeGRPCUserIDResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(userendpoint.UserIDResponse)
	return &pb.UserIDReply{Id: resp.ID, Err: err2str(resp.Err)}, nil
}

func encodeGRPCUserIDRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(userendpoint.UserIDRequest)
	return &pb.UserIDRequest{Name: req.Name, Password: req.Password}, nil
}

func decodeGRPCUserIDResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.UserIDReply)
	return userendpoint.UserIDResponse{ID: reply.Id, Err: str2err(reply.Err)}, nil
}

func decodeGRPCIsExistsRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.IsExistsRequest)
	return userendpoint.IsExistsRequest{ID: req.Id}, nil
}

func encodeGRPCIsExistsResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(userendpoint.IsExistsResponse)
	return &pb.IsExistsReply{V: resp.V, Err: err2str(resp.Err)}, nil
}

func encodeGRPCIsExistsRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(userendpoint.IsExistsRequest)
	return &pb.IsExistsRequest{Id: req.ID}, nil
}

func decodeGRPCIsExistsResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.IsExistsReply)
	return userendpoint.IsExistsResponse{V: reply.V, Err: str2err(reply.Err)}, nil
}

func str2err(s string) error {
	if s == "" {
		return nil
	}

	switch s {
	case usersvc.ErrInvalidArgument.Error():
		return usersvc.ErrInvalidArgument
	case usersvc.ErrUserNotFound.Error():
		return usersvc.ErrUserNotFound
	}

	return errors.New(s)
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
