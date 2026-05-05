package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Env                string
	UploadDir          string
	MaxUploadBytes     int64
	// SendGrid
	SendGridAPIKey   string
	SendGridFromEmail string
	// アプリのベース URL（パスワードリセットリンク生成用）
	AppBaseURL string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/circular_board?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		AccessTokenExpiry:  getDurationEnv("ACCESS_TOKEN_EXPIRY_MIN", 15) * time.Minute,
		RefreshTokenExpiry: getDurationEnv("REFRESH_TOKEN_EXPIRY_DAYS", 7) * time.Hour * 24,
		Env:                getEnv("APP_ENV", "development"),
		UploadDir:          getEnv("UPLOAD_DIR", "/app/uploads"),
		MaxUploadBytes:     getInt64Env("MAX_UPLOAD_BYTES", 10<<20),
		SendGridAPIKey:    getEnv("SENDGRID_API_KEY", ""),
		SendGridFromEmail: getEnv("SENDGRID_FROM_EMAIL", "noreply@example.com"),
		AppBaseURL:        getEnv("APP_BASE_URL", "http://localhost"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue int64) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return time.Duration(defaultValue)
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return time.Duration(defaultValue)
	}
	return time.Duration(n)
}

func getInt64Env(key string, defaultValue int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return defaultValue
	}
	return n
}
