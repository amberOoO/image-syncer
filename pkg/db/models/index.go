package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BasicUUIDObject struct {
	/* record info */
	ID        uuid.UUID `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
