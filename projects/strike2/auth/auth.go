package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
)

// SimpleAuth provides single-token proxy authentication without a database.
// The token is read from the PROXY_TOKEN environment variable.
type SimpleAuth struct {
	token string
}

// NewSimpleAuth creates a new SimpleAuth with the given token.
// If token is empty, a random one is generated and logged to stdout.
func NewSimpleAuth(token string) *SimpleAuth {
	if token == "" {
		token = generateRandomToken()
		log.Printf("Generated proxy token: %s", token)
	}
	return &SimpleAuth{token: token}
}

// Token returns the configured proxy token (for startup logging).
func (a *SimpleAuth) Token() string {
	return a.token
}

// ValidateProxyAuth checks the Proxy-Authorization header against the configured token.
// Format: Basic base64(username:token) — the username is ignored, only the token matters.
func (a *SimpleAuth) ValidateProxyAuth(r *http.Request) bool {
	proxyAuth := r.Header.Get("Proxy-Authorization")
	if proxyAuth == "" {
		return false
	}

	if !strings.HasPrefix(proxyAuth, "Basic ") {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(proxyAuth[6:])
	if err != nil {
		return false
	}

	// Format: "anything:token" — username ignored
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return false
	}

	return parts[1] == a.token
}

// Reject407 writes a 407 Proxy Authentication Required response.
func Reject407(w http.ResponseWriter) {
	w.Header().Set("Proxy-Authenticate", `Basic realm="Strike2"`)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusProxyAuthRequired)
	w.Write([]byte("407 Proxy Authentication Required\r\n"))
}

func generateRandomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return "sk_live_" + hex.EncodeToString(b)
}
