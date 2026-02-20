package models

import "github.com/google/uuid"

// Balance represents a simplified debt between two users
type Balance struct {
	From       uuid.UUID `json:"from"`
	FromName   string    `json:"from_name"`
	To         uuid.UUID `json:"to"`
	ToName     string    `json:"to_name"`
	Amount     float64   `json:"amount"`
	Currency   string    `json:"currency"`
}

// FriendBalance represents the overall balance with a single friend
type FriendBalance struct {
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	AvatarURL string    `json:"avatar_url,omitempty"`
	Amount    float64   `json:"amount"` // positive = they owe you, negative = you owe them
	Currency  string    `json:"currency"`
}

// GroupBalanceSummary is returned for GET /api/groups/:id/balances
type GroupBalanceSummary struct {
	GroupID    uuid.UUID  `json:"group_id"`
	GroupName  string     `json:"group_name"`
	Balances   []Balance  `json:"balances"`
	TotalSpent float64   `json:"total_spent"`
}

// OverallBalanceSummary is returned for GET /api/balances
type OverallBalanceSummary struct {
	TotalOwed    float64         `json:"total_owed"`    // total others owe you
	TotalOwing   float64         `json:"total_owing"`   // total you owe others
	Friends      []FriendBalance `json:"friends"`
}
