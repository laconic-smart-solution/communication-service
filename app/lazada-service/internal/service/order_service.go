package service

import (
	"context"
	"errors"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/response"
)

type OrderService interface {
	List(ctx context.Context, creds client.Credentials, req request.ListOrdersRequest) (*response.OrderListResponse, error)
}

type orderServiceImpl struct {
	logger *logrus.Logger
	client client.LazadaClient
}

var _ OrderService = (*orderServiceImpl)(nil)

func NewOrderService(logger *logrus.Logger, lazadaClient client.LazadaClient) OrderService {
	return &orderServiceImpl{logger: logger, client: lazadaClient}
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

	result, err := s.client.ListOrders(ctx, creds, req.AccessToken, page, limit, req.UpdatedFrom)
	if err != nil {
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("lazada rate limit exceeded")
		}
		s.logger.WithError(err).Error("list orders")
		return nil, errors.New("failed to list orders")
	}

	if err := s.enrichOrderItems(ctx, creds, req.AccessToken, result); err != nil {
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("lazada rate limit exceeded")
		}
		s.logger.WithError(err).Warn("enrich order items; returning orders without items")
	}

	return result, nil
}

func (s *orderServiceImpl) enrichOrderItems(
	ctx context.Context,
	creds client.Credentials,
	accessToken string,
	result *response.OrderListResponse,
) error {
	if result == nil || len(result.Items) == 0 {
		return nil
	}

	orderIDs := make([]string, 0, len(result.Items))
	for _, order := range result.Items {
		if order.ExternalOrderID != "" {
			orderIDs = append(orderIDs, order.ExternalOrderID)
		}
	}
	if len(orderIDs) == 0 {
		return nil
	}

	itemsByOrder, err := s.client.GetMultipleOrderItems(ctx, creds, accessToken, orderIDs)
	if err != nil {
		s.logger.WithError(err).Warn("multi order items fetch failed; falling back to per-order")
		itemsByOrder = make(map[string][]response.OrderItemResponse)
	}

	enriched := 0
	for i := range result.Items {
		orderID := result.Items[i].ExternalOrderID
		items, ok := itemsByOrder[orderID]
		if !ok || len(items) == 0 {
			fallback, fallbackErr := s.client.GetOrderItems(ctx, creds, accessToken, orderID)
			if fallbackErr != nil {
				s.logger.WithError(fallbackErr).WithField("order_id", orderID).Warn("order items fallback failed")
				result.Items[i].Items = []response.OrderItemResponse{}
				continue
			}
			items = fallback
		}
		if items == nil {
			items = []response.OrderItemResponse{}
		}
		result.Items[i].Items = items
		if len(items) > 0 {
			enriched++
			if currency := currencyFromOrderItems(items); currency != "" {
				result.Items[i].Currency = currency
			}
			if buyerID := buyerIDFromOrderItems(items); buyerID != "" {
				result.Items[i].ExternalCustomerID = buyerID
			}
		}
	}

	s.logger.WithFields(logrus.Fields{
		"orders":   len(result.Items),
		"enriched": enriched,
	}).Info("order line-item enrichment complete")

	return nil
}

func currencyFromOrderItems(items []response.OrderItemResponse) string {
	for _, item := range items {
		if currency := strings.ToUpper(strings.TrimSpace(item.Currency)); currency != "" {
			return currency
		}
	}
	return ""
}

func buyerIDFromOrderItems(items []response.OrderItemResponse) string {
	for _, item := range items {
		if buyerID := strings.TrimSpace(item.BuyerID); buyerID != "" {
			return buyerID
		}
	}
	return ""
}
