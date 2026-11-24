package config

import (
	"os"
)

type Config struct {
	Port         string
	DBPath       string
	JWTSecret    string
	UploadAudio  string
	UploadCovers string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "vynza-super-secret-key-change-in-production-2026"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "vynza.db"
	}

	return &Config{
		Port:         port,
		DBPath:       dbPath,
		JWTSecret:    jwtSecret,
		UploadAudio:  "uploads/audio",
		UploadCovers: "uploads/covers",
	}
}
