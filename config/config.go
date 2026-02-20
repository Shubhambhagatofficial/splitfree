package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DatabaseURL      string
	RedisURL         string
	JWTSecret        string
	SendGridAPIKey   string
	SendGridFrom     string
	FirebaseCredPath string
	AppName          string
	AppURL           string
}

var AppConfig *Config

func Load() {
	godotenv.Load() // Load .env file if present

	AppConfig = &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgresql://postgres:hneVjGIxLOJaifavqHvFVVWhZWxmkYTU@postgres.railway.internal:5432/railway"),
		RedisURL:         getEnv("REDIS_URL", "redis://default:DDXmTfkbvcZEthmspowFOpquRMyHTObI@redis.railway.internal:6379"),
		JWTSecret:        getEnv("JWT_SECRET", "9430306906sb7976331377sb9939010140pd"),
		SendGridAPIKey:   getEnv("SENDGRID_API_KEY", ""),
		SendGridFrom:     getEnv("SENDGRID_FROM_EMAIL", "noreply@splitapp.com"),
		FirebaseCredPath: getEnv("FIREBASE_CREDENTIALS", "firebase-credentials.json"),
		AppName:          getEnv("APP_NAME", "SplitFree"),
		AppURL:           getEnv("APP_URL", "https://splitfree-production.up.railway.app"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
