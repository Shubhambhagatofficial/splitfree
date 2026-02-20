package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Activity struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GroupID     uuid.UUID `gorm:"type:uuid;index" json:"group_id"`
	GroupName   string    `gorm:"-" json:"group_name,omitempty"`
	UserID      uuid.UUID `gorm:"type:uuid" json:"user_id"`
	User        User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Type        string    `gorm:"not null;size:30" json:"type"` // expense_added, expense_updated, expense_deleted, settlement, member_joined, member_left
	ReferenceID uuid.UUID `gorm:"type:uuid" json:"reference_id,omitempty"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

func (a *Activity) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}
