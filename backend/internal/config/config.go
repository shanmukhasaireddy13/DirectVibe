package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Environment specifies the deployment environment
type Environment string

const (
	EnvDev  Environment = "dev"
	EnvStg  Environment = "stg"
	EnvProd Environment = "prod"
)

// Config holds all application configuration
type Config struct {
	Env              Environment
	Port             string
	FrontendURL      string // For CORS
	StrictMatchWait  int    // seconds before moving to relaxed matching
	RelaxedMatchWait int    // seconds before moving to fallback matching
}

// Load reads from .env files and environment variables to build the Config
func Load() *Config {
	// Attempt to load .env file if it exists, ignore errors if it doesn't (useful in prod)
	_ = godotenv.Load()

	envStr := strings.ToLower(getEnv("APP_ENV", "dev"))
	env := EnvDev
	switch envStr {
	case "prod", "production":
		env = EnvProd
	case "stg", "staging":
		env = EnvStg
	}

	cfg := &Config{
		Env:              env,
		Port:             getEnv("PORT", "8080"),
		FrontendURL:      getEnv("FRONTEND_URL", "*"), // Default allow all in dev
		StrictMatchWait:  getEnvAsInt("STRICT_MATCH_WAIT_SEC", 5),
		RelaxedMatchWait: getEnvAsInt("RELAXED_MATCH_WAIT_SEC", 10),
	}

	log.Printf("Loaded Config for Environment: %s", cfg.Env)

	// In production, we should probably enforce a specific frontend URL
	if cfg.Env == EnvProd && cfg.FrontendURL == "*" {
		log.Println("WARNING: FRONTEND_URL is set to '*' in production. This is insecure.")
	}

	return cfg
}

// Helper functions for reading env vars
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return fallback
}
