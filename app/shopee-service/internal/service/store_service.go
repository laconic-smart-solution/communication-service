package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/shopee-service/internal/response"
)

type StoreService interface {
	AuthorizeURL(state string) *response.AuthorizeURLResponse
	Connect(ctx context.Context, tenantID uuid.UUID, req request.ConnectStoreRequest, actor *uuid.UUID) (*response.TokenConnectResponse, error)
	Refresh(ctx context.Context, req request.RefreshStoreRequest) (*response.TokenConnectResponse, error)
}

type storeServiceImpl struct {
	logger *logrus.Logger
	client client.ShopeeClient
}

var _ StoreService = (*storeServiceImpl)(nil)

func NewStoreService(logger *logrus.Logger, shopeeClient client.ShopeeClient) StoreService {
	return &storeServiceImpl{logger: logger, client: shopeeClient}
}

func (s *storeServiceImpl) AuthorizeURL(state string) *response.AuthorizeURLResponse {
	return &response.AuthorizeURLResponse{URL: s.client.BuildAuthorizeURL(state)}
}

func (s *storeServiceImpl) Connect(ctx context.Context, tenantID uuid.UUID, req request.ConnectStoreRequest, actor *uuid.UUID) (*response.TokenConnectResponse, error) {
	_ = tenantID
	_ = actor

	tokens, err := s.client.ExchangeToken(ctx, req.Code, req.ShopID)
	if err != nil {
		if errors.Is(err, client.ErrInvalidCode) {
			return nil, errors.New("invalid authorization code")
		}
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("shopee rate limit exceeded")
		}
		s.logger.WithError(err).Error("exchange token")
		return nil, errors.New("failed to connect store")
	}

	sellerID := tokens.ShopID
	if sellerID == "" {
		sellerID = req.ShopID
	}
	if sellerID == "" {
		return nil, errors.New("shopee shop id not found in token response")
	}

	return toTokenResponse(tokens, sellerID, req.DisplayName), nil
}

func (s *storeServiceImpl) Refresh(ctx context.Context, req request.RefreshStoreRequest) (*response.TokenConnectResponse, error) {
	tokens, err := s.client.RefreshToken(ctx, req.RefreshToken, req.ShopID)
	if err != nil {
		if errors.Is(err, client.ErrInvalidToken) {
			return nil, errors.New("refresh token expired, reconnect store")
		}
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("shopee rate limit exceeded")
		}
		s.logger.WithError(err).Error("refresh token")
		return nil, errors.New("failed to refresh store tokens")
	}

	sellerID := tokens.ShopID
	if sellerID == "" {
		sellerID = req.ShopID
	}
	return toTokenResponse(tokens, sellerID, ""), nil
}

func toTokenResponse(tokens *client.TokenPair, sellerID, preferredDisplayName string) *response.TokenConnectResponse {
	now := time.Now()
	displayName := preferredDisplayName
	if displayName == "" {
		displayName = "Shopee Shop " + sellerID
	}
	country := tokens.Country
	if country == "" {
		country = "SG"
	}

	return &response.TokenConnectResponse{
		SellerID:         sellerID,
		DisplayName:      displayName,
		Country:          country,
		Account:          sellerID,
		ShortCode:        sellerID,
		AccessToken:      tokens.AccessToken,
		RefreshToken:     tokens.RefreshToken,
		TokenExpiresAt:   expiryFromSeconds(now, tokens.ExpiresIn),
		RefreshExpiresAt: expiryFromSeconds(now, tokens.RefreshExpiresIn),
	}
}

func expiryFromSeconds(from time.Time, seconds int64) *time.Time {
	if seconds <= 0 {
		return nil
	}
	t := from.Add(time.Duration(seconds) * time.Second)
	return &t
}
