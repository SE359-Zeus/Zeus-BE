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
	SMTPHost         string
	SMTPPort         string
	SMTPUser         string
	SMTPPass         string
	EmailFromAddress string
	EmailFromName    string
	EmailTemplateDir string
	RabbitMQURL      string
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
		SMTPHost:         getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:         getEnv("SMTP_PORT", "587"),
		SMTPUser:         getEnv("SMTP_USER", ""),
		SMTPPass:         getEnv("SMTP_PASS", ""),
		EmailFromAddress: getEnv("EMAIL_FROM_ADDRESS", ""),
		EmailFromName:    getEnv("EMAIL_FROM_NAME", ""),
		EmailTemplateDir: strings.TrimSpace(getEnv("EMAIL_TEMPLATE_DIR", "templates")),
		RabbitMQURL:      getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
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
