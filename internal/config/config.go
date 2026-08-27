package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Auth     AuthConfig
	Upload   UploadConfig
	Email    EmailConfig
}

type ServerConfig struct {
	Port           string
	Environment    string
	Host           string
	AllowedOrigins []string
}

type DatabaseConfig struct {
	URL string
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

type AuthConfig struct {
	GoogleClientID     string
	AllowedEmailDomain string // e.g. "college.edu"; empty allows any domain
}

type UploadConfig struct {
	Dir          string
	MaxSize      int64
	AllowedTypes []string
}

type EmailConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// Don't return error if .env doesn't exist
		fmt.Println("No .env file found, using environment variables")
	}

	config := &Config{
		Server: ServerConfig{
			Port:           getEnv("PORT", "8080"),
			Environment:    getEnv("ENVIRONMENT", "development"),
			Host:           getEnv("HOST", "localhost"),
			AllowedOrigins: getEnvAsSlice("ALLOWED_ORIGINS", []string{"http://localhost:3000", "http://localhost:3001"}),
		},
		Database: DatabaseConfig{
			URL: getEnv("DATABASE_URL", "postgres://lostfound_user:lostfound_password@localhost:5432/lostfound?sslmode=disable"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", defaultJWTSecret),
			Expiration: getEnvAsDuration("JWT_EXPIRATION", 24*time.Hour),
		},
		Auth: AuthConfig{
			GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			AllowedEmailDomain: getEnv("ALLOWED_EMAIL_DOMAIN", ""),
		},
		Upload: UploadConfig{
			Dir:     getEnv("UPLOAD_DIR", "./uploads"),
			MaxSize: getEnvAsInt64("MAX_FILE_SIZE", 10*1024*1024), // 10MB
			// .webp removed: the imaging library cannot decode/encode it,
			// so webp uploads previously failed with a 500 at thumbnail time
			AllowedTypes: []string{".jpg", ".jpeg", ".png", ".gif"},
		},
		Email: EmailConfig{
			Host:     getEnv("SMTP_HOST", ""),
			Port:     getEnvAsInt("SMTP_PORT", 587),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@lostfound.com"),
		},
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// defaultJWTSecret is the placeholder used for local development. Sessions
// signed with a value everybody can read are not sessions, so it is rejected
// outside development.
const defaultJWTSecret = "your-secret-key-change-in-production"

// validate rejects configurations that are fine locally but unsafe once
// ENVIRONMENT is anything other than development. Failing at startup is the
// point: a misconfigured deployment should not boot and quietly serve traffic.
func (c *Config) validate() error {
	if c.Server.Environment == "development" {
		return nil
	}

	var problems []string
	if c.JWT.Secret == defaultJWTSecret || c.JWT.Secret == "" {
		problems = append(problems, "JWT_SECRET is unset or still the development placeholder")
	}
	if c.Auth.GoogleClientID == "" {
		problems = append(problems, "GOOGLE_CLIENT_ID is unset, so Google Sign-In cannot verify tokens")
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration for ENVIRONMENT=%q: %s", c.Server.Environment, strings.Join(problems, "; "))
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
