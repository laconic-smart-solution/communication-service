package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/laconic-smart-solution/communication-service/app/facebook-service/internal/models"
)

type ChannelRepository interface {
	Create(ch *models.Channel) error
	GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Channel, error)
	ListByTenant(tenantID uuid.UUID, page, limit int) ([]models.Channel, int64, error)
	DeleteByIDAndTenant(id, tenantID uuid.UUID, deletedBy *uuid.UUID) error
}

type channelRepository struct{ db *gorm.DB }

var _ ChannelRepository = (*channelRepository)(nil)

func NewChannelRepository(db *gorm.DB) ChannelRepository {
	return &channelRepository{db: db}
}

func (r *channelRepository) Create(ch *models.Channel) error {
	return r.db.Create(ch).Error
}

func (r *channelRepository) GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Channel, error) {
	var ch models.Channel
	if err := r.db.First(&ch, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *channelRepository) ListByTenant(tenantID uuid.UUID, page, limit int) ([]models.Channel, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	var total int64
	q := r.db.Model(&models.Channel{}).Where("tenant_id = ?", tenantID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []models.Channel
	if err := q.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *channelRepository) DeleteByIDAndTenant(id, tenantID uuid.UUID, deletedBy *uuid.UUID) error {
	res := r.db.Model(&models.Channel{}).Where("id = ? AND tenant_id = ?", id, tenantID).
		Updates(map[string]interface{}{"deleted_by": deletedBy})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return r.db.Delete(&models.Channel{}, "id = ? AND tenant_id = ?", id, tenantID).Error
}
