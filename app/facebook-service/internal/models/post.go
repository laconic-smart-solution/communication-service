package models

import "github.com/google/uuid"

const (
	PostStatusDraft     = "draft"
	PostStatusPublished = "published"
	PostStatusFailed    = "failed"
	PostStatusDeleted   = "deleted"
)

type Post struct {
	Base
	ChannelID     uuid.UUID `gorm:"type:uuid;not null;index" json:"channel_id"`
	TenantID      uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Message       string    `gorm:"type:text;not null" json:"message"`
	Status        string    `gorm:"type:varchar(32);not null;index" json:"status"`
	ExternalID    string    `gorm:"type:varchar(128);index" json:"external_id,omitempty"`
	ProviderError string    `gorm:"type:text" json:"-"`
}

func (Post) TableName() string { return "fb_posts" }
