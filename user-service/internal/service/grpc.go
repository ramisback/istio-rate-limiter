package service

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/ramisback/istio-rate-limiter/user-service/internal/models"
	pb "github.com/ramisback/istio-rate-limiter/user-service/proto"
)

type grpcServer struct {
	pb.UnimplementedUserServiceServer
	service UserService
	logger  *zap.Logger
}

// NewGRPCServer creates a new gRPC server for the user service
func NewGRPCServer(service UserService, logger *zap.Logger) pb.UserServiceServer {
	return &grpcServer{
		service: service,
		logger:  logger,
	}
}

func (s *grpcServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.User, error) {
	user, err := models.NewUser(
		req.Email,
		req.Password,
		req.FirstName,
		req.LastName,
		req.Role,
	)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	if err := s.service.CreateUser(ctx, user); err != nil {
		if err == ErrEmailTaken {
			return nil, status.Error(codes.AlreadyExists, "email already taken")
		}
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return &pb.User{
		Id:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	}, nil
}

func (s *grpcServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	user, err := s.service.GetUser(ctx, req.Id)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	return &pb.User{
		Id:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	}, nil
}

func (s *grpcServer) UpdateUser(ctx context.Context, req *pb.UpdateUserRequest) (*pb.User, error) {
	user, err := s.service.GetUser(ctx, req.Id)
	if err != nil {
		if err == ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	user.UpdateDetails(req.Email, req.FirstName, req.LastName)
	if req.Password != "" {
		if err := user.UpdatePassword(req.Password); err != nil {
			return nil, status.Error(codes.Internal, "failed to update password")
		}
	}

	if err := s.service.UpdateUser(ctx, user); err != nil {
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &pb.User{
		Id:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	}, nil
}

func (s *grpcServer) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if err := s.service.DeleteUser(ctx, req.Id); err != nil {
		if err == ErrUserNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete user")
	}

	return &pb.DeleteUserResponse{}, nil
}

func (s *grpcServer) ValidateCredentials(ctx context.Context, req *pb.ValidateCredentialsRequest) (*pb.ValidateCredentialsResponse, error) {
	user, err := s.service.ValidateCredentials(ctx, req.Email, req.Password)
	if err != nil {
		if err == ErrInvalidCredentials {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Error(codes.Internal, "failed to validate credentials")
	}

	return &pb.ValidateCredentialsResponse{
		Valid: true,
		User: &pb.User{
			Id:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      user.Role,
		},
	}, nil
}
