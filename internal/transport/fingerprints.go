package transport

import (
	tls "github.com/refraction-networking/utls"
)

// Fingerprint represents a browser TLS fingerprint
type Fingerprint struct {
	ID        tls.ClientHelloID
	Name      string
	UserAgent string
}

// GetFingerprints returns a list of available browser fingerprints
func GetFingerprints() []Fingerprint {
	return []Fingerprint{
		{
			ID:        tls.HelloChrome_Auto,
			Name:      "Chrome",
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		},
		{
			ID:        tls.HelloFirefox_Auto,
			Name:      "Firefox",
			UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		},
		{
			ID:        tls.HelloIOS_Auto,
			Name:      "Safari",
			UserAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		},
	}
}

// GetRandomFingerprint returns a random fingerprint from the list
func GetRandomFingerprint() Fingerprint {
	fps := GetFingerprints()
	return fps[0]
}
