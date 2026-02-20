package handlers

import (
	"net/http"
	"splitwise-backend/database"
	"splitwise-backend/models"
	"splitwise-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GET /api/groups/:id/balances
func GetGroupBalances(c *gin.Context) {
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

	var group models.Group
	database.DB.First(&group, groupID)

	// Calculate net balances from expenses
	netBalance := calculateNetBalances(groupID)

	// Simplify debts
	balances := simplifyDebts(netBalance)

	// Calculate total spent
	var totalSpent float64
	database.DB.Model(&models.Expense{}).Where("group_id = ?", groupID).Select("COALESCE(SUM(amount), 0)").Scan(&totalSpent)

	summary := models.GroupBalanceSummary{
		GroupID:    groupID,
		GroupName:  group.Name,
		Balances:   balances,
		TotalSpent: totalSpent,
	}

	utils.SuccessResponse(c, http.StatusOK, "", summary)
}

// GET /api/balances — overall balances across all groups for current user
func GetOverallBalances(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)

	// Get all groups the user is part of
	var memberships []models.GroupMember
	database.DB.Where("user_id = ?", userID).Find(&memberships)

	// Aggregate balances across all groups
	friendBalances := make(map[uuid.UUID]float64)

	for _, m := range memberships {
		netBalance := calculateNetBalances(m.GroupID)
		balances := simplifyDebts(netBalance)

		for _, b := range balances {
			if b.From == userID {
				// I owe this person
				friendBalances[b.To] -= b.Amount
			} else if b.To == userID {
				// This person owes me
				friendBalances[b.From] += b.Amount
			}
		}
	}

	var totalOwed, totalOwing float64
	var friends []models.FriendBalance

	for friendID, amount := range friendBalances {
		if utils.RoundToTwo(amount) == 0 {
			continue
		}

		var user models.User
		database.DB.First(&user, friendID)

		friends = append(friends, models.FriendBalance{
			UserID:    friendID,
			Name:      user.Name,
			Email:     user.Email,
			AvatarURL: user.AvatarURL,
			Amount:    utils.RoundToTwo(amount),
			Currency:  "INR",
		})

		if amount > 0 {
			totalOwed += amount
		} else {
			totalOwing += -amount
		}
	}

	summary := models.OverallBalanceSummary{
		TotalOwed:  utils.RoundToTwo(totalOwed),
		TotalOwing: utils.RoundToTwo(totalOwing),
		Friends:    friends,
	}

	utils.SuccessResponse(c, http.StatusOK, "", summary)
}

// Calculate net balance for each user in a group
func calculateNetBalances(groupID uuid.UUID) map[uuid.UUID]float64 {
	netBalance := make(map[uuid.UUID]float64)

	// Process expenses
	var expenses []models.Expense
	database.DB.Where("group_id = ?", groupID).Find(&expenses)

	for _, exp := range expenses {
		var splits []models.ExpenseSplit
		database.DB.Where("expense_id = ?", exp.ID).Find(&splits)

		for _, s := range splits {
			if s.UserID == exp.PaidBy {
				// Payer is owed: (what others owe for this expense) - (payer's own share)
				// We handle this by: payer paid full amount, owes their share
				netBalance[exp.PaidBy] += s.OwedAmount // gets cancelled below
			}
			// Everyone owes their share to the payer
			netBalance[s.UserID] -= s.OwedAmount
		}
		// Payer paid the full amount
		netBalance[exp.PaidBy] += exp.Amount
	}

	// Process settlements
	var settlements []models.Settlement
	database.DB.Where("group_id = ?", groupID).Find(&settlements)

	for _, s := range settlements {
		netBalance[s.PaidBy] -= s.Amount // payer's balance decreases
		netBalance[s.PaidTo] += s.Amount // payee's balance increases
	}

	return netBalance
}

// Simplify debts using greedy algorithm
func simplifyDebts(netBalance map[uuid.UUID]float64) []models.Balance {
	type userBalance struct {
		UserID uuid.UUID
		Amount float64
	}

	var creditors []userBalance // people who are owed money (positive balance)
	var debtors []userBalance   // people who owe money (negative balance)

	for userID, amount := range netBalance {
		rounded := utils.RoundToTwo(amount)
		if rounded > 0.01 {
			creditors = append(creditors, userBalance{userID, rounded})
		} else if rounded < -0.01 {
			debtors = append(debtors, userBalance{userID, -rounded})
		}
	}

	var results []models.Balance
	i, j := 0, 0

	for i < len(debtors) && j < len(creditors) {
		amount := debtors[i].Amount
		if creditors[j].Amount < amount {
			amount = creditors[j].Amount
		}
		amount = utils.RoundToTwo(amount)

		// Get user names
		var fromUser, toUser models.User
		database.DB.First(&fromUser, debtors[i].UserID)
		database.DB.First(&toUser, creditors[j].UserID)

		results = append(results, models.Balance{
			From:     debtors[i].UserID,
			FromName: fromUser.Name,
			To:       creditors[j].UserID,
			ToName:   toUser.Name,
			Amount:   amount,
			Currency: "INR",
		})

		debtors[i].Amount = utils.RoundToTwo(debtors[i].Amount - amount)
		creditors[j].Amount = utils.RoundToTwo(creditors[j].Amount - amount)

		if debtors[i].Amount < 0.01 {
			i++
		}
		if creditors[j].Amount < 0.01 {
			j++
		}
	}

	return results
}
