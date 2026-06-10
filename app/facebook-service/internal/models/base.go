package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Base struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	CreatedBy *uuid.UUID     `gorm:"type:uuid"             json:"created_by,omitempty"`
	UpdatedAt time.Time      `json:"updated_at"`
	UpdatedBy *uuid.UUID     `gorm:"type:uuid"             json:"updated_by,omitempty"`
	IsEnable  bool           `gorm:"not null;default:true" json:"is_enable"`
	DeletedAt gorm.DeletedAt `gorm:"index"                 json:"-"`
	DeletedBy *uuid.UUID     `gorm:"type:uuid"             json:"deleted_by,omitempty"`
}
