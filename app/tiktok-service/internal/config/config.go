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

	GatewaySharedSecret string
	TikTokAppKey        string
	TikTokAppSecret     string
	TikTokRedirectURI   string
	TikTokAuthHost      string
	TikTokAPIHost       string

	DatabaseURL          string
	DBMaxOpenConns       int
	DBMaxIdleConns       int
	DBConnMaxLifetimeMin int
	DBAutoMigrate        bool

	CORSAllowOrigins string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		AppEnv:   getEnv("APP_ENV", "development"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		Port:     getEnv("TIKTOK_SERVICE_PORT", getEnv("PORT", "8086")),

		GatewaySharedSecret: getEnv("GATEWAY_SECRET", getEnv("GATEWAY_SHARED_SECRET", "")),
		TikTokAppKey:        getEnv("TIKTOK_APP_KEY", ""),
		TikTokAppSecret:     getEnv("TIKTOK_APP_SECRET", ""),
		TikTokRedirectURI:   getEnv("TIKTOK_REDIRECT_URI", ""),
		TikTokAuthHost:      getEnv("TIKTOK_AUTH_HOST", "https://auth.tiktok-shops.com"),
		TikTokAPIHost:       getEnv("TIKTOK_API_HOST", "https://open-api.tiktokglobalshop.com"),

		DatabaseURL:          getEnv("DATABASE_URL", ""),
		DBMaxOpenConns:       getEnvInt("DB_MAX_OPEN_CONNS", 20),
		DBMaxIdleConns:       getEnvInt("DB_MAX_IDLE_CONNS", 5),
		DBConnMaxLifetimeMin: getEnvInt("DB_CONN_MAX_LIFETIME_MIN", 30),
		DBAutoMigrate:        getEnvBool("DB_AUTO_MIGRATE", true),

		CORSAllowOrigins: getEnv("CORS_ALLOW_ORIGINS", "*"),
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
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
