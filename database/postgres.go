package database

import (
	"log"
	"splitwise-backend/config"
	"splitwise-backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() {
	var err error
	DB, err = gorm.Open(postgres.Open(config.AppConfig.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("✅ Database connected successfully")

	// Auto-migrate all models
	err = DB.AutoMigrate(
		&models.User{},
		&models.Group{},
		&models.GroupMember{},
		&models.Expense{},
		&models.ExpenseSplit{},
		&models.Settlement{},
		&models.Activity{},
		&models.Invitation{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("✅ Database migrated successfully")
}
