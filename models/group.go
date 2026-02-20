package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Group struct {
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey" json:"id"`
	Name      string        `gorm:"not null;size:100" json:"name"`
	Type      string        `gorm:"default:other;size:20" json:"type"` // home, trip, couple, other
	ImageURL  string        `json:"image_url,omitempty"`
	CreatedBy uuid.UUID     `gorm:"type:uuid" json:"created_by"`
	Creator   User          `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Members   []GroupMember `gorm:"foreignKey:GroupID" json:"members,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

func (g *Group) BeforeCreate(tx *gorm.DB) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	return nil
}

type GroupMember struct {
	GroupID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"group_id"`
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	User     User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Role     string    `gorm:"default:member;size:20" json:"role"` // admin, member
	JoinedAt time.Time `gorm:"autoCreateTime" json:"joined_at"`
}

// Request structs
type CreateGroupRequest struct {
	Name    string   `json:"name" binding:"required"`
	Type    string   `json:"type"`
	Members []string `json:"members"` // list of user IDs or emails
}

type AddMemberRequest struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Phone  string `json:"phone"`
}

// Response structs
type GroupResponse struct {
	ID        uuid.UUID            `json:"id"`
	Name      string               `json:"name"`
	Type      string               `json:"type"`
	ImageURL  string               `json:"image_url,omitempty"`
	CreatedBy uuid.UUID            `json:"created_by"`
	Members   []GroupMemberResponse `json:"members"`
	CreatedAt time.Time            `json:"created_at"`
}

type GroupMemberResponse struct {
	UserID   uuid.UUID `json:"user_id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	AvatarURL string   `json:"avatar_url,omitempty"`
	Role     string    `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
}
