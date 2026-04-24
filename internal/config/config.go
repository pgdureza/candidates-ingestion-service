package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	APIPort            string
	DatabaseURL        string
	GCPProject         string
	PubSubTopic        string
	PubSubEmulatorHost string
	WorkerCount        int
	WorkerTimeout      time.Duration
	LogLevel           string
}

func Load() *Config {
	return &Config{
		APIPort:            getEnv("API_PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/candidates?sslmode=disable"),
		GCPProject:         getEnv("GCP_PROJECT", "test-project"),
		PubSubTopic:        getEnv("PUBSUB_TOPIC", "candidate-applications"),
		PubSubEmulatorHost: getEnv("PUBSUB_EMULATOR_HOST", "localhost:8085"),
		WorkerCount:        getEnvInt("WORKER_COUNT", 10),
		WorkerTimeout:      getEnvDuration("WORKER_TIMEOUT", 30*time.Second),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}
