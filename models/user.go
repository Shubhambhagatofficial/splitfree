package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Email        string    `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Phone        string    `gorm:"size:20" json:"phone,omitempty"`
	Name         string    `gorm:"not null;size:100" json:"name"`
	PasswordHash string    `gorm:"not null;size:255" json:"-"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	FCMToken     string    `json:"-"`
	Currency     string    `gorm:"default:INR;size:3" json:"currency"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// Response struct (what we return to clients)
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone,omitempty"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Phone:     u.Phone,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		Currency:  u.Currency,
		CreatedAt: u.CreatedAt,
	}
}
