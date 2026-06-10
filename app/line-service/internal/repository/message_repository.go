package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/laconic-smart-solution/communication-service/app/line-service/internal/models"
)

type MessageRepository interface {
	Create(m *models.Message) error
	Update(m *models.Message) error
	GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

var _ MessageRepository = (*messageRepository)(nil)

func NewMessageRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(m *models.Message) error {
	return r.db.Create(m).Error
}

func (r *messageRepository) Update(m *models.Message) error {
	return r.db.Save(m).Error
}

func (r *messageRepository) GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Message, error) {
	var m models.Message
	if err := r.db.First(&m, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		return nil, err
	}
	return &m, nil
}
