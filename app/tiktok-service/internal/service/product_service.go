package service

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/response"
)

type ProductService interface {
	List(ctx context.Context, req request.ListProductsRequest) (*response.ProductListResponse, error)
}

type productServiceImpl struct {
	logger *logrus.Logger
}

func NewProductService(logger *logrus.Logger) ProductService {
	return &productServiceImpl{logger: logger}
}

func (s *productServiceImpl) List(ctx context.Context, req request.ListProductsRequest) (*response.ProductListResponse, error) {
	_ = ctx
	_ = req
	s.logger.Debug("tiktok product list not implemented yet")
	return &response.ProductListResponse{Items: []response.ProductResponse{}, Total: 0}, nil
}
