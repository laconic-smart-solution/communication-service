package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/repository"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/response"
)

type OrderService interface {
	List(ctx context.Context, tenantID, storeID uuid.UUID, updatedFrom string) ([]response.OrderResponse, error)
}

type orderServiceImpl struct {
	logger *logrus.Logger
	repo   repository.StoreRepository
}

var _ OrderService = (*orderServiceImpl)(nil)

func NewOrderService(logger *logrus.Logger, repo repository.StoreRepository) OrderService {
	return &orderServiceImpl{logger: logger, repo: repo}
}

func (s *orderServiceImpl) List(ctx context.Context, tenantID, storeID uuid.UUID, updatedFrom string) ([]response.OrderResponse, error) {
	_, err := s.repo.GetByIDAndTenant(storeID, tenantID)
	if err != nil {
		return nil, errors.New("store not found")
	}

	_ = updatedFrom
	return []response.OrderResponse{}, nil
}
