package models

import (
	"github.com/google/uuid"
)

const (
	MessageDirectionInbound  = "inbound"
	MessageDirectionOutbound = "outbound"

	MessageStatusPending = "pending"
	MessageStatusSent    = "sent"
	MessageStatusFailed  = "failed"
)

type Message struct {
	Base
	ChannelID     uuid.UUID `gorm:"type:uuid;not null;index" json:"channel_id"`
	TenantID      uuid.UUID `gorm:"type:uuid;not null;index" json:"tenant_id"`
	Direction     string    `gorm:"type:varchar(16);not null" json:"direction"`
	RecipientID   string    `gorm:"type:varchar(128);not null" json:"recipient_id"`
	Body          string    `gorm:"type:text" json:"body"`
	Status        string    `gorm:"type:varchar(32);not null;index" json:"status"`
	ExternalID    string    `gorm:"type:varchar(128);index" json:"external_id,omitempty"`
	ProviderError string    `gorm:"type:text" json:"-"`
}

func (Message) TableName() string { return "line_messages" }
