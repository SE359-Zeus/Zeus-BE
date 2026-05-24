package config

import (
	"os"
	"strconv"
)

// Config holds all application configuration loaded from environment variables
type Config struct {
	// Server
	Port    string
	BaseURL string

	// Database
	SQLiteDBPath string

	// Cache (Valkey/Redis)
	ValkeyAddr string

	// External Services
	MRPServiceURL string
	SCMServiceURL string

	// Logging
	LogLevel string
}

// Load loads configuration from environment variables with sensible defaults
func Load() *Config {
	return &Config{
		Port:          getenv("SALES_PORT", "8082"),
		BaseURL:       getenv("SALES_BASE_URL", "http://localhost:8082"),
		SQLiteDBPath:  getenv("SALES_SQLITE_DB", "./sales.db"),
		ValkeyAddr:    getenv("SALES_VALKEY_ADDR", "localhost:6379"),
		MRPServiceURL: getenv("MRP_URL", "http://localhost:8082"),
		SCMServiceURL: getenv("SCM_URL", "http://localhost:8083"),
		LogLevel:      getenv("LOG_LEVEL", "info"),
	}
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
