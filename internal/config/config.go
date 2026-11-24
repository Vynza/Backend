package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DBDriver     string // "sqlite" or "postgres" (for Supabase)
	DBUrl        string // Supabase PostgreSQL connection string
	JWTSecret    string
	UploadAudio  string
	UploadCovers string
}

func Load() *Config {
	_ = godotenv.Load() // Automatically read .env if present
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "vynza-super-secret-key-change-in-production-2026"
	}

	// Supabase PostgreSQL support:
	// Example: postgresql://postgres.xxxx:yourpassword@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?sslmode=require
	dbUrl := os.Getenv("DATABASE_URL")
	dbDriver := "postgres"
	if dbUrl == "" {
		dbDriver = "sqlite"
		dbUrl = os.Getenv("DB_PATH")
		if dbUrl == "" {
			dbUrl = "vynza.db"
		}
	}

	return &Config{
		Port:         port,
		DBDriver:     dbDriver,
		DBUrl:        dbUrl,
		JWTSecret:    jwtSecret,
		UploadAudio:  "uploads/audio",
		UploadCovers: "uploads/covers",
	}
}
