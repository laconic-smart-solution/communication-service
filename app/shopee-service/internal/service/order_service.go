package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/repository"
	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/response"
)

type OrderService interface {
	List(ctx context.Context, tenantID, storeID uuid.UUID, updatedFrom string) ([]response.OrderResponse, error)
}

type orderServiceImpl struct {
	logger *logrus.Logger
	repo   repository.StoreRepository
	client client.ShopeeClient
}

var _ OrderService = (*orderServiceImpl)(nil)

func NewOrderService(logger *logrus.Logger, repo repository.StoreRepository, shopeeClient client.ShopeeClient) OrderService {
	return &orderServiceImpl{logger: logger, repo: repo, client: shopeeClient}
}

func (s *orderServiceImpl) List(ctx context.Context, tenantID, storeID uuid.UUID, updatedFrom string) ([]response.OrderResponse, error) {
	store, err := s.repo.GetByIDAndTenant(storeID, tenantID)
	if err != nil {
		return nil, errors.New("store not found")
	}

	_ = store
	_ = updatedFrom

	// Order list integration with Shopee Open API is vendor-specific; return empty until wired.
	return []response.OrderResponse{}, nil
}
