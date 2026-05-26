package config

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort          string
	DBPath              string
	RabbitMQURL         string
	ValkeyAddr          string
	AgingThresholdYears int
	JwtPublicKeyPath    string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		slog.Info(".env file not found, using OS env vars and defaults",
			slog.String("service", "scm"),
			slog.String("event", "config_fallback"),
		)
	}
	return &Config{
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		DBPath:              getEnv("DB_PATH", "scm.db"),
		RabbitMQURL:         getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		ValkeyAddr:          getEnv("VALKEY_ADDR", "redis://localhost:6379"),
		AgingThresholdYears: getEnvInt("AGING_THRESHOLD_YEARS", 5),
		JwtPublicKeyPath:    getEnv("JWT_PUBLIC_KEY_PATH", ""),
	}
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
