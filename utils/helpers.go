package utils

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func SuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func ErrorResponse(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Message: message,
	})
}

func BadRequest(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusUnauthorized, message)
}

func NotFound(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusNotFound, message)
}

func InternalError(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusInternalServerError, message)
}

// Parse UUID from string
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// Get current user ID from context (set by auth middleware)
func GetCurrentUserID(c *gin.Context) uuid.UUID {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	return userID.(uuid.UUID)
}

// Round to 2 decimal places
func RoundToTwo(val float64) float64 {
	return math.Round(val*100) / 100
}

// Pagination helpers
type PaginationQuery struct {
	Page  int `form:"page,default=1"`
	Limit int `form:"limit,default=20"`
}

func (p *PaginationQuery) Offset() int {
	return (p.Page - 1) * p.Limit
}
