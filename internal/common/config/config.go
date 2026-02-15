package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Server
	Port string

	// Database (optional - for future projects)
	DatabaseURL string

	// Strike2 project
	Strike2ProxyToken    string
	Strike2CaptchaKey    string
	Strike2UpstreamProxy string
	Strike2Fingerprint   string
	Strike2Workers       int
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          os.Getenv("DATABASE_URL"),
		Strike2ProxyToken:    os.Getenv("PROXY_TOKEN"),
		Strike2CaptchaKey:    os.Getenv("CAPTCHA_API_KEY"),
		Strike2UpstreamProxy: os.Getenv("UPSTREAM_PROXY"),
		Strike2Fingerprint:   getEnv("FINGERPRINT", "chrome"),
		Strike2Workers:       getEnvInt("WORKERS", 500),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil {
			return n
		}
	}
	return fallback
}
