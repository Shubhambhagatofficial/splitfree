package handlers

import (
	"net/http"
	"splitwise-backend/database"
	"splitwise-backend/models"
	"splitwise-backend/utils"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type RegisterRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password" binding:"required,min=6"`
	Currency string `json:"currency"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string              `json:"token"`
	User  models.UserResponse `json:"user"`
}

// POST /auth/register
func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// Check if email already exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		utils.BadRequest(c, "Email already registered")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		utils.InternalError(c, "Failed to hash password")
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "INR"
	}

	user := models.User{
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: string(hashedPassword),
		Currency:     currency,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		utils.InternalError(c, "Failed to create user")
		return
	}

	// Check if this user has any pending invitations and auto-accept them
	go acceptPendingInvitations(user)

	// Generate JWT
	token, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		utils.InternalError(c, "Failed to generate token")
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "Registration successful", AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}

// POST /auth/login
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	if err := database.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		utils.Unauthorized(c, "Invalid email or password")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		utils.Unauthorized(c, "Invalid email or password")
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Email)
	if err != nil {
		utils.InternalError(c, "Failed to generate token")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Login successful", AuthResponse{
		Token: token,
		User:  user.ToResponse(),
	})
}

// Auto-accept pending invitations when a user registers
func acceptPendingInvitations(user models.User) {
	var invitations []models.Invitation
	database.DB.Where("(email = ? OR phone = ?) AND status = ?", user.Email, user.Phone, "pending").Find(&invitations)

	for _, inv := range invitations {
		// Add user to the group
		member := models.GroupMember{
			GroupID: inv.GroupID,
			UserID:  user.ID,
			Role:    "member",
		}
		database.DB.Create(&member)

		// Update invitation status
		database.DB.Model(&inv).Update("status", "accepted")

		// Log activity
		var group models.Group
		database.DB.First(&group, inv.GroupID)
		database.DB.Create(&models.Activity{
			GroupID:     inv.GroupID,
			UserID:      user.ID,
			Type:        "member_joined",
			Description: user.Name + " joined " + group.Name,
		})
	}
}
