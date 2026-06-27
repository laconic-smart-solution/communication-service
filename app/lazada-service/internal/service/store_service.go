package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/mapper"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/repository"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/response"
	pkgresponse "github.com/laconic-smart-solution/communication-service/pkg/response"
)

type StoreService interface {
	AuthorizeURL(state string) *response.AuthorizeURLResponse
	Connect(ctx context.Context, tenantID uuid.UUID, req request.ConnectStoreRequest, actor *uuid.UUID) (*response.StoreResponse, error)
	List(ctx context.Context, tenantID uuid.UUID, req request.ListStoresRequest) (*pkgresponse.PagedResponse[response.StoreResponse], error)
	Revoke(ctx context.Context, tenantID, storeID uuid.UUID, actor *uuid.UUID) error
	Refresh(ctx context.Context, tenantID, storeID uuid.UUID, actor *uuid.UUID) (*response.StoreResponse, error)
}

type storeServiceImpl struct {
	logger *logrus.Logger
	repo   repository.StoreRepository
	client client.LazadaClient
}

var _ StoreService = (*storeServiceImpl)(nil)

func NewStoreService(logger *logrus.Logger, repo repository.StoreRepository, lazadaClient client.LazadaClient) StoreService {
	return &storeServiceImpl{logger: logger, repo: repo, client: lazadaClient}
}

func (s *storeServiceImpl) AuthorizeURL(state string) *response.AuthorizeURLResponse {
	return &response.AuthorizeURLResponse{URL: s.client.BuildAuthorizeURL(state)}
}

func (s *storeServiceImpl) Connect(ctx context.Context, tenantID uuid.UUID, req request.ConnectStoreRequest, actor *uuid.UUID) (*response.StoreResponse, error) {
	tokens, err := s.client.ExchangeToken(ctx, req.Code)
	if err != nil {
		if errors.Is(err, client.ErrInvalidCode) {
			return nil, errors.New("invalid authorization code")
		}
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("lazada rate limit exceeded")
		}
		s.logger.WithError(err).Error("exchange token")
		return nil, errors.New("failed to connect store")
	}

	if tokens.SellerID == "" {
		return nil, errors.New("lazada seller id not found in token response")
	}

	existing, err := s.repo.GetByTenantSellerAndCountry(tenantID, tokens.SellerID, tokens.Country)
	if err == nil && existing != nil {
		return nil, errors.New("store already connected")
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.WithError(err).Error("check duplicate store")
		return nil, errors.New("failed to connect store")
	}

	now := time.Now()
	displayName := req.DisplayName
	if displayName == "" {
		if tokens.ShortCode != "" {
			displayName = tokens.ShortCode
		} else {
			displayName = tokens.Account
		}
	}

	store := &models.Store{
		TenantID:         tenantID,
		SellerID:         tokens.SellerID,
		DisplayName:      displayName,
		Country:          tokens.Country,
		Account:          tokens.Account,
		ShortCode:        tokens.ShortCode,
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		TokenExpiresAt:   expiryFromSeconds(now, tokens.ExpiresIn),
		RefreshExpiresAt: expiryFromSeconds(now, tokens.RefreshExpiresIn),
	}
	store.CreatedBy = actor
	store.UpdatedBy = actor

	if err := s.repo.Create(store); err != nil {
		s.logger.WithError(err).Error("connect store")
		return nil, errors.New("failed to connect store")
	}

	res := mapper.ToStoreResponse(*store)
	return &res, nil
}

func (s *storeServiceImpl) List(ctx context.Context, tenantID uuid.UUID, req request.ListStoresRequest) (*pkgresponse.PagedResponse[response.StoreResponse], error) {
	page, limit := req.Page, req.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	items, total, err := s.repo.ListByTenant(tenantID, page, limit)
	if err != nil {
		s.logger.WithError(err).Error("list stores")
		return nil, errors.New("failed to list stores")
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &pkgresponse.PagedResponse[response.StoreResponse]{
		Items: mapper.ToStoreResponses(items),
		Meta: pkgresponse.PageMeta{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
		},
	}, nil
}

func (s *storeServiceImpl) Revoke(ctx context.Context, tenantID, storeID uuid.UUID, actor *uuid.UUID) error {
	if err := s.repo.DeleteByIDAndTenant(storeID, tenantID, actor); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("store not found")
		}
		s.logger.WithError(err).Error("revoke store")
		return errors.New("failed to revoke store")
	}
	return nil
}

func (s *storeServiceImpl) Refresh(ctx context.Context, tenantID, storeID uuid.UUID, actor *uuid.UUID) (*response.StoreResponse, error) {
	store, err := s.repo.GetByIDAndTenant(storeID, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("store not found")
		}
		s.logger.WithError(err).Error("get store for refresh")
		return nil, errors.New("failed to refresh store tokens")
	}

	if store.RefreshToken == "" {
		return nil, errors.New("store has no refresh token")
	}

	tokens, err := s.client.RefreshToken(ctx, store.RefreshToken)
	if err != nil {
		if errors.Is(err, client.ErrInvalidToken) {
			return nil, errors.New("refresh token expired, reconnect store")
		}
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("lazada rate limit exceeded")
		}
		s.logger.WithError(err).Error("refresh token")
		return nil, errors.New("failed to refresh store tokens")
	}

	now := time.Now()
	store.AccessToken = tokens.AccessToken
	store.RefreshToken = tokens.RefreshToken
	store.TokenExpiresAt = expiryFromSeconds(now, tokens.ExpiresIn)
	store.RefreshExpiresAt = expiryFromSeconds(now, tokens.RefreshExpiresIn)
	if tokens.Account != "" {
		store.Account = tokens.Account
	}
	if tokens.ShortCode != "" {
		store.ShortCode = tokens.ShortCode
	}
	store.UpdatedBy = actor

	if err := s.repo.Update(store); err != nil {
		s.logger.WithError(err).Error("update store tokens")
		return nil, errors.New("failed to refresh store tokens")
	}

	res := mapper.ToStoreResponse(*store)
	return &res, nil
}

func expiryFromSeconds(from time.Time, seconds int64) *time.Time {
	if seconds <= 0 {
		return nil
	}
	t := from.Add(time.Duration(seconds) * time.Second)
	return &t
}
