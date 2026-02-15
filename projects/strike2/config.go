package strike2

import (
	"os"
	"strconv"
)

// LoadConfig reads Strike2 configuration from environment variables.
// All Strike2-specific variables use the STRIKE2_ prefix.
func LoadConfig() Config {
	return Config{
		ProxyToken:    os.Getenv("STRIKE2_PROXY_TOKEN"),
		CaptchaKey:    os.Getenv("STRIKE2_CAPTCHA_KEY"),
		UpstreamProxy: os.Getenv("STRIKE2_UPSTREAM_PROXY"),
		Fingerprint:   getEnv("FINGERPRINT", "chrome"),
		Workers:       getEnvInt("WORKERS", 500),
	}
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
