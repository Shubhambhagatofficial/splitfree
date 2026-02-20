package handlers

import (
	"fmt"
	"net/http"
	"splitwise-backend/database"
	"splitwise-backend/models"
	"splitwise-backend/services"
	"splitwise-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// POST /api/groups/:id/settle
func CreateSettlement(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "Invalid group ID")
		return
	}

	if !isMember(groupID, userID) {
		utils.Unauthorized(c, "You are not a member of this group")
		return
	}

	var req models.CreateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	paidTo, err := uuid.Parse(req.PaidTo)
	if err != nil {
		utils.BadRequest(c, "Invalid paid_to user ID")
		return
	}

	settlement := models.Settlement{
		GroupID: groupID,
		PaidBy:  userID,
		PaidTo:  paidTo,
		Amount:  req.Amount,
		Notes:   req.Notes,
	}

	if err := database.DB.Create(&settlement).Error; err != nil {
		utils.InternalError(c, "Failed to create settlement")
		return
	}

	// Log activity
	var payer, payee models.User
	database.DB.First(&payer, userID)
	database.DB.First(&payee, paidTo)
	var group models.Group
	database.DB.First(&group, groupID)

	database.DB.Create(&models.Activity{
		GroupID:     groupID,
		UserID:      userID,
		Type:        "settlement",
		ReferenceID: settlement.ID,
		Description: fmt.Sprintf("%s paid %s %s %.2f", payer.Name, payee.Name, "INR", req.Amount),
	})

	// Notify the payee
	go services.GetNotificationService().NotifySettlement(settlement, payer, payee, group)

	utils.SuccessResponse(c, http.StatusCreated, "Settlement recorded", settlement)
}

// GET /api/groups/:id/settlements
func GetGroupSettlements(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "Invalid group ID")
		return
	}

	if !isMember(groupID, userID) {
		utils.Unauthorized(c, "You are not a member of this group")
		return
	}

	var settlements []models.Settlement
	database.DB.Where("group_id = ?", groupID).
		Preload("Payer").Preload("Payee").
		Order("created_at DESC").
		Find(&settlements)

	utils.SuccessResponse(c, http.StatusOK, "", settlements)
}
