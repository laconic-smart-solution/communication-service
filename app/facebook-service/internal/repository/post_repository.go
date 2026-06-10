package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/models"
)

type PostRepository interface {
	Create(p *models.Post) error
	Update(p *models.Post) error
	GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Post, error)
	ListByTenant(tenantID uuid.UUID, channelID *uuid.UUID, page, limit int) ([]models.Post, int64, error)
}

type postRepository struct{ db *gorm.DB }

var _ PostRepository = (*postRepository)(nil)

func NewPostRepository(db *gorm.DB) PostRepository {
	return &postRepository{db: db}
}

func (r *postRepository) Create(p *models.Post) error {
	return r.db.Create(p).Error
}

func (r *postRepository) Update(p *models.Post) error {
	return r.db.Save(p).Error
}

func (r *postRepository) GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Post, error) {
	var p models.Post
	if err := r.db.First(&p, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *postRepository) ListByTenant(tenantID uuid.UUID, channelID *uuid.UUID, page, limit int) ([]models.Post, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	q := r.db.Model(&models.Post{}).Where("tenant_id = ?", tenantID)
	if channelID != nil {
		q = q.Where("channel_id = ?", *channelID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.Post
	if err := q.Order("created_at DESC").Offset((page-1)*limit).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
