package service

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/response"
)

type OrderService interface {
	List(ctx context.Context, accessToken, sellerID, updatedFrom string) ([]response.OrderResponse, error)
}

type orderServiceImpl struct {
	logger *logrus.Logger
	client client.LazadaClient
}

var _ OrderService = (*orderServiceImpl)(nil)

func NewOrderService(logger *logrus.Logger, lazadaClient client.LazadaClient) OrderService {
	return &orderServiceImpl{logger: logger, client: lazadaClient}
}

func (s *orderServiceImpl) List(ctx context.Context, accessToken, sellerID, updatedFrom string) ([]response.OrderResponse, error) {
	_ = ctx
	_ = accessToken
	_ = sellerID
	_ = updatedFrom
	return []response.OrderResponse{}, nil
}
