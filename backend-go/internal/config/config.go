package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr           string
	JWTSecret          string
	JWTIssuer          string
	JWTTTL             time.Duration
	DatabaseURL        string
	CORSAllowedOrigins []string
	DevLoginEnabled    bool
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:           envOr("HTTP_ADDR", ":4000"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		JWTIssuer:          envOr("JWT_ISSUER", "chihuo-api"),
		JWTTTL:             durationOr("JWT_TTL", 7*24*time.Hour),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		CORSAllowedOrigins: csvEnv("CORS_ALLOWED_ORIGINS", []string{"*"}),
		DevLoginEnabled:    boolOr("DEV_LOGIN_ENABLED", false),
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must be at least 32 bytes and provided through the environment")
	}
	if cfg.JWTTTL <= 0 {
		return Config{}, errors.New("JWT_TTL must be positive")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationOr(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func boolOr(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func csvEnv(key string, fallback []string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
