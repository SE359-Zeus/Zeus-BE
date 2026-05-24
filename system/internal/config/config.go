package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort       string
	DBPath           string
	JWTKeyPath       string
	ValkeyAddr       string
	ResendAPIKey     string
	EmailFromAddress string
	EmailFromName    string
	EmailTemplateDir string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf(".env file not found, using OS env vars and defaults")
	}
	return &Config{
		ServerPort:       getEnv("SERVER_PORT", "8083"),
		DBPath:           getEnv("DB_PATH", "system.db"),
		JWTKeyPath:       getEnv("JWT_PRIVATE_KEY_PATH", ""),
		ValkeyAddr:       normalizeAddr(getEnv("VALKEY_ADDR", "localhost:6379")),
		ResendAPIKey:     getEnv("RESEND_API_KEY", ""),
		EmailFromAddress: getEnv("EMAIL_FROM_ADDRESS", getEnv("RESEND_FROM_EMAIL", "")),
		EmailFromName:    getEnv("EMAIL_FROM_NAME", ""),
		EmailTemplateDir: strings.TrimSpace(getEnv("EMAIL_TEMPLATE_DIR", "templates")),
	}
}

func normalizeAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "redis://")
	if i := strings.Index(addr, "/"); i >= 0 {
		addr = addr[:i]
	}
	return addr
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
