package services

import (
	"log"
	"splitwise-backend/database"
	"splitwise-backend/models"

	"github.com/google/uuid"
)

// InviteToGroup creates an invitation and sends email/SMS
func InviteToGroup(groupID uuid.UUID, invitedBy uuid.UUID, email string, phone string) {
	// Check if invitation already exists
	var existing models.Invitation
	query := database.DB.Where("group_id = ? AND status = ?", groupID, "pending")
	if email != "" {
		query = query.Where("email = ?", email)
	} else if phone != "" {
		query = query.Where("phone = ?", phone)
	}

	if err := query.First(&existing).Error; err == nil {
		log.Printf("⚠️  Invitation already exists for %s/%s in group %s", email, phone, groupID)
		return
	}

	// Check if user is already registered
	var existingUser models.User
	if email != "" {
		if err := database.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
			// User exists, just add them to the group
			var existingMember models.GroupMember
			if err := database.DB.Where("group_id = ? AND user_id = ?", groupID, existingUser.ID).First(&existingMember).Error; err != nil {
				database.DB.Create(&models.GroupMember{
					GroupID: groupID,
					UserID:  existingUser.ID,
					Role:    "member",
				})
				log.Printf("✅ Added existing user %s to group %s", email, groupID)
			}
			return
		}
	}

	// Create invitation
	invitation := models.Invitation{
		GroupID:   groupID,
		InvitedBy: invitedBy,
		Email:     email,
		Phone:     phone,
		Status:    "pending",
	}

	if err := database.DB.Create(&invitation).Error; err != nil {
		log.Printf("❌ Failed to create invitation: %v", err)
		return
	}

	// Send notification
	var inviter models.User
	database.DB.First(&inviter, invitedBy)
	var group models.Group
	database.DB.First(&group, groupID)

	if email != "" {
		GetNotificationService().NotifyInvitation(email, inviter.Name, group.Name)
	}

	log.Printf("✅ Invitation sent to %s/%s for group %s", email, phone, groupID)
}
