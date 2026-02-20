package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Invitation struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	GroupID   uuid.UUID `gorm:"type:uuid;index" json:"group_id"`
	Group     Group     `gorm:"foreignKey:GroupID" json:"group,omitempty"`
	InvitedBy uuid.UUID `gorm:"type:uuid" json:"invited_by"`
	Inviter   User      `gorm:"foreignKey:InvitedBy" json:"inviter,omitempty"`
	Email     string    `gorm:"size:255" json:"email,omitempty"`
	Phone     string    `gorm:"size:20" json:"phone,omitempty"`
	Status    string    `gorm:"default:pending;size:20" json:"status"` // pending, accepted, declined
	CreatedAt time.Time `json:"created_at"`
}

func (i *Invitation) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

type InviteRequest struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}
