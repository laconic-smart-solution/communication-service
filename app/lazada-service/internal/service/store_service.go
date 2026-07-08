package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/lazada-service/internal/response"
)

type StoreService interface {
	AuthorizeURL(state string) *response.AuthorizeURLResponse
	Connect(ctx context.Context, tenantID uuid.UUID, req request.ConnectStoreRequest, actor *uuid.UUID) (*response.TokenConnectResponse, error)
	Refresh(ctx context.Context, req request.RefreshStoreRequest) (*response.TokenConnectResponse, error)
}

type storeServiceImpl struct {
	logger *logrus.Logger
	client client.LazadaClient
}

var _ StoreService = (*storeServiceImpl)(nil)

func NewStoreService(logger *logrus.Logger, lazadaClient client.LazadaClient) StoreService {
	return &storeServiceImpl{logger: logger, client: lazadaClient}
}

func (s *storeServiceImpl) AuthorizeURL(state string) *response.AuthorizeURLResponse {
	return &response.AuthorizeURLResponse{URL: s.client.BuildAuthorizeURL(state)}
}

func (s *storeServiceImpl) Connect(ctx context.Context, tenantID uuid.UUID, req request.ConnectStoreRequest, actor *uuid.UUID) (*response.TokenConnectResponse, error) {
	_ = tenantID
	_ = actor

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

	return toTokenResponse(tokens, req.DisplayName), nil
}

func (s *storeServiceImpl) Refresh(ctx context.Context, req request.RefreshStoreRequest) (*response.TokenConnectResponse, error) {
	tokens, err := s.client.RefreshToken(ctx, req.RefreshToken)
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

	return toTokenResponse(tokens, ""), nil
}

func toTokenResponse(tokens *client.TokenPair, preferredDisplayName string) *response.TokenConnectResponse {
	now := time.Now()
	displayName := preferredDisplayName
	if displayName == "" {
		if tokens.ShortCode != "" {
			displayName = tokens.ShortCode
		} else {
			displayName = tokens.Account
		}
	}

	return &response.TokenConnectResponse{
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
}

func expiryFromSeconds(from time.Time, seconds int64) *time.Time {
	if seconds <= 0 {
		return nil
	}
	t := from.Add(time.Duration(seconds) * time.Second)
	return &t
}
