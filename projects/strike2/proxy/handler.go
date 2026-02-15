package proxy

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Goga74/platform2/internal/transport"
)

// Handler handles HTTP/HTTPS proxy requests with JA3 fingerprint spoofing
type Handler struct {
	config      *Config
	fingerprint transport.Fingerprint
	upstreamURL *url.URL
}

// NewHandler creates a new proxy handler
func NewHandler(config *Config) (*Handler, error) {
	h := &Handler{
		config: config,
	}

	// Set fingerprint (default to Chrome)
	if config.Fingerprint != "" {
		for _, fp := range transport.GetFingerprints() {
			if strings.EqualFold(fp.Name, config.Fingerprint) {
				h.fingerprint = fp
				break
			}
		}
	}
	if h.fingerprint.Name == "" {
		h.fingerprint = transport.GetRandomFingerprint()
	}

	// Parse upstream proxy if configured
	if config.HasUpstreamProxy() {
		var err error
		h.upstreamURL, err = url.Parse(config.UpstreamProxy)
		if err != nil {
			return nil, fmt.Errorf("invalid upstream proxy URL: %w", err)
		}
		log.Printf("[Proxy] Upstream proxy configured: %s", h.upstreamURL.Host)
	}

	return h, nil
}

// ServeHTTP handles incoming proxy requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		h.handleConnect(w, r)
	} else {
		h.handleHTTP(w, r)
	}
}

// handleConnect handles HTTPS CONNECT tunneling
func (h *Handler) handleConnect(w http.ResponseWriter, r *http.Request) {
	log.Printf("[Proxy] CONNECT %s", r.Host)

	targetAddr := r.Host
	if !strings.Contains(targetAddr, ":") {
		targetAddr += ":443"
	}

	client, err := transport.NewUTLSClient(h.fingerprint)
	if err != nil {
		http.Error(w, "Failed to create TLS client", http.StatusInternalServerError)
		return
	}

	var targetConn net.Conn
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if h.upstreamURL != nil {
		targetConn, err = h.dialViaUpstreamProxy(ctx, targetAddr)
	} else {
		targetConn, err = h.dialDirectTLS(ctx, targetAddr, client)
	}

	if err != nil {
		log.Printf("[Proxy] CONNECT failed to %s: %v", targetAddr, err)
		http.Error(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusBadGateway)
		return
	}
	defer targetConn.Close()

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer clientConn.Close()

	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	done := make(chan struct{}, 2)

	go func() {
		io.Copy(targetConn, clientConn)
		done <- struct{}{}
	}()

	go func() {
		io.Copy(clientConn, targetConn)
		done <- struct{}{}
	}()

	<-done
}

// dialDirectTLS connects directly to target using raw TCP (client does TLS)
func (h *Handler) dialDirectTLS(ctx context.Context, addr string, client *transport.UTLSClient) (net.Conn, error) {
	_ = client // client not used for CONNECT tunneling — client performs TLS
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return dialer.DialContext(ctx, "tcp", addr)
}

// dialViaUpstreamProxy connects through upstream proxy
func (h *Handler) dialViaUpstreamProxy(ctx context.Context, targetAddr string) (net.Conn, error) {
	dialer := &net.Dialer{
		Timeout:   15 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	proxyAddr := h.upstreamURL.Host
	if !strings.Contains(proxyAddr, ":") {
		proxyAddr += ":80"
	}

	conn, err := dialer.DialContext(ctx, "tcp", proxyAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to upstream proxy: %w", err)
	}

	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", targetAddr, targetAddr)

	if h.upstreamURL.User != nil {
		auth := h.upstreamURL.User.String()
		encoded := base64.StdEncoding.EncodeToString([]byte(auth))
		connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", encoded)
	}

	connectReq += "\r\n"

	_, err = conn.Write([]byte(connectReq))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send CONNECT to upstream: %w", err)
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read upstream CONNECT response: %w", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		conn.Close()
		return nil, fmt.Errorf("upstream proxy CONNECT failed: %s", resp.Status)
	}

	return conn, nil
}

// handleHTTP handles plain HTTP proxy requests
func (h *Handler) handleHTTP(w http.ResponseWriter, r *http.Request) {
	if !r.URL.IsAbs() {
		http.Error(w, "This is a proxy server. Send absolute URLs.", http.StatusBadRequest)
		return
	}

	log.Printf("[Proxy] %s %s", r.Method, r.URL.String())

	client, err := transport.NewUTLSClient(h.fingerprint)
	if err != nil {
		http.Error(w, "Failed to create client", http.StatusInternalServerError)
		return
	}

	outReq, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	copyHeaders(outReq.Header, r.Header)

	outReq.Header.Del("Proxy-Connection")
	outReq.Header.Del("Proxy-Authorization")

	resp, err := client.Do(outReq)
	if err != nil {
		log.Printf("[Proxy] Request failed: %v", err)
		http.Error(w, fmt.Sprintf("Request failed: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeaders(w.Header(), resp.Header)

	w.WriteHeader(resp.StatusCode)

	io.Copy(w, resp.Body)
}

// copyHeaders copies headers from src to dst, excluding hop-by-hop headers
func copyHeaders(dst, src http.Header) {
	hopByHop := map[string]bool{
		"Connection":          true,
		"Keep-Alive":          true,
		"Proxy-Authenticate":  true,
		"Proxy-Authorization": true,
		"Proxy-Connection":    true,
		"Te":                  true,
		"Trailer":             true,
		"Transfer-Encoding":   true,
		"Upgrade":             true,
	}

	for key, values := range src {
		if hopByHop[key] {
			continue
		}
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
