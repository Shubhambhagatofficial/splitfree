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

// POST /api/groups
func CreateGroup(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)

	var req models.CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	groupType := req.Type
	if groupType == "" {
		groupType = "other"
	}

	group := models.Group{
		Name:      req.Name,
		Type:      groupType,
		CreatedBy: userID,
	}

	if err := database.DB.Create(&group).Error; err != nil {
		utils.InternalError(c, "Failed to create group")
		return
	}

	// Add creator as admin member
	member := models.GroupMember{
		GroupID: group.ID,
		UserID:  userID,
		Role:    "admin",
	}
	database.DB.Create(&member)

	// Add other members if provided
	for _, memberInput := range req.Members {
		memberUUID, err := uuid.Parse(memberInput)
		if err != nil {
			// Might be an email, try to find user
			var user models.User
			if dbErr := database.DB.Where("email = ?", memberInput).First(&user).Error; dbErr == nil {
				memberUUID = user.ID
			} else {
				// Send invitation
				go services.InviteToGroup(group.ID, userID, memberInput, "")
				continue
			}
		}

		if memberUUID != userID {
			database.DB.Create(&models.GroupMember{
				GroupID: group.ID,
				UserID:  memberUUID,
				Role:    "member",
			})
		}
	}

	// Log activity
	var creator models.User
	database.DB.First(&creator, userID)
	database.DB.Create(&models.Activity{
		GroupID:     group.ID,
		UserID:      userID,
		Type:        "group_created",
		ReferenceID: group.ID,
		Description: fmt.Sprintf("%s created group \"%s\"", creator.Name, group.Name),
	})

	// Return group with members
	response := buildGroupResponse(group.ID)
	utils.SuccessResponse(c, http.StatusCreated, "Group created", response)
}

// GET /api/groups
func GetGroups(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)

	var memberships []models.GroupMember
	database.DB.Where("user_id = ?", userID).Find(&memberships)

	var groupIDs []uuid.UUID
	for _, m := range memberships {
		groupIDs = append(groupIDs, m.GroupID)
	}

	var groups []models.Group
	if len(groupIDs) > 0 {
		database.DB.Where("id IN ?", groupIDs).Order("created_at DESC").Find(&groups)
	}

	var responses []models.GroupResponse
	for _, g := range groups {
		responses = append(responses, buildGroupResponse(g.ID))
	}

	utils.SuccessResponse(c, http.StatusOK, "", responses)
}

// GET /api/groups/:id
func GetGroup(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "Invalid group ID")
		return
	}

	// Check membership
	if !isMember(groupID, userID) {
		utils.Unauthorized(c, "You are not a member of this group")
		return
	}

	response := buildGroupResponse(groupID)
	utils.SuccessResponse(c, http.StatusOK, "", response)
}

// PUT /api/groups/:id
func UpdateGroup(c *gin.Context) {
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

	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		ImageURL string `json:"image_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Type != "" {
		updates["type"] = req.Type
	}
	if req.ImageURL != "" {
		updates["image_url"] = req.ImageURL
	}

	database.DB.Model(&models.Group{}).Where("id = ?", groupID).Updates(updates)

	response := buildGroupResponse(groupID)
	utils.SuccessResponse(c, http.StatusOK, "Group updated", response)
}

// POST /api/groups/:id/members
func AddMember(c *gin.Context) {
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

	var req models.AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	var targetUser models.User
	found := false

	if req.UserID != "" {
		memberUUID, _ := uuid.Parse(req.UserID)
		if err := database.DB.First(&targetUser, memberUUID).Error; err == nil {
			found = true
		}
	}

	if !found && req.Email != "" {
		if err := database.DB.Where("email = ?", req.Email).First(&targetUser).Error; err == nil {
			found = true
		}
	}

	if !found && req.Phone != "" {
		if err := database.DB.Where("phone = ?", req.Phone).First(&targetUser).Error; err == nil {
			found = true
		}
	}

	if found {
		// Check if already a member
		var existing models.GroupMember
		if err := database.DB.Where("group_id = ? AND user_id = ?", groupID, targetUser.ID).First(&existing).Error; err == nil {
			utils.BadRequest(c, "User is already a member of this group")
			return
		}

		database.DB.Create(&models.GroupMember{
			GroupID: groupID,
			UserID:  targetUser.ID,
			Role:    "member",
		})

		// Log activity and notify
		var adder models.User
		database.DB.First(&adder, userID)
		var group models.Group
		database.DB.First(&group, groupID)

		database.DB.Create(&models.Activity{
			GroupID:     groupID,
			UserID:      userID,
			Type:        "member_joined",
			Description: fmt.Sprintf("%s added %s to %s", adder.Name, targetUser.Name, group.Name),
		})

		go services.GetNotificationService().NotifyMemberAdded(group, adder, targetUser)

		utils.SuccessResponse(c, http.StatusOK, "Member added", targetUser.ToResponse())
	} else {
		// User not registered — send invitation
		email := req.Email
		phone := req.Phone
		go services.InviteToGroup(groupID, userID, email, phone)
		utils.SuccessResponse(c, http.StatusOK, "Invitation sent", nil)
	}
}

// DELETE /api/groups/:id/members/:uid
func RemoveMember(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "Invalid group ID")
		return
	}

	memberUID, err := uuid.Parse(c.Param("uid"))
	if err != nil {
		utils.BadRequest(c, "Invalid user ID")
		return
	}

	// Only admin or self can remove
	var membership models.GroupMember
	database.DB.Where("group_id = ? AND user_id = ?", groupID, userID).First(&membership)
	if membership.Role != "admin" && userID != memberUID {
		utils.Unauthorized(c, "Only admins can remove other members")
		return
	}

	database.DB.Where("group_id = ? AND user_id = ?", groupID, memberUID).Delete(&models.GroupMember{})

	var removedUser models.User
	database.DB.First(&removedUser, memberUID)
	var group models.Group
	database.DB.First(&group, groupID)

	database.DB.Create(&models.Activity{
		GroupID:     groupID,
		UserID:      userID,
		Type:        "member_left",
		Description: fmt.Sprintf("%s left %s", removedUser.Name, group.Name),
	})

	utils.SuccessResponse(c, http.StatusOK, "Member removed", nil)
}

// POST /api/groups/:id/invite
func InviteToGroupHandler(c *gin.Context) {
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

	var req models.InviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	if req.Email == "" && req.Phone == "" {
		utils.BadRequest(c, "Email or phone required")
		return
	}

	go services.InviteToGroup(groupID, userID, req.Email, req.Phone)

	utils.SuccessResponse(c, http.StatusOK, "Invitation sent", nil)
}

// Helper: check group membership
func isMember(groupID, userID uuid.UUID) bool {
	var count int64
	database.DB.Model(&models.GroupMember{}).Where("group_id = ? AND user_id = ?", groupID, userID).Count(&count)
	return count > 0
}

// Helper: build full group response with members
func buildGroupResponse(groupID uuid.UUID) models.GroupResponse {
	var group models.Group
	database.DB.First(&group, groupID)

	var members []models.GroupMember
	database.DB.Where("group_id = ?", groupID).Find(&members)

	var memberResponses []models.GroupMemberResponse
	for _, m := range members {
		var user models.User
		database.DB.First(&user, m.UserID)
		memberResponses = append(memberResponses, models.GroupMemberResponse{
			UserID:    user.ID,
			Name:      user.Name,
			Email:     user.Email,
			AvatarURL: user.AvatarURL,
			Role:      m.Role,
			JoinedAt:  m.JoinedAt,
		})
	}

	return models.GroupResponse{
		ID:        group.ID,
		Name:      group.Name,
		Type:      group.Type,
		ImageURL:  group.ImageURL,
		CreatedBy: group.CreatedBy,
		Members:   memberResponses,
		CreatedAt: group.CreatedAt,
	}
}
