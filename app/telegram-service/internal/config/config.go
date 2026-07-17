package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	LogLevel string
	Port     string

	GatewaySharedSecret string
	CORSAllowOrigins    string

	BatchMinInterval   time.Duration
	BatchMaxRecipients int
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Port:     getEnv("TELEGRAM_SERVICE_PORT", getEnv("PORT", "8087")),

		GatewaySharedSecret: getEnv("GATEWAY_SECRET", getEnv("GATEWAY_SHARED_SECRET", "")),
		CORSAllowOrigins:    getEnv("CORS_ALLOW_ORIGINS", "*"),

		BatchMinInterval:   time.Duration(getEnvInt("TELEGRAM_BATCH_MIN_INTERVAL_MS", 50)) * time.Millisecond,
		BatchMaxRecipients: getEnvInt("TELEGRAM_BATCH_MAX_RECIPIENTS", 10000),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	raw := getEnv(key, "")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}
