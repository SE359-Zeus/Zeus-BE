package configs

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port             string
	BaseURL          string
	Env              string
	ValkeyAddr       string
	RabbitMQURL      string
	JwtPublicKeyPath string
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	ShutdownTimeout  time.Duration
}

func Load() *Config {
	loadDotEnv()

	port := getEnv("MRP_PORT", "8081")
	return &Config{
		Port:             port,
		BaseURL:          getEnv("MRP_BASE_URL", "http://localhost:"+port),
		Env:              getEnv("MRP_ENV", "development"),
		ValkeyAddr:       normalizeAddr(getEnv("MRP_VALKEY_ADDR", getEnv("VALKEY_ADDR", "localhost:6379"))),
		RabbitMQURL:      getEnv("MRP_RABBITMQ_URL", getEnv("RABBITMQ_URL", "")),
		JwtPublicKeyPath: getEnv("JWT_PUBLIC_KEY_PATH", ""),
		ReadTimeout:      time.Duration(getEnvInt("MRP_READ_TIMEOUT_SEC", 15)) * time.Second,
		WriteTimeout:     time.Duration(getEnvInt("MRP_WRITE_TIMEOUT_SEC", 15)) * time.Second,
		ShutdownTimeout:  time.Duration(getEnvInt("MRP_SHUTDOWN_TIMEOUT_SEC", 10)) * time.Second,
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

func loadDotEnv() {
	paths := []string{".env", filepath.Join("..", ".env")}
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			value = strings.Trim(value, `"`)
			if key == "" {
				continue
			}
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, value)
			}
		}
		_ = file.Close()
		return
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return parsed
}
