package handlers

import (
	"net/http"
	"splitwise-backend/database"
	"splitwise-backend/models"
	"splitwise-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GET /api/activity — global activity feed for current user
func GetActivity(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)

	var pagination utils.PaginationQuery
	c.ShouldBindQuery(&pagination)

	// Get all groups user is in
	var memberships []models.GroupMember
	database.DB.Where("user_id = ?", userID).Find(&memberships)

	var groupIDs []uuid.UUID
	for _, m := range memberships {
		groupIDs = append(groupIDs, m.GroupID)
	}

	var activities []models.Activity
	if len(groupIDs) > 0 {
		database.DB.Where("group_id IN ?", groupIDs).
			Preload("User").
			Order("created_at DESC").
			Offset(pagination.Offset()).
			Limit(pagination.Limit).
			Find(&activities)

		// Attach group names
		groupNames := make(map[uuid.UUID]string)
		var groups []models.Group
		database.DB.Where("id IN ?", groupIDs).Find(&groups)
		for _, g := range groups {
			groupNames[g.ID] = g.Name
		}
		for i := range activities {
			activities[i].GroupName = groupNames[activities[i].GroupID]
		}
	}

	utils.SuccessResponse(c, http.StatusOK, "", activities)
}

// GET /api/groups/:id/activity — activity feed for a specific group
func GetGroupActivity(c *gin.Context) {
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

	var pagination utils.PaginationQuery
	c.ShouldBindQuery(&pagination)

	var activities []models.Activity
	database.DB.Where("group_id = ?", groupID).
		Preload("User").
		Order("created_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit).
		Find(&activities)

	utils.SuccessResponse(c, http.StatusOK, "", activities)
}
