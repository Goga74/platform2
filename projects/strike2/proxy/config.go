package proxy

import (
	"net/url"
)

// Config holds proxy configuration
type Config struct {
	// UpstreamProxy is the optional upstream proxy URL (e.g., http://user:pass@host:port)
	// If set, all requests will be forwarded through this proxy
	UpstreamProxy string

	// Fingerprint is the browser fingerprint to use for JA3 spoofing
	Fingerprint string
}

// ParseUpstreamProxy parses and validates the upstream proxy URL
func (c *Config) ParseUpstreamProxy() (*url.URL, error) {
	if c.UpstreamProxy == "" {
		return nil, nil
	}
	return url.Parse(c.UpstreamProxy)
}

// HasUpstreamProxy returns true if upstream proxy is configured
func (c *Config) HasUpstreamProxy() bool {
	return c.UpstreamProxy != ""
}
