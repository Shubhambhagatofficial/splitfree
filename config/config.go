package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	SendGridAPIKey  string
	SendGridFrom    string
	FirebaseCredPath string
	AppName         string
	AppURL          string
}

var AppConfig *Config

func Load() {
	godotenv.Load() // Load .env file if present

	AppConfig = &Config{
		Port:            getEnv("PORT", "8080"),
		DatabaseURL:     getEnv("DATABASE_URL", "host=localhost user=postgres password=postgres dbname=splitwise_dev port=5432 sslmode=disable"),
		RedisURL:        getEnv("REDIS_URL", "localhost:6379"),
		JWTSecret:       getEnv("JWT_SECRET", "your-super-secret-key-change-in-production"),
		SendGridAPIKey:  getEnv("SENDGRID_API_KEY", ""),
		SendGridFrom:    getEnv("SENDGRID_FROM_EMAIL", "noreply@splitapp.com"),
		FirebaseCredPath: getEnv("FIREBASE_CREDENTIALS", "firebase-credentials.json"),
		AppName:         getEnv("APP_NAME", "SplitApp"),
		AppURL:          getEnv("APP_URL", "http://localhost:8080"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
