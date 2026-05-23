package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort  string
	DBPath      string
	JWTKeyPath  string
	ValkeyAddr  string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Printf(".env file not found, using OS env vars and defaults")
	}
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8083"),
		DBPath:     getEnv("DB_PATH", "system.db"),
		JWTKeyPath: getEnv("JWT_PRIVATE_KEY_PATH", ""),
		ValkeyAddr: getEnv("VALKEY_ADDR", "localhost:6379"),
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
