package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv   string
	LogLevel string
	Port     string

	GatewaySharedSecret string
	LazadaAppKey        string
	LazadaAppSecret     string
	LazadaRedirectURI   string
	LazadaCountry       string
	LazadaForceAuth     bool
	LazadaAuthBaseURL   string

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
		Port:     getEnv("LAZADA_SERVICE_PORT", getEnv("PORT", "8084")),

		GatewaySharedSecret: getEnv("GATEWAY_SECRET", getEnv("GATEWAY_SHARED_SECRET", "")),
		LazadaAppKey:        getEnv("LAZADA_APP_KEY", ""),
		LazadaAppSecret:     getEnv("LAZADA_APP_SECRET", ""),
		LazadaRedirectURI:   strings.TrimRight(getEnv("LAZADA_REDIRECT_URI", ""), "/"),
		LazadaCountry:       strings.ToLower(strings.TrimSpace(getEnv("LAZADA_COUNTRY", ""))),
		LazadaForceAuth:     getEnvBool("LAZADA_FORCE_AUTH", false),
		LazadaAuthBaseURL:   strings.TrimRight(strings.TrimSpace(getEnv("LAZADA_AUTH_BASE_URL", "")), "/"),

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
