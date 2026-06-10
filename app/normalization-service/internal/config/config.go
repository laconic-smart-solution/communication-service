package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	LogLevel string
	Port     string

	GatewaySharedSecret  string
	LineServiceURL       string
	FacebookServiceURL   string

	CORSAllowOrigins   string
	PlaygroundEnabled  bool
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Port:     getEnv("NORMALIZATION_SERVICE_PORT", getEnv("PORT", "8090")),

		GatewaySharedSecret: getEnv("GATEWAY_SECRET", getEnv("GATEWAY_SHARED_SECRET", "")),
		LineServiceURL:     getEnv("LINE_SERVICE_URL", "http://localhost:8083"),
		FacebookServiceURL: getEnv("FACEBOOK_SERVICE_URL", "http://localhost:8081"),

		CORSAllowOrigins:  getEnv("CORS_ALLOW_ORIGINS", "*"),
		PlaygroundEnabled: getEnvBool("PLAYGROUND_ENABLED", false),
	}
}

func (c *Config) PlatformURL(platform string) string {
	switch platform {
	case "line":
		return c.LineServiceURL
	case "facebook":
		return c.FacebookServiceURL
	default:
		return ""
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
