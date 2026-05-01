package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	AccessTokenExpiry   time.Duration
	RefreshTokenExpiry  time.Duration
	Env                 string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/circular_board?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", "change-me-in-production"),
		AccessTokenExpiry:  getDurationEnv("ACCESS_TOKEN_EXPIRY_MIN", 15) * time.Minute,
		RefreshTokenExpiry: getDurationEnv("REFRESH_TOKEN_EXPIRY_DAYS", 7) * time.Hour * 24,
		Env:                getEnv("APP_ENV", "development"),
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
