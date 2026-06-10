package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/client"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/mapper"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/models"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/repository"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/request"
	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/response"
)

type PostService interface {
	Create(ctx context.Context, tenantID uuid.UUID, req request.CreatePostRequest, actor *uuid.UUID) (*response.PostResponse, error)
	GetByID(ctx context.Context, tenantID, postID uuid.UUID) (*response.PostResponse, error)
	List(ctx context.Context, tenantID uuid.UUID, req request.ListPostsRequest) (*response.PagedResponse[response.PostResponse], error)
	Update(ctx context.Context, tenantID, postID uuid.UUID, req request.UpdatePostRequest, actor *uuid.UUID) (*response.PostResponse, error)
	Delete(ctx context.Context, tenantID, postID uuid.UUID, actor *uuid.UUID) error
}

type postServiceImpl struct {
	logger   *logrus.Logger
	channels repository.ChannelRepository
	posts    repository.PostRepository
	graph    client.GraphClient
}

var _ PostService = (*postServiceImpl)(nil)

func NewPostService(
	logger *logrus.Logger,
	channels repository.ChannelRepository,
	posts repository.PostRepository,
	graph client.GraphClient,
) PostService {
	return &postServiceImpl{logger: logger, channels: channels, posts: posts, graph: graph}
}

func (s *postServiceImpl) Create(ctx context.Context, tenantID uuid.UUID, req request.CreatePostRequest, actor *uuid.UUID) (*response.PostResponse, error) {
	ch, err := s.channels.GetByIDAndTenant(req.ChannelID, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("channel not found")
		}
		return nil, err
	}

	post := &models.Post{
		ChannelID: req.ChannelID,
		TenantID:  tenantID,
		Message:   req.Message,
		Status:    models.PostStatusDraft,
	}
	post.CreatedBy = actor
	post.UpdatedBy = actor

	if err := s.posts.Create(post); err != nil {
		s.logger.WithError(err).Error("create post")
		return nil, errors.New("failed to create post")
	}

	externalID, err := s.graph.CreatePost(ctx, ch.ExternalID, ch.AccessToken, req.Message)
	if err != nil {
		post.Status = models.PostStatusFailed
		post.ProviderError = err.Error()
		_ = s.posts.Update(post)
		return nil, mapGraphErr(err, "failed to publish post")
	}

	post.Status = models.PostStatusPublished
	post.ExternalID = externalID
	if err := s.posts.Update(post); err != nil {
		s.logger.WithError(err).Error("update post after publish")
		return nil, errors.New("failed to update post status")
	}

	res := mapper.ToPostResponse(*post)
	return &res, nil
}

func (s *postServiceImpl) GetByID(ctx context.Context, tenantID, postID uuid.UUID) (*response.PostResponse, error) {
	post, err := s.posts.GetByIDAndTenant(postID, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("post not found")
		}
		return nil, err
	}
	res := mapper.ToPostResponse(*post)
	return &res, nil
}

func (s *postServiceImpl) List(ctx context.Context, tenantID uuid.UUID, req request.ListPostsRequest) (*response.PagedResponse[response.PostResponse], error) {
	page, limit := req.Page, req.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	var channelID *uuid.UUID
	if req.ChannelID != "" {
		id, err := uuid.Parse(req.ChannelID)
		if err != nil {
			return nil, errors.New("invalid channel id")
		}
		channelID = &id
	}
	items, total, err := s.posts.ListByTenant(tenantID, channelID, page, limit)
	if err != nil {
		s.logger.WithError(err).Error("list posts")
		return nil, errors.New("failed to list posts")
	}
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}
	return &response.PagedResponse[response.PostResponse]{
		Items: mapper.ToPostResponses(items),
		Meta: response.PageMeta{Page: page, Limit: limit, Total: total, TotalPages: totalPages},
	}, nil
}

func (s *postServiceImpl) Update(ctx context.Context, tenantID, postID uuid.UUID, req request.UpdatePostRequest, actor *uuid.UUID) (*response.PostResponse, error) {
	post, err := s.posts.GetByIDAndTenant(postID, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("post not found")
		}
		return nil, err
	}
	if post.Status == models.PostStatusDeleted {
		return nil, errors.New("post already deleted")
	}
	if post.ExternalID == "" {
		return nil, errors.New("post is not published on facebook")
	}

	ch, err := s.channels.GetByIDAndTenant(post.ChannelID, tenantID)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	if err := s.graph.UpdatePost(ctx, post.ExternalID, ch.AccessToken, *req.Message); err != nil {
		return nil, mapGraphErr(err, "failed to update post")
	}

	post.Message = *req.Message
	post.UpdatedBy = actor
	if err := s.posts.Update(post); err != nil {
		s.logger.WithError(err).Error("update post")
		return nil, errors.New("failed to save post")
	}
	res := mapper.ToPostResponse(*post)
	return &res, nil
}

func (s *postServiceImpl) Delete(ctx context.Context, tenantID, postID uuid.UUID, actor *uuid.UUID) error {
	post, err := s.posts.GetByIDAndTenant(postID, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("post not found")
		}
		return err
	}
	if post.ExternalID != "" && post.Status != models.PostStatusDeleted {
		ch, err := s.channels.GetByIDAndTenant(post.ChannelID, tenantID)
		if err != nil {
			return errors.New("channel not found")
		}
		if err := s.graph.DeletePost(ctx, post.ExternalID, ch.AccessToken); err != nil {
			return mapGraphErr(err, "failed to delete post on facebook")
		}
	}
	post.Status = models.PostStatusDeleted
	post.UpdatedBy = actor
	post.DeletedBy = actor
	if err := s.posts.Update(post); err != nil {
		s.logger.WithError(err).Error("delete post")
		return errors.New("failed to delete post")
	}
	return nil
}

func mapGraphErr(err error, fallback string) error {
	if errors.Is(err, client.ErrRateLimited) {
		return errors.New("facebook rate limit exceeded")
	}
	if errors.Is(err, client.ErrInvalidToken) {
		return errors.New("invalid facebook page token")
	}
	return errors.New(fallback)
}
