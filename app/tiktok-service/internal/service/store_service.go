package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/response"
)

type StoreService interface {
	AuthorizeURL(creds client.Credentials, state string) *response.AuthorizeURLResponse
	Connect(ctx context.Context, creds client.Credentials, tenantID uuid.UUID, req request.ConnectStoreRequest, actor *uuid.UUID) (*response.TokenConnectResponse, error)
	Refresh(ctx context.Context, creds client.Credentials, req request.RefreshStoreRequest) (*response.TokenConnectResponse, error)
}

type storeServiceImpl struct {
	logger *logrus.Logger
	client client.TikTokClient
}

var _ StoreService = (*storeServiceImpl)(nil)

func NewStoreService(logger *logrus.Logger, tiktokClient client.TikTokClient) StoreService {
	return &storeServiceImpl{logger: logger, client: tiktokClient}
}

func (s *storeServiceImpl) AuthorizeURL(creds client.Credentials, state string) *response.AuthorizeURLResponse {
	return &response.AuthorizeURLResponse{URL: s.client.BuildAuthorizeURL(creds, state)}
}

func (s *storeServiceImpl) Connect(ctx context.Context, creds client.Credentials, tenantID uuid.UUID, req request.ConnectStoreRequest, actor *uuid.UUID) (*response.TokenConnectResponse, error) {
	_ = tenantID
	_ = actor

	tokens, err := s.client.ExchangeToken(ctx, creds, req.Code)
	if err != nil {
		if errors.Is(err, client.ErrInvalidCode) {
			return nil, errors.New("invalid authorization code")
		}
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("tiktok rate limit exceeded")
		}
		s.logger.WithError(err).Error("exchange token")
		return nil, errors.New("failed to connect store")
	}

	sellerID := tokens.ShopID
	// Prefer concrete shop id from authorized shops when available.
	if shop, shopErr := s.client.ResolveShopCipher(ctx, creds, tokens.AccessToken, ""); shopErr == nil && shop != nil && shop.ID != "" {
		sellerID = shop.ID
		if tokens.Country == "" && shop.Region != "" {
			tokens.Country = shop.Region
		}
		if tokens.Account == "" && shop.Name != "" {
			tokens.Account = shop.Name
		}
	}
	if sellerID == "" {
		return nil, errors.New("tiktok shop id not found in token response")
	}

	return toTokenResponse(tokens, sellerID, req.DisplayName), nil
}

func (s *storeServiceImpl) Refresh(ctx context.Context, creds client.Credentials, req request.RefreshStoreRequest) (*response.TokenConnectResponse, error) {
	tokens, err := s.client.RefreshToken(ctx, creds, req.RefreshToken)
	if err != nil {
		if errors.Is(err, client.ErrInvalidToken) {
			return nil, errors.New("refresh token expired, reconnect store")
		}
		if errors.Is(err, client.ErrRateLimited) {
			return nil, errors.New("tiktok rate limit exceeded")
		}
		s.logger.WithError(err).Error("refresh token")
		return nil, errors.New("failed to refresh store tokens")
	}

	sellerID := tokens.ShopID
	if shop, shopErr := s.client.ResolveShopCipher(ctx, creds, tokens.AccessToken, ""); shopErr == nil && shop != nil && shop.ID != "" {
		sellerID = shop.ID
	}
	return toTokenResponse(tokens, sellerID, ""), nil
}

func toTokenResponse(tokens *client.TokenPair, sellerID, preferredDisplayName string) *response.TokenConnectResponse {
	now := time.Now()
	displayName := preferredDisplayName
	if displayName == "" {
		if tokens.Account != "" {
			displayName = tokens.Account
		} else {
			displayName = "TikTok Shop " + sellerID
		}
	}
	country := tokens.Country
	if country == "" {
		country = "US"
	}

	return &response.TokenConnectResponse{
		SellerID:         sellerID,
		DisplayName:      displayName,
		Country:          country,
		Account:          tokens.Account,
		ShortCode:        tokens.ShortCode,
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
