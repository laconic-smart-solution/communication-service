package config

import (
	"os"
	"path/filepath"
	"strings"

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
	loadDotEnv()

	return &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Port:     getEnv("TIKTOK_SERVICE_PORT", getEnv("PORT", "8086")),

		GatewaySharedSecret: getEnv("GATEWAY_SECRET", getEnv("GATEWAY_SHARED_SECRET", "")),
		TikTokRedirectURI:   strings.TrimRight(getEnv("TIKTOK_REDIRECT_URI", ""), "/"),
		TikTokAuthHost:      getEnv("TIKTOK_AUTH_HOST", "https://auth.tiktok-shops.com"),
		TikTokAPIHost:       getEnv("TIKTOK_API_HOST", "https://open-api.tiktokglobalshop.com"),

		CORSAllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "*"),
	}
}

func loadDotEnv() {
	candidates := []string{
		".env",
		filepath.Join("app", "tiktok-service", ".env"),
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates,
			filepath.Join(dir, ".env"),
			filepath.Join(dir, "..", "..", ".env"),
		)
	}
	for _, path := range candidates {
		_ = godotenv.Load(path)
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
