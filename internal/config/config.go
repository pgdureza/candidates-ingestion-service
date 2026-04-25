package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	APIPort          string
	DatabaseURL      string
	GCPProject       string
	PubSubTopic      string
	WorkerCount      int
	WorkerTimeout    time.Duration
	LogLevel         string
	Outbox           OutboxConfig
	PollIntervalMs   int
	WebhookRateLimit int
	CircuitBreaker   CircuitBreakerConfig
}

type OutboxConfig struct {
	RetentionDays   int
	CleanupSchedule string
	BatchSize       int
}

type CircuitBreakerConfig struct {
	FailureThreshold int
	OpenTimeoutS     int
	HalfOpenTimeoutS int
}

func Load() *Config {
	// side effect for local development for pubsub
	pubSubEmulatorHost := getEnv("PUBSUB_EMULATOR_HOST", "localhost:8085")
	if pubSubEmulatorHost != "" {
		os.Setenv("PUBSUB_EMULATOR_HOST", pubSubEmulatorHost)
	}

	return &Config{
		APIPort:          getEnv("API_PORT", "8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/candidates?sslmode=disable"),
		GCPProject:       getEnv("GCP_PROJECT", "test-project"),
		PubSubTopic:      getEnv("PUBSUB_TOPIC", "candidate-applications"),
		WorkerCount:      getEnvInt("WORKER_COUNT", 10),
		WorkerTimeout:    getEnvDuration("WORKER_TIMEOUT", 30*time.Second),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
		PollIntervalMs:   getEnvInt("POLL_INTERVAL_MS", 1000),
		WebhookRateLimit: getEnvInt("WEBHOOK_RATE_LIMIT", 1000),
		Outbox: OutboxConfig{
			RetentionDays:   getEnvInt("OUTBOX_RETENTION_DAYS", 0),
			CleanupSchedule: getEnv("OUTBOX_CLEANUP_SCHEDULE", "@every 15s"),
			BatchSize:       getEnvInt("OUTBOX_BATCH_SIZE", 50),
		},
		CircuitBreaker: CircuitBreakerConfig{
			FailureThreshold: getEnvInt("CIRCUITBREAKER_FAILURE_THREASHOLD", 5),
			OpenTimeoutS:     getEnvInt("CIRCUITBREAKER_TIMEOUT_S", 2000),
			HalfOpenTimeoutS: getEnvInt("CIRCUITBREAKER_HALF_OPEN_TIMEOUT_S", 1000),
		},
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
