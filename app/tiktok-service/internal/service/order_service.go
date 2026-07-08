package service

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/response"
)

type OrderService interface {
	List(ctx context.Context, accessToken, sellerID, updatedFrom string) ([]response.OrderResponse, error)
}

type orderServiceImpl struct {
	logger *logrus.Logger
}

func NewOrderService(logger *logrus.Logger) OrderService {
	return &orderServiceImpl{logger: logger}
}

func (s *orderServiceImpl) List(ctx context.Context, accessToken, sellerID, updatedFrom string) ([]response.OrderResponse, error) {
	_ = ctx
	_ = accessToken
	_ = sellerID
	_ = updatedFrom
	return []response.OrderResponse{}, nil
}
