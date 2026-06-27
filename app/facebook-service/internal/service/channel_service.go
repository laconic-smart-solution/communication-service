package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/mapper"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/repository"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/response"
	pkgresponse "github.com/laconic-smart-solution/communication-service/pkg/response"
)

type ChannelService interface {
	Connect(ctx context.Context, tenantID uuid.UUID, req request.ConnectChannelRequest, actor *uuid.UUID) (*response.ChannelResponse, error)
	List(ctx context.Context, tenantID uuid.UUID, req request.ListChannelsRequest) (*pkgresponse.PagedResponse[response.ChannelResponse], error)
	Revoke(ctx context.Context, tenantID, channelID uuid.UUID, actor *uuid.UUID) error
}

type channelServiceImpl struct {
	logger *logrus.Logger
	repo   repository.ChannelRepository
}

var _ ChannelService = (*channelServiceImpl)(nil)

func NewChannelService(logger *logrus.Logger, repo repository.ChannelRepository) ChannelService {
	return &channelServiceImpl{logger: logger, repo: repo}
}

func (s *channelServiceImpl) Connect(ctx context.Context, tenantID uuid.UUID, req request.ConnectChannelRequest, actor *uuid.UUID) (*response.ChannelResponse, error) {
	ch := &models.Channel{
		TenantID:    tenantID,
		ExternalID:  req.ExternalID,
		DisplayName: req.DisplayName,
		AccessToken: req.AccessToken,
	}
	ch.CreatedBy = actor
	ch.UpdatedBy = actor
	if err := s.repo.Create(ch); err != nil {
		s.logger.WithError(err).Error("connect channel")
		return nil, errors.New("failed to connect channel")
	}
	res := mapper.ToChannelResponse(*ch)
	return &res, nil
}

func (s *channelServiceImpl) List(ctx context.Context, tenantID uuid.UUID, req request.ListChannelsRequest) (*pkgresponse.PagedResponse[response.ChannelResponse], error) {
	page, limit := req.Page, req.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	items, total, err := s.repo.ListByTenant(tenantID, page, limit)
	if err != nil {
		s.logger.WithError(err).Error("list channels")
		return nil, errors.New("failed to list channels")
	}
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}
	return &pkgresponse.PagedResponse[response.ChannelResponse]{
		Items: mapper.ToChannelResponses(items),
		Meta:  pkgresponse.PageMeta{Page: page, Limit: limit, Total: total, TotalPages: totalPages},
	}, nil
}

func (s *channelServiceImpl) Revoke(ctx context.Context, tenantID, channelID uuid.UUID, actor *uuid.UUID) error {
	if err := s.repo.DeleteByIDAndTenant(channelID, tenantID, actor); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("channel not found")
		}
		s.logger.WithError(err).Error("revoke channel")
		return errors.New("failed to revoke channel")
	}
	return nil
}
