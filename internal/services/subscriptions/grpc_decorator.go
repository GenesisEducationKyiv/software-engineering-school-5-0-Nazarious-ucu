package subscriptions

import (
	"context"

	"github.com/Nazarious-ucu/weather-subscription-api/internal/models"

	"github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/subs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type subscriber interface {
	Subscribe(ctx context.Context, data models.UserSubData) error
	Confirm(ctx context.Context, token string) (bool, error)
	Unsubscribe(ctx context.Context, token string) (bool, error)
}

type SubscriptionGRPCServer struct {
	subs.UnimplementedSubscriptionServiceServer
	service subscriber
}

func NewSubscriptionGRPCServer(service subscriber) *SubscriptionGRPCServer {
	return &SubscriptionGRPCServer{service: service}
}

func (s *SubscriptionGRPCServer) Subscribe(
	ctx context.Context,
	req *subs.SubscribeRequest,
) (*subs.MessageResponse, error) {
	data := models.UserSubData{
		Email:     req.GetEmail(),
		City:      req.GetCity(),
		Frequency: req.GetFrequency(),
	}

	err := s.service.Subscribe(ctx, data)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "subscribe failed: %v", err)
	}

	return &subs.MessageResponse{Message: "Subscribed successfully"}, nil
}

func (s *SubscriptionGRPCServer) Confirm(
	ctx context.Context,
	req *subs.TokenRequest,
) (*emptypb.Empty, error) {
	ok, err := s.service.Confirm(ctx, req.GetToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "confirm error: %v", err)
	}
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid or expired token")
	}
	return &emptypb.Empty{}, nil
}

func (s *SubscriptionGRPCServer) Unsubscribe(
	ctx context.Context,
	req *subs.TokenRequest,
) (*emptypb.Empty, error) {
	ok, err := s.service.Unsubscribe(ctx, req.GetToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "unsubscribe error: %v", err)
	}
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "invalid token or already unsubscribed")
	}
	return &emptypb.Empty{}, nil
}
