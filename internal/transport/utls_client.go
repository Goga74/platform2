package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// UTLSClient wraps HTTP client with uTLS for JA3 fingerprint spoofing.
// It correctly handles both HTTP/1.1 and HTTP/2 protocols.
type UTLSClient struct {
	transport   *utlsRoundTripper
	fingerprint Fingerprint
}

// utlsRoundTripper implements http.RoundTripper with proper HTTP/2 support over uTLS.
//
// PROBLEM SOLVED:
// Standard http.Transport with http2.ConfigureTransport() doesn't work correctly
// when using custom DialTLSContext. The issue is that http2.ConfigureTransport
// expects to manage TLS itself, but we pre-establish TLS via uTLS.
//
// When the server responds with HTTP/2 frames, http.Transport fails with:
// "malformed HTTP response \x00\x00\x12\x04..."
// because it tries to parse binary HTTP/2 frames as HTTP/1.x text.
//
// SOLUTION:
// We create a custom RoundTripper that:
// 1. Establishes uTLS connection with JA3 spoofing
// 2. Checks negotiated ALPN protocol after handshake
// 3. Uses http2.Transport for h2 connections
// 4. Uses http.Transport for http/1.1 connections
type utlsRoundTripper struct {
	fingerprint    Fingerprint
	http1Transport *http.Transport

	// Connection pool for HTTP/2 - we reuse http2.ClientConn per host
	h2ConnPool   map[string]*http2.ClientConn
	h2ConnPoolMu sync.RWMutex

	// Configuration
	dialer     *net.Dialer
	tlsTimeout time.Duration
}

// newUTLSRoundTripper creates a new round tripper with uTLS support
func newUTLSRoundTripper(fp Fingerprint) *utlsRoundTripper {
	rt := &utlsRoundTripper{
		fingerprint: fp,
		h2ConnPool:  make(map[string]*http2.ClientConn),
		dialer: &net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		},
		tlsTimeout: 10 * time.Second,
	}

	// HTTP/1.1 transport for non-H2 connections
	rt.http1Transport = &http.Transport{
		DialContext:           rt.dialer.DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:  rt.tlsTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return rt
}

// dialUTLS establishes a uTLS connection with JA3 fingerprint spoofing
func (rt *utlsRoundTripper) dialUTLS(ctx context.Context, network, addr string) (*utls.UConn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	conn, err := rt.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, fmt.Errorf("TCP dial failed: %w", err)
	}

	uConn := utls.UClient(conn, &utls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
	}, rt.fingerprint.ID)

	if deadline, ok := ctx.Deadline(); ok {
		uConn.SetDeadline(deadline)
	} else {
		uConn.SetDeadline(time.Now().Add(rt.tlsTimeout))
	}

	if err := uConn.Handshake(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("TLS handshake failed: %w", err)
	}

	uConn.SetDeadline(time.Time{})

	return uConn, nil
}

// RoundTrip implements http.RoundTripper
func (rt *utlsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" {
		return rt.http1Transport.RoundTrip(req)
	}

	addr := req.URL.Host
	if !strings.Contains(addr, ":") {
		addr += ":443"
	}

	rt.h2ConnPoolMu.RLock()
	h2Conn, exists := rt.h2ConnPool[addr]
	rt.h2ConnPoolMu.RUnlock()

	if exists && h2Conn.CanTakeNewRequest() {
		resp, err := h2Conn.RoundTrip(req)
		if err == nil {
			return resp, nil
		}
		rt.h2ConnPoolMu.Lock()
		delete(rt.h2ConnPool, addr)
		rt.h2ConnPoolMu.Unlock()
	}

	ctx := req.Context()
	uConn, err := rt.dialUTLS(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	negotiatedProto := uConn.ConnectionState().NegotiatedProtocol

	log.Printf("[uTLS] %s -> Protocol: %s", addr, negotiatedProto)

	switch negotiatedProto {
	case "h2":
		return rt.roundTripH2(req, uConn, addr)
	case "http/1.1", "":
		return rt.roundTripH1(req, uConn)
	default:
		uConn.Close()
		return nil, fmt.Errorf("unsupported protocol: %s", negotiatedProto)
	}
}

// roundTripH2 handles HTTP/2 requests
func (rt *utlsRoundTripper) roundTripH2(req *http.Request, conn *utls.UConn, addr string) (*http.Response, error) {
	h2Transport := &http2.Transport{
		AllowHTTP:       false,
		ReadIdleTimeout: 30 * time.Second,
		PingTimeout:     15 * time.Second,
	}

	h2Conn, err := h2Transport.NewClientConn(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create HTTP/2 connection: %w", err)
	}

	rt.h2ConnPoolMu.Lock()
	rt.h2ConnPool[addr] = h2Conn
	rt.h2ConnPoolMu.Unlock()

	return h2Conn.RoundTrip(req)
}

// roundTripH1 handles HTTP/1.1 requests over existing TLS connection
func (rt *utlsRoundTripper) roundTripH1(req *http.Request, conn *utls.UConn) (*http.Response, error) {
	transport := &http.Transport{
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return conn, nil
		},
		DisableKeepAlives:     false,
		MaxIdleConns:          1,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return resp, nil
}

// NewUTLSClient creates a new client with specified fingerprint
func NewUTLSClient(fp Fingerprint) (*UTLSClient, error) {
	rt := newUTLSRoundTripper(fp)

	return &UTLSClient{
		transport:   rt,
		fingerprint: fp,
	}, nil
}

// Get performs HTTP GET request with spoofed fingerprint
func (c *UTLSClient) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	c.setDefaultHeaders(req)

	return c.Do(req)
}

// Post performs HTTP POST request with spoofed fingerprint
func (c *UTLSClient) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.fingerprint.UserAgent)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "*/*")

	return c.Do(req)
}

// Do performs custom HTTP request with timeout
func (c *UTLSClient) Do(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.fingerprint.UserAgent)
	}

	ctx := req.Context()
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		req = req.WithContext(ctx)
	}

	return c.transport.RoundTrip(req)
}

// setDefaultHeaders sets browser-like headers
func (c *UTLSClient) setDefaultHeaders(req *http.Request) {
	req.Header.Set("User-Agent", c.fingerprint.UserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="120", "Google Chrome";v="120", "Not?A_Brand";v="99"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}
