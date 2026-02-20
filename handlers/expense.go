package handlers

import (
	"fmt"
	"net/http"
	"splitwise-backend/database"
	"splitwise-backend/models"
	"splitwise-backend/services"
	"splitwise-backend/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// POST /api/groups/:id/expenses
func CreateExpense(c *gin.Context) {
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

	var req models.CreateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// Parse expense date
	expenseDate := time.Now()
	if req.ExpenseDate != "" {
		parsed, err := time.Parse("2006-01-02", req.ExpenseDate)
		if err == nil {
			expenseDate = parsed
		}
	}

	currency := req.Currency
	if currency == "" {
		currency = "INR"
	}

	expense := models.Expense{
		GroupID:     groupID,
		PaidBy:      userID,
		Description: req.Description,
		Amount:      req.Amount,
		Currency:    currency,
		Category:    req.Category,
		SplitType:   req.SplitType,
		Notes:       req.Notes,
		ExpenseDate: expenseDate,
	}

	if err := database.DB.Create(&expense).Error; err != nil {
		utils.InternalError(c, "Failed to create expense")
		return
	}

	// Calculate and create splits
	splits, err := calculateSplits(expense, req.Splits, groupID)
	if err != nil {
		// Rollback expense
		database.DB.Delete(&expense)
		utils.BadRequest(c, err.Error())
		return
	}

	for _, split := range splits {
		split.ExpenseID = expense.ID
		database.DB.Create(&split)
	}

	// Log activity
	var payer models.User
	database.DB.First(&payer, userID)
	var group models.Group
	database.DB.First(&group, groupID)

	database.DB.Create(&models.Activity{
		GroupID:     groupID,
		UserID:      userID,
		Type:        "expense_added",
		ReferenceID: expense.ID,
		Description: fmt.Sprintf("%s added \"%s\" (%s %.2f)", payer.Name, expense.Description, expense.Currency, expense.Amount),
	})

	// Send notifications asynchronously
	go services.GetNotificationService().NotifyExpenseAdded(expense, splits, payer, group)

	// Build response
	response := buildExpenseResponse(expense.ID)
	utils.SuccessResponse(c, http.StatusCreated, "Expense added", response)
}

// GET /api/groups/:id/expenses
func GetGroupExpenses(c *gin.Context) {
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

	var expenses []models.Expense
	database.DB.Where("group_id = ?", groupID).
		Order("expense_date DESC, created_at DESC").
		Offset(pagination.Offset()).
		Limit(pagination.Limit).
		Find(&expenses)

	var responses []models.ExpenseResponse
	for _, e := range expenses {
		responses = append(responses, buildExpenseResponse(e.ID))
	}

	utils.SuccessResponse(c, http.StatusOK, "", responses)
}

// GET /api/expenses/:id
func GetExpense(c *gin.Context) {
	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "Invalid expense ID")
		return
	}

	response := buildExpenseResponse(expenseID)
	if response.ID == uuid.Nil {
		utils.NotFound(c, "Expense not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", response)
}

// PUT /api/expenses/:id
func UpdateExpense(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)
	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "Invalid expense ID")
		return
	}

	var expense models.Expense
	if err := database.DB.First(&expense, expenseID).Error; err != nil {
		utils.NotFound(c, "Expense not found")
		return
	}

	if !isMember(expense.GroupID, userID) {
		utils.Unauthorized(c, "You are not a member of this group")
		return
	}

	var req models.UpdateExpenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	updates := map[string]interface{}{}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Amount > 0 {
		updates["amount"] = req.Amount
	}
	if req.Category != "" {
		updates["category"] = req.Category
	}
	if req.Notes != "" {
		updates["notes"] = req.Notes
	}

	database.DB.Model(&expense).Updates(updates)

	// Recalculate splits if amount or split type changed
	if req.Amount > 0 || req.SplitType != "" || len(req.Splits) > 0 {
		// Delete old splits
		database.DB.Where("expense_id = ?", expenseID).Delete(&models.ExpenseSplit{})

		// Reload expense
		database.DB.First(&expense, expenseID)

		splitType := req.SplitType
		if splitType == "" {
			splitType = expense.SplitType
		}
		expense.SplitType = splitType

		splits, err := calculateSplits(expense, req.Splits, expense.GroupID)
		if err != nil {
			utils.BadRequest(c, err.Error())
			return
		}

		for _, split := range splits {
			split.ExpenseID = expense.ID
			database.DB.Create(&split)
		}
	}

	// Log activity
	var editor models.User
	database.DB.First(&editor, userID)

	database.DB.Create(&models.Activity{
		GroupID:     expense.GroupID,
		UserID:      userID,
		Type:        "expense_updated",
		ReferenceID: expense.ID,
		Description: fmt.Sprintf("%s updated \"%s\"", editor.Name, expense.Description),
	})

	response := buildExpenseResponse(expense.ID)
	utils.SuccessResponse(c, http.StatusOK, "Expense updated", response)
}

// DELETE /api/expenses/:id
func DeleteExpense(c *gin.Context) {
	userID := utils.GetCurrentUserID(c)
	expenseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		utils.BadRequest(c, "Invalid expense ID")
		return
	}

	var expense models.Expense
	if err := database.DB.First(&expense, expenseID).Error; err != nil {
		utils.NotFound(c, "Expense not found")
		return
	}

	if !isMember(expense.GroupID, userID) {
		utils.Unauthorized(c, "You are not a member of this group")
		return
	}

	// Log before deleting
	var deleter models.User
	database.DB.First(&deleter, userID)

	database.DB.Create(&models.Activity{
		GroupID:     expense.GroupID,
		UserID:      userID,
		Type:        "expense_deleted",
		Description: fmt.Sprintf("%s deleted \"%s\" (%s %.2f)", deleter.Name, expense.Description, expense.Currency, expense.Amount),
	})

	// Delete splits and expense
	database.DB.Where("expense_id = ?", expenseID).Delete(&models.ExpenseSplit{})
	database.DB.Delete(&expense)

	utils.SuccessResponse(c, http.StatusOK, "Expense deleted", nil)
}

// Calculate splits based on split type
func calculateSplits(expense models.Expense, splitInputs []models.SplitInput, groupID uuid.UUID) ([]models.ExpenseSplit, error) {
	var splits []models.ExpenseSplit

	switch expense.SplitType {
	case "equal":
		// Split equally among all group members
		var members []models.GroupMember
		database.DB.Where("group_id = ?", groupID).Find(&members)

		if len(members) == 0 {
			return nil, fmt.Errorf("no members in group")
		}

		perPerson := utils.RoundToTwo(expense.Amount / float64(len(members)))

		// Handle rounding remainder
		remainder := utils.RoundToTwo(expense.Amount - perPerson*float64(len(members)))

		for i, m := range members {
			amount := perPerson
			if i == 0 {
				amount = utils.RoundToTwo(amount + remainder) // first person gets the remainder
			}
			paidAmount := 0.0
			if m.UserID == expense.PaidBy {
				paidAmount = expense.Amount
			}

			splits = append(splits, models.ExpenseSplit{
				UserID:     m.UserID,
				OwedAmount: amount,
				PaidAmount: paidAmount,
			})
		}

	case "exact":
		// Each person owes a specific amount
		if len(splitInputs) == 0 {
			return nil, fmt.Errorf("splits required for exact split type")
		}

		var total float64
		for _, s := range splitInputs {
			total += s.Value
		}

		if utils.RoundToTwo(total) != utils.RoundToTwo(expense.Amount) {
			return nil, fmt.Errorf("split amounts (%.2f) don't add up to total (%.2f)", total, expense.Amount)
		}

		for _, s := range splitInputs {
			uid, err := uuid.Parse(s.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user ID: %s", s.UserID)
			}

			paidAmount := 0.0
			if uid == expense.PaidBy {
				paidAmount = expense.Amount
			}

			splits = append(splits, models.ExpenseSplit{
				UserID:     uid,
				OwedAmount: utils.RoundToTwo(s.Value),
				PaidAmount: paidAmount,
			})
		}

	case "percentage":
		// Each person owes a percentage
		if len(splitInputs) == 0 {
			return nil, fmt.Errorf("splits required for percentage split type")
		}

		var totalPercent float64
		for _, s := range splitInputs {
			totalPercent += s.Value
		}

		if utils.RoundToTwo(totalPercent) != 100.0 {
			return nil, fmt.Errorf("percentages must add up to 100, got %.2f", totalPercent)
		}

		for _, s := range splitInputs {
			uid, err := uuid.Parse(s.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user ID: %s", s.UserID)
			}

			owedAmount := utils.RoundToTwo(expense.Amount * s.Value / 100.0)
			paidAmount := 0.0
			if uid == expense.PaidBy {
				paidAmount = expense.Amount
			}

			splits = append(splits, models.ExpenseSplit{
				UserID:     uid,
				OwedAmount: owedAmount,
				PaidAmount: paidAmount,
			})
		}

	case "shares":
		// Split by shares (e.g., 2 shares, 1 share, 3 shares)
		if len(splitInputs) == 0 {
			return nil, fmt.Errorf("splits required for shares split type")
		}

		var totalShares float64
		for _, s := range splitInputs {
			totalShares += s.Value
		}

		if totalShares <= 0 {
			return nil, fmt.Errorf("total shares must be greater than 0")
		}

		for _, s := range splitInputs {
			uid, err := uuid.Parse(s.UserID)
			if err != nil {
				return nil, fmt.Errorf("invalid user ID: %s", s.UserID)
			}

			owedAmount := utils.RoundToTwo(expense.Amount * s.Value / totalShares)
			paidAmount := 0.0
			if uid == expense.PaidBy {
				paidAmount = expense.Amount
			}

			splits = append(splits, models.ExpenseSplit{
				UserID:     uid,
				OwedAmount: owedAmount,
				PaidAmount: paidAmount,
			})
		}

	default:
		return nil, fmt.Errorf("invalid split type: %s", expense.SplitType)
	}

	return splits, nil
}

// Build expense response with payer name and split details
func buildExpenseResponse(expenseID uuid.UUID) models.ExpenseResponse {
	var expense models.Expense
	if err := database.DB.First(&expense, expenseID).Error; err != nil {
		return models.ExpenseResponse{}
	}

	var payer models.User
	database.DB.First(&payer, expense.PaidBy)

	var dbSplits []models.ExpenseSplit
	database.DB.Where("expense_id = ?", expenseID).Find(&dbSplits)

	var splitResponses []models.SplitResponse
	for _, s := range dbSplits {
		var user models.User
		database.DB.First(&user, s.UserID)
		splitResponses = append(splitResponses, models.SplitResponse{
			UserID:     s.UserID,
			UserName:   user.Name,
			OwedAmount: s.OwedAmount,
			PaidAmount: s.PaidAmount,
		})
	}

	return models.ExpenseResponse{
		ID:          expense.ID,
		GroupID:     expense.GroupID,
		PaidBy:      expense.PaidBy,
		PayerName:   payer.Name,
		Description: expense.Description,
		Amount:      expense.Amount,
		Currency:    expense.Currency,
		Category:    expense.Category,
		SplitType:   expense.SplitType,
		Notes:       expense.Notes,
		ExpenseDate: expense.ExpenseDate,
		Splits:      splitResponses,
		CreatedAt:   expense.CreatedAt,
	}
}
