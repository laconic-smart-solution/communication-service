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
		Port:     getEnv("WECHAT_SERVICE_PORT", getEnv("PORT", "8090")),

		GatewaySharedSecret: getEnv("GATEWAY_SECRET", getEnv("GATEWAY_SHARED_SECRET", "")),
		CORSAllowOrigins:    getEnv("CORS_ALLOW_ORIGINS", "*"),

		BatchMinInterval:   time.Duration(getEnvInt("WECHAT_BATCH_MIN_INTERVAL_MS", 250)) * time.Millisecond,
		BatchMaxRecipients: getEnvInt("WECHAT_BATCH_MAX_RECIPIENTS", 500),
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
