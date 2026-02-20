package handlers

import (
	"net/http"
	"splitwise-backend/database"
	"splitwise-backend/models"
	"splitwise-backend/utils"

	"github.com/gin-gonic/gin"
)

type UpdateProfileRequest struct {
	Name      string `json:"name"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar_url"`
	Currency  string `json:"currency"`
}

type UpdateFCMTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// GET /api/users/me
func GetProfile(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", user.ToResponse())
}

// PUT /api/users/me
func UpdateProfile(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var user models.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		utils.NotFound(c, "User not found")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.AvatarURL != "" {
		updates["avatar_url"] = req.AvatarURL
	}
	if req.Currency != "" {
		updates["currency"] = req.Currency
	}

	database.DB.Model(&user).Updates(updates)

	utils.SuccessResponse(c, http.StatusOK, "Profile updated", user.ToResponse())
}

// PUT /api/users/me/fcm-token
func UpdateFCMToken(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)

	var req UpdateFCMTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	database.DB.Model(&models.User{}).Where("id = ?", userID).Update("fcm_token", req.Token)

	utils.SuccessResponse(c, http.StatusOK, "FCM token updated", nil)
}

// POST /api/users/search
func SearchUsers(c *gin.Context) {
	var req struct {
		Query string `json:"query" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var users []models.User
	database.DB.Where("email ILIKE ? OR name ILIKE ? OR phone ILIKE ?",
		"%"+req.Query+"%", "%"+req.Query+"%", "%"+req.Query+"%").
		Limit(20).
		Find(&users)

	var responses []models.UserResponse
	for _, u := range users {
		responses = append(responses, u.ToResponse())
	}

	utils.SuccessResponse(c, http.StatusOK, "", responses)
}
