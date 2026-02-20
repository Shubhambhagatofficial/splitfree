package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Settlement struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GroupID   uuid.UUID `gorm:"type:uuid;index" json:"group_id"`
	PaidBy    uuid.UUID `gorm:"type:uuid" json:"paid_by"`
	Payer     User      `gorm:"foreignKey:PaidBy" json:"payer,omitempty"`
	PaidTo    uuid.UUID `gorm:"type:uuid" json:"paid_to"`
	Payee     User      `gorm:"foreignKey:PaidTo" json:"payee,omitempty"`
	Amount    float64   `gorm:"type:decimal(12,2);not null" json:"amount"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Settlement) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type CreateSettlementRequest struct {
	GroupID string  `json:"group_id" binding:"required"`
	PaidTo  string  `json:"paid_to" binding:"required"`
	Amount  float64 `json:"amount" binding:"required,gt=0"`
	Notes   string  `json:"notes"`
}
