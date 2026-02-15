package scraper

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Goga74/platform2/internal/transport"
)

// FetchRequest represents a single fetch request
type FetchRequest struct {
	URL         string            `json:"url"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Method      string            `json:"method,omitempty"`
	Body        string            `json:"body,omitempty"`
}

// FetchResponse represents the result of a fetch operation
type FetchResponse struct {
	URL        string            `json:"url"`
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers,omitempty"`
	Body       string            `json:"body"`
	Error      string            `json:"error,omitempty"`
	Duration   int64             `json:"duration_ms"`
}

// BatchRequest represents a batch of fetch requests
type BatchRequest struct {
	Requests []FetchRequest `json:"requests"`
}

// BatchResponse represents results of batch fetch
type BatchResponse struct {
	Results []FetchResponse `json:"results"`
	Total   int             `json:"total"`
	Success int             `json:"success"`
	Failed  int             `json:"failed"`
}

// ScraperService manages concurrent fetch operations
type ScraperService struct {
	workerPool  chan struct{}
	clientCache map[string]*transport.UTLSClient
	cacheMu     sync.RWMutex
}

// NewScraperService creates a new scraper with specified worker pool size
func NewScraperService(poolSize int) *ScraperService {
	return &ScraperService{
		workerPool:  make(chan struct{}, poolSize),
		clientCache: make(map[string]*transport.UTLSClient),
	}
}

// getClient returns cached or creates new uTLS client for fingerprint
func (s *ScraperService) getClient(fingerprintName string) (*transport.UTLSClient, error) {
	s.cacheMu.RLock()
	if client, ok := s.clientCache[fingerprintName]; ok {
		s.cacheMu.RUnlock()
		return client, nil
	}
	s.cacheMu.RUnlock()

	fp := s.findFingerprint(fingerprintName)

	client, err := transport.NewUTLSClient(fp)
	if err != nil {
		return nil, err
	}

	s.cacheMu.Lock()
	s.clientCache[fingerprintName] = client
	s.cacheMu.Unlock()

	return client, nil
}

// findFingerprint finds fingerprint by name or returns default
func (s *ScraperService) findFingerprint(name string) transport.Fingerprint {
	if name == "" {
		return transport.GetRandomFingerprint()
	}

	name = strings.ToLower(name)
	for _, fp := range transport.GetFingerprints() {
		if strings.ToLower(fp.Name) == name {
			return fp
		}
	}

	return transport.GetRandomFingerprint()
}

// FetchURL performs a single URL fetch with fingerprint spoofing
func (s *ScraperService) FetchURL(ctx context.Context, req FetchRequest) FetchResponse {
	start := time.Now()

	select {
	case s.workerPool <- struct{}{}:
		defer func() { <-s.workerPool }()
	case <-ctx.Done():
		return FetchResponse{
			URL:   req.URL,
			Error: "context cancelled while waiting for worker",
		}
	}

	client, err := s.getClient(req.Fingerprint)
	if err != nil {
		return FetchResponse{
			URL:      req.URL,
			Error:    fmt.Sprintf("failed to create client: %v", err),
			Duration: time.Since(start).Milliseconds(),
		}
	}

	method := req.Method
	if method == "" {
		method = "GET"
	}

	var body io.Reader
	if req.Body != "" {
		body = strings.NewReader(req.Body)
	}

	httpReq, err := http.NewRequestWithContext(ctx, method, req.URL, body)
	if err != nil {
		return FetchResponse{
			URL:      req.URL,
			Error:    fmt.Sprintf("failed to create request: %v", err),
			Duration: time.Since(start).Milliseconds(),
		}
	}

	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return FetchResponse{
			URL:      req.URL,
			Error:    fmt.Sprintf("request failed: %v", err),
			Duration: time.Since(start).Milliseconds(),
		}
	}
	defer resp.Body.Close()

	var reader io.Reader = resp.Body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzReader, err := gzip.NewReader(resp.Body)
		if err == nil {
			defer gzReader.Close()
			reader = gzReader
		}
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(reader, 10*1024*1024)) // 10MB limit
	if err != nil {
		return FetchResponse{
			URL:        req.URL,
			StatusCode: resp.StatusCode,
			Error:      fmt.Sprintf("failed to read body: %v", err),
			Duration:   time.Since(start).Milliseconds(),
		}
	}

	headers := make(map[string]string)
	for key := range resp.Header {
		headers[key] = resp.Header.Get(key)
	}

	return FetchResponse{
		URL:        req.URL,
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       string(bodyBytes),
		Duration:   time.Since(start).Milliseconds(),
	}
}

// FetchBatch performs concurrent fetch of multiple URLs
func (s *ScraperService) FetchBatch(ctx context.Context, batch BatchRequest) BatchResponse {
	results := make([]FetchResponse, len(batch.Requests))
	var wg sync.WaitGroup

	for i, req := range batch.Requests {
		wg.Add(1)
		go func(idx int, r FetchRequest) {
			defer wg.Done()
			results[idx] = s.FetchURL(ctx, r)
		}(i, req)
	}

	wg.Wait()

	success, failed := 0, 0
	for _, res := range results {
		if res.Error == "" && res.StatusCode >= 200 && res.StatusCode < 400 {
			success++
		} else {
			failed++
		}
	}

	return BatchResponse{
		Results: results,
		Total:   len(results),
		Success: success,
		Failed:  failed,
	}
}
