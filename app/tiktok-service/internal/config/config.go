package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	LogLevel string
	Port     string

	GatewaySharedSecret string
	TikTokRedirectURI   string
	TikTokAuthHost      string
	TikTokAPIHost       string

	CORSAllowOrigins string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Port:     getEnv("TIKTOK_SERVICE_PORT", getEnv("PORT", "8086")),

		GatewaySharedSecret: getEnv("GATEWAY_SECRET", getEnv("GATEWAY_SHARED_SECRET", "")),
		TikTokRedirectURI:   getEnv("TIKTOK_REDIRECT_URI", ""),
		TikTokAuthHost:      getEnv("TIKTOK_AUTH_HOST", "https://auth.tiktok-shops.com"),
		TikTokAPIHost:       getEnv("TIKTOK_API_HOST", "https://open-api.tiktokglobalshop.com"),

		CORSAllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "*"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
