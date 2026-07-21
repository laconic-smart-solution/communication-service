package service

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/response"
)

type OrderService interface {
	List(ctx context.Context, creds client.Credentials, req request.ListOrdersRequest) (*response.OrderListResponse, error)
}

type orderServiceImpl struct {
	logger *logrus.Logger
	client client.TikTokClient
}

var _ OrderService = (*orderServiceImpl)(nil)

func NewOrderService(logger *logrus.Logger, tiktokClient client.TikTokClient) OrderService {
	return &orderServiceImpl{logger: logger, client: tiktokClient}
}

func (s *orderServiceImpl) List(ctx context.Context, creds client.Credentials, req request.ListOrdersRequest) (*response.OrderListResponse, error) {
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

	shopID := req.ShopID
	if shopID == "" {
		shopID = req.SellerID
	}

	result, err := s.client.ListOrders(ctx, creds, req.AccessToken, shopID, page, limit, req.UpdatedFrom)
	if err != nil {
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("tiktok rate limit exceeded")
		}
		s.logger.WithError(err).Error("list orders")
		return nil, errors.New("failed to list orders")
	}
	if result == nil {
		return &response.OrderListResponse{Items: []response.OrderResponse{}, Total: 0}, nil
	}
	return result, nil
}
