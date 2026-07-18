package service

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/response"
)

type ProductService interface {
	List(ctx context.Context, creds client.Credentials, req request.ListProductsRequest) (*response.ProductListResponse, error)
	Get(ctx context.Context, creds client.Credentials, req request.GetProductRequest) (*response.ProductDetailResponse, error)
}

type productServiceImpl struct {
	logger *logrus.Logger
	client client.TikTokClient
}

var _ ProductService = (*productServiceImpl)(nil)

func NewProductService(logger *logrus.Logger, tiktokClient client.TikTokClient) ProductService {
	return &productServiceImpl{logger: logger, client: tiktokClient}
}

func (s *productServiceImpl) List(ctx context.Context, creds client.Credentials, req request.ListProductsRequest) (*response.ProductListResponse, error) {
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

	result, err := s.client.ListProducts(ctx, creds, req.AccessToken, shopID, page, limit, req.Search)
	if err != nil {
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("tiktok rate limit exceeded")
		}
		s.logger.WithError(err).Error("list products")
		return nil, errors.New("failed to list products")
	}
	if result == nil {
		return &response.ProductListResponse{Items: []response.ProductResponse{}, Total: 0}, nil
	}
	return result, nil
}

func (s *productServiceImpl) Get(ctx context.Context, creds client.Credentials, req request.GetProductRequest) (*response.ProductDetailResponse, error) {
	shopID := req.ShopID
	if shopID == "" {
		shopID = req.SellerID
	}

	result, err := s.client.GetProduct(ctx, creds, req.AccessToken, shopID, req.ExternalProductID)
	if err != nil {
		if errors.Is(err, client.ErrNotFound) {
			return nil, client.ErrNotFound
		}
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("tiktok rate limit exceeded")
		}
		s.logger.WithError(err).Error("get product")
		return nil, errors.New("failed to get product")
	}
	return result, nil
}
