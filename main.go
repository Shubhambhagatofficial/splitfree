package main

import (
	"log"
	"splitwise-backend/config"
	"splitwise-backend/database"
	"splitwise-backend/handlers"
	"splitwise-backend/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	config.Load()

	// Connect to database
	database.Connect()

	// Connect to Redis (optional, won't crash if unavailable)
	database.ConnectRedis()

	// Setup router
	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": config.AppConfig.AppName,
		})
	})

	// ==========================================
	// AUTH ROUTES (public)
	// ==========================================
	auth := r.Group("/auth")
	{
		auth.POST("/register", handlers.Register)
		auth.POST("/login", handlers.Login)
	}

	// ==========================================
	// API ROUTES (authenticated)
	// ==========================================
	api := r.Group("/api")
	api.Use(middleware.AuthRequired())
	{
		// User
		api.GET("/users/me", handlers.GetProfile)
		api.PUT("/users/me", handlers.UpdateProfile)
		api.PUT("/users/me/fcm-token", handlers.UpdateFCMToken)
		api.POST("/users/search", handlers.SearchUsers)

		// Groups
		api.POST("/groups", handlers.CreateGroup)
		api.GET("/groups", handlers.GetGroups)
		api.GET("/groups/:id", handlers.GetGroup)
		api.PUT("/groups/:id", handlers.UpdateGroup)
		api.POST("/groups/:id/members", handlers.AddMember)
		api.DELETE("/groups/:id/members/:uid", handlers.RemoveMember)
		api.POST("/groups/:id/invite", handlers.InviteToGroupHandler)

		// Expenses
		api.POST("/groups/:id/expenses", handlers.CreateExpense)
		api.GET("/groups/:id/expenses", handlers.GetGroupExpenses)
		api.GET("/expenses/:id", handlers.GetExpense)
		api.PUT("/expenses/:id", handlers.UpdateExpense)
		api.DELETE("/expenses/:id", handlers.DeleteExpense)

		// Balances
		api.GET("/groups/:id/balances", handlers.GetGroupBalances)
		api.GET("/balances", handlers.GetOverallBalances)

		// Settlements
		api.POST("/groups/:id/settle", handlers.CreateSettlement)
		api.GET("/groups/:id/settlements", handlers.GetGroupSettlements)

		// Activity
		api.GET("/activity", handlers.GetActivity)
		api.GET("/groups/:id/activity", handlers.GetGroupActivity)
	}

	// Start server
	port := config.AppConfig.Port
	log.Printf("🚀 %s server starting on port %s", config.AppConfig.AppName, port)
	log.Printf("📡 API docs: http://%s:%s/health", config.AppConfig.AppURL, port)

	addr := "0.0.0.0:" + port
	log.Printf("🚀 Listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
