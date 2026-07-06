package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/laconic-smart-solution/communication-service/app/tiktok-service/internal/models"
)

type StoreRepository interface {
	Create(store *models.Store) error
	Update(store *models.Store) error
	GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Store, error)
	GetByTenantSellerAndCountry(tenantID uuid.UUID, sellerID, country string) (*models.Store, error)
	ListByTenant(tenantID uuid.UUID, page, limit int) ([]models.Store, int64, error)
	DeleteByIDAndTenant(id, tenantID uuid.UUID, deletedBy *uuid.UUID) error
}

type storeRepository struct {
	db *gorm.DB
}

var _ StoreRepository = (*storeRepository)(nil)

func NewStoreRepository(db *gorm.DB) StoreRepository {
	return &storeRepository{db: db}
}

func (r *storeRepository) Create(store *models.Store) error {
	return r.db.Create(store).Error
}

func (r *storeRepository) Update(store *models.Store) error {
	return r.db.Save(store).Error
}

func (r *storeRepository) GetByIDAndTenant(id, tenantID uuid.UUID) (*models.Store, error) {
	var store models.Store
	if err := r.db.First(&store, "id = ? AND tenant_id = ?", id, tenantID).Error; err != nil {
		return nil, err
	}
	return &store, nil
}

func (r *storeRepository) GetByTenantSellerAndCountry(tenantID uuid.UUID, sellerID, country string) (*models.Store, error) {
	var store models.Store
	if err := r.db.First(&store, "tenant_id = ? AND seller_id = ? AND country = ?", tenantID, sellerID, country).Error; err != nil {
		return nil, err
	}
	return &store, nil
}

func (r *storeRepository) ListByTenant(tenantID uuid.UUID, page, limit int) ([]models.Store, int64, error) {
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
	q := r.db.Model(&models.Store{}).Where("tenant_id = ?", tenantID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []models.Store
	offset := (page - 1) * limit
	if err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *storeRepository) DeleteByIDAndTenant(id, tenantID uuid.UUID, deletedBy *uuid.UUID) error {
	res := r.db.Model(&models.Store{}).
		Where("id = ? AND tenant_id = ?", id, tenantID).
		Updates(map[string]interface{}{"deleted_by": deletedBy})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return r.db.Delete(&models.Store{}, "id = ? AND tenant_id = ?", id, tenantID).Error
}
