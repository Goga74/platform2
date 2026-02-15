package config

import (
	"fmt"
	"os"
)

type Config struct {
	// Server
	Port string

	// Database
	DatabaseURL string

	// Strike2 project
	Strike2CaptchaKey string
	Strike2ProxyToken string
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		Strike2CaptchaKey: os.Getenv("STRIKE2_CAPTCHA_KEY"),
		Strike2ProxyToken: os.Getenv("STRIKE2_PROXY_TOKEN"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
