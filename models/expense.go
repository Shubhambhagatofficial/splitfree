package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Expense struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	GroupID     uuid.UUID      `gorm:"type:uuid;index" json:"group_id"`
	Group       Group          `gorm:"foreignKey:GroupID" json:"-"`
	PaidBy      uuid.UUID      `gorm:"type:uuid" json:"paid_by"`
	Payer       User           `gorm:"foreignKey:PaidBy" json:"payer,omitempty"`
	Description string         `gorm:"not null;size:255" json:"description"`
	Amount      float64        `gorm:"type:decimal(12,2);not null" json:"amount"`
	Currency    string         `gorm:"default:INR;size:3" json:"currency"`
	Category    string         `gorm:"size:50" json:"category"` // food, transport, rent, utilities, entertainment, other
	SplitType   string         `gorm:"not null;size:20" json:"split_type"` // equal, exact, percentage, shares
	ReceiptURL  string         `json:"receipt_url,omitempty"`
	Notes       string         `json:"notes,omitempty"`
	ExpenseDate time.Time      `gorm:"type:date;default:CURRENT_DATE" json:"expense_date"`
	Splits      []ExpenseSplit `gorm:"foreignKey:ExpenseID" json:"splits,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (e *Expense) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

type ExpenseSplit struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ExpenseID  uuid.UUID `gorm:"type:uuid;index" json:"expense_id"`
	UserID     uuid.UUID `gorm:"type:uuid" json:"user_id"`
	User       User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	OwedAmount float64   `gorm:"type:decimal(12,2);not null" json:"owed_amount"`
	PaidAmount float64   `gorm:"type:decimal(12,2);default:0" json:"paid_amount"`
	CreatedAt  time.Time `json:"created_at"`
}

func (es *ExpenseSplit) BeforeCreate(tx *gorm.DB) error {
	if es.ID == uuid.Nil {
		es.ID = uuid.New()
	}
	return nil
}

// Request structs
type CreateExpenseRequest struct {
	GroupID     string        `json:"group_id" binding:"required"`
	Description string        `json:"description" binding:"required"`
	Amount      float64       `json:"amount" binding:"required,gt=0"`
	Currency    string        `json:"currency"`
	Category    string        `json:"category"`
	SplitType   string        `json:"split_type" binding:"required,oneof=equal exact percentage shares"`
	Notes       string        `json:"notes"`
	ExpenseDate string        `json:"expense_date"` // YYYY-MM-DD
	Splits      []SplitInput  `json:"splits"`       // required for exact, percentage, shares
}

type SplitInput struct {
	UserID string  `json:"user_id" binding:"required"`
	Value  float64 `json:"value"` // exact amount, percentage, or share count
}

type UpdateExpenseRequest struct {
	Description string       `json:"description"`
	Amount      float64      `json:"amount"`
	Category    string       `json:"category"`
	SplitType   string       `json:"split_type"`
	Notes       string       `json:"notes"`
	Splits      []SplitInput `json:"splits"`
}

// Response
type ExpenseResponse struct {
	ID          uuid.UUID            `json:"id"`
	GroupID     uuid.UUID            `json:"group_id"`
	PaidBy      uuid.UUID            `json:"paid_by"`
	PayerName   string               `json:"payer_name"`
	Description string               `json:"description"`
	Amount      float64              `json:"amount"`
	Currency    string               `json:"currency"`
	Category    string               `json:"category"`
	SplitType   string               `json:"split_type"`
	Notes       string               `json:"notes,omitempty"`
	ExpenseDate time.Time            `json:"expense_date"`
	Splits      []SplitResponse      `json:"splits"`
	CreatedAt   time.Time            `json:"created_at"`
}

type SplitResponse struct {
	UserID     uuid.UUID `json:"user_id"`
	UserName   string    `json:"user_name"`
	OwedAmount float64   `json:"owed_amount"`
	PaidAmount float64   `json:"paid_amount"`
}
