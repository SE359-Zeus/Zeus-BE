package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Config holds all application configuration loaded from environment variables
type Config struct {
	// Server
	Port             string
	BaseURL          string
	JwtPublicKeyPath string
	RabbitMQURL      string

	// Database
	SQLiteDBPath string

	// Cache (Valkey/Redis)
	ValkeyAddr string

	// External Services
	MRPServiceURL string
	SCMServiceURL string
	ScmAPIKey     string
	MrpAPIKey     string

	// Logging
	LogLevel string

	// Observability
	Env           string
	AlloyURL      string
	AlloyUsername string
	AlloyPassword string
}

var loadEnvOnce sync.Once

// Load loads configuration from environment variables with sensible defaults
func Load() *Config {
	loadEnvOnce.Do(loadDotEnv)
	return &Config{
		Port:             getenvAny("8082", "SALES_PORT", "PORT"),
		BaseURL:          getenvAny("http://localhost:8082", "SALES_BASE_URL", "BASE_URL"),
		JwtPublicKeyPath: getenvAny("", "JWT_PUBLIC_KEY_PATH"),
		RabbitMQURL:      getenvAny("amqp://guest:guest@localhost:5672/", "RABBITMQ_URL"),
		SQLiteDBPath:     getenvAny("./sales.db", "SALES_SQLITE_DB", "SQLITE_DB"),
		ValkeyAddr:       getenvAny("localhost:6379", "SALES_VALKEY_ADDR", "VALKEY_ADDR"),
		MRPServiceURL:    getenvAny("http://localhost:8083", "MRP_BASE_URL", "MRP_URL"),
		SCMServiceURL:    getenvAny("http://localhost:8083", "SCM_BASE_URL", "SCM_URL"),
		ScmAPIKey:        getenvAny("scmkey01-admin-20260524", "SCM_API_KEY", "scm_api_key", "X_API_KEY"),
		MrpAPIKey:        getenvAny("mrpkey01-admin-20260524", "MRP_API_KEY", "mrp_api_key"),
		LogLevel:         getenvAny("info", "LOG_LEVEL"),
		Env:              getenvAny("development", "SALES_ENV", "APP_ENV"),
		AlloyURL:         getenvAny("", "ALLOY_URL"),
		AlloyUsername:    getenvAny("", "ALLOY_USERNAME"),
		AlloyPassword:    getenvAny("", "ALLOY_PASSWORD"),
	}
}

func loadDotEnv() {
	file, err := os.Open(".env")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}
		if len(value) >= 2 {
			if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
				value = value[1 : len(value)-1]
			}
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}

func getenvAny(fallback string, keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return fallback
}

// Helper function to get environment variable with fallback
func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// Helper function to get environment variable as integer with fallback
func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	intVal, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return intVal
}

// Helper function to get environment variable as boolean with fallback
func getenvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	boolVal, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return boolVal
}

// GetMRPURL returns the MRP service URL from config or environment
func GetMRPURL() string {
	if v := os.Getenv("MRP_URL"); v != "" {
		return v
	}
	return "http://localhost:8082"
}

// GetSCMURL returns the SCM service URL from config or environment
func GetSCMURL() string {
	if v := os.Getenv("SCM_URL"); v != "" {
		return v
	}
	return "http://localhost:8083"
}
