package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                   string
	LogLevel               string
	DBHost                 string
	DBPort                 string
	DBUser                 string
	DBPassword             string
	DBName                 string
	DBSSLMode              string
	DBTimezone             string
	SessionTTLHours        int
	BootstrapAdminUsername string
	BootstrapAdminPassword string
	SMTPHost               string
	SMTPPort               string
	SMTPUsername           string
	SMTPPassword           string
	SMTPFromEmail          string
	SMTPFromName           string
	OTPTTLMinutes          int
	ResetSessionTTLMinutes int
	CORSAllowedOrigins     []string
}

func Load() (*Config, error) {
	// Best-effort env loading:
	// - ".env" when running from backend directory
	// - "backend/.env" when running from repository root
	_ = godotenv.Load(".env")
	_ = godotenv.Load("backend/.env")

	return &Config{
		Port:                   getEnv("PORT", "8080"),
		LogLevel:               getEnv("LOG_LEVEL", "INFO"),
		DBHost:                 getEnv("DB_HOST", "127.0.0.1"),
		DBPort:                 getEnv("DB_PORT", "5433"),
		DBUser:                 getEnv("DB_USER", "admin"),
		DBPassword:             getEnv("DB_PASSWORD", "admin123"),
		DBName:                 getEnv("DB_NAME", "oper-plan"),
		DBSSLMode:              getEnv("DB_SSLMODE", "disable"),
		DBTimezone:             getEnv("DB_TIMEZONE", "Asia/Qyzylorda"),
		SessionTTLHours:        getEnvInt("SESSION_TTL_HOURS", 24),
		BootstrapAdminUsername: getEnv("BOOTSTRAP_ADMIN_USERNAME", "admin"),
		BootstrapAdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		SMTPHost:               getEnv("SMTP_HOST", ""),
		SMTPPort:               getEnv("SMTP_PORT", "587"),
		SMTPUsername:           getEnv("SMTP_USERNAME", ""),
		SMTPPassword:           getEnv("SMTP_PASSWORD", ""),
		SMTPFromEmail:          getEnv("SMTP_FROM_EMAIL", ""),
		SMTPFromName:           getEnv("SMTP_FROM_NAME", "Oper Plan"),
		OTPTTLMinutes:          getEnvInt("OTP_TTL_MINUTES", 10),
		ResetSessionTTLMinutes: getEnvInt("RESET_SESSION_TTL_MINUTES", 15),
		CORSAllowedOrigins: getEnvList(
			"CORS_ALLOWED_ORIGINS",
			[]string{
				"http://localhost:5173",
				"http://127.0.0.1:5173",
				"http://localhost:4173",
				"http://127.0.0.1:4173",
			},
		),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return defaultValue
	}

	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return defaultValue
	}

	return parsed
}

func getEnvList(key string, defaultValue []string) []string {
	value, exists := os.LookupEnv(key)
	if !exists || strings.TrimSpace(value) == "" {
		return defaultValue
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}

	return result
}
