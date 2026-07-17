package service

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/response"
)

type OrderService interface {
	List(ctx context.Context, req request.ListOrdersRequest) (*response.OrderListResponse, error)
}

type orderServiceImpl struct {
	logger *logrus.Logger
	client client.LazadaClient
}

var _ OrderService = (*orderServiceImpl)(nil)

func NewOrderService(logger *logrus.Logger, lazadaClient client.LazadaClient) OrderService {
	return &orderServiceImpl{logger: logger, client: lazadaClient}
}

func (s *orderServiceImpl) List(ctx context.Context, req request.ListOrdersRequest) (*response.OrderListResponse, error) {
	page := req.Page
	if page < 1 {
		page = 1
	}
	limit := req.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	result, err := s.client.ListOrders(ctx, req.AccessToken, page, limit, req.UpdatedFrom)
	if err != nil {
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("lazada rate limit exceeded")
		}
		s.logger.WithError(err).Error("list orders")
		return nil, errors.New("failed to list orders")
	}
	return result, nil
}
