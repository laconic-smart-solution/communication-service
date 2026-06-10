package models

import "github.com/google/uuid"

type Channel struct {
	Base
	TenantID    uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	ExternalID  string    `gorm:"type:varchar(128);not null" json:"external_id"` // Facebook Page ID
	DisplayName string    `gorm:"type:varchar(255)" json:"display_name"`
	AccessToken string    `gorm:"type:text;not null" json:"-"`
}

func (Channel) TableName() string { return "fb_channels" }
