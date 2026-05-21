package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort          string
	DBPath              string
	RabbitMQURL         string
	AgingThresholdYears int
	JwtPublicKey        string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf(".env file not found, using OS env vars and defaults")
	}
	return &Config{
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		DBPath:              getEnv("DB_PATH", "scm.db"),
		RabbitMQURL:         getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),
		AgingThresholdYears: getEnvInt("AGING_THRESHOLD_YEARS", 5),
		JwtPublicKey:        getEnv("JWT_PUBLIC_KEY", ""),
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
