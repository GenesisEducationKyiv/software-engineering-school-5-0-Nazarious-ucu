package decorators

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	weatherpb "github.com/Nazarious-ucu/weather-subscription-api/protos/gen/go/v1.alpha/weather"
)

type WeatherGRPCServer struct {
	weatherpb.UnimplementedWeatherServiceServer
	service weatherGetterService
}

func NewWeatherGRPCServer(service weatherGetterService) *WeatherGRPCServer {
	return &WeatherGRPCServer{service: service}
}

func (s *WeatherGRPCServer) GetByCity(
	ctx context.Context,
	req *weatherpb.WeatherRequest,
) (*weatherpb.WeatherResponse, error) {
	data, err := s.service.GetByCity(ctx, req.City)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "weather fetch error: %v", err)
	}
	return &weatherpb.WeatherResponse{
		City:        data.City,
		Temperature: data.Temperature,
		Condition:   data.Condition,
	}, nil
}
