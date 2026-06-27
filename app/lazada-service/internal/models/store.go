package models

import (
	"time"

	"github.com/google/uuid"
)

type Store struct {
	Base
	TenantID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	SellerID         string     `gorm:"type:varchar(128);not null" json:"seller_id"`
	DisplayName      string     `gorm:"type:varchar(255)" json:"display_name"`
	Country          string     `gorm:"type:varchar(8);not null" json:"country"`
	Account          string     `gorm:"type:varchar(255)" json:"account"`
	ShortCode        string     `gorm:"type:varchar(64)" json:"short_code"`
	AccessToken      string     `gorm:"type:text;not null" json:"-"`
	RefreshToken     string     `gorm:"type:text" json:"-"`
	TokenExpiresAt   *time.Time `json:"token_expires_at,omitempty"`
	RefreshExpiresAt *time.Time `json:"refresh_expires_at,omitempty"`
}

func (Store) TableName() string { return "lazada_stores" }
