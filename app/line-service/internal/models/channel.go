package models

import (
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	Base
	TenantID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"tenant_id"`
	ExternalID     string     `gorm:"type:varchar(128);not null" json:"external_id"`
	DisplayName    string     `gorm:"type:varchar(255)" json:"display_name"`
	AccessToken    string     `gorm:"type:text;not null" json:"-"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
}

func (Channel) TableName() string { return "line_channels" }
