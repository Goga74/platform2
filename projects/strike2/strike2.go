package strike2

import (
	"log"
	"net/http"

	"github.com/Goga74/platform2/projects/strike2/auth"
	"github.com/Goga74/platform2/projects/strike2/captcha"
	"github.com/Goga74/platform2/projects/strike2/proxy"
	"github.com/Goga74/platform2/projects/strike2/scraper"
)

// Strike2 is the main Strike2 project instance
type Strike2 struct {
	proxyHandler  *proxy.Handler
	proxyEnabled  bool
	simpleAuth    *auth.SimpleAuth
	scraper       *scraper.ScraperService
	captchaSolver *captcha.Solver
}

// Config holds Strike2 initialization parameters
type Config struct {
	CaptchaAPIKey string
	UpstreamProxy string
	Fingerprint   string
	ProxyToken    string
	Workers       int
}

// New creates and initializes a Strike2 instance
func New(cfg Config) (*Strike2, error) {
	workers := cfg.Workers
	if workers <= 0 {
		workers = 500
	}

	s := &Strike2{
		scraper: scraper.NewScraperService(workers),
	}

	// Initialize simple authentication
	if cfg.ProxyToken != "" {
		s.simpleAuth = auth.NewSimpleAuth(cfg.ProxyToken)
		log.Printf("[Strike2] Proxy authentication: ENABLED")
	} else {
		log.Printf("[Strike2] Proxy authentication: DISABLED (no PROXY_TOKEN set)")
	}

	// Initialize captcha solver
	if cfg.CaptchaAPIKey != "" {
		s.captchaSolver = captcha.NewSolver(cfg.CaptchaAPIKey)
		log.Printf("[Strike2] 2Captcha integration: ENABLED")
	} else {
		log.Printf("[Strike2] 2Captcha integration: DISABLED (no API key)")
	}

	// Initialize proxy handler
	proxyConfig := &proxy.Config{
		UpstreamProxy: cfg.UpstreamProxy,
		Fingerprint:   cfg.Fingerprint,
	}
	proxyHandler, err := proxy.NewHandler(proxyConfig)
	if err != nil {
		log.Printf("[Strike2] Warning: Failed to initialize proxy handler: %v", err)
	} else {
		s.proxyHandler = proxyHandler
		s.proxyEnabled = true
		log.Printf("[Strike2] Proxy mode: ENABLED")
		if cfg.UpstreamProxy != "" {
			log.Printf("[Strike2] Upstream proxy: configured")
		}
	}

	return s, nil
}

// WrapHandler returns a combined HTTP handler that routes between
// the API router (gin) and the proxy handler.
// CONNECT requests and absolute-URL requests go to the proxy.
// Everything else goes to the API router.
func (s *Strike2) WrapHandler(apiRouter http.Handler) http.Handler {
	return &combinedHandler{
		apiRouter:    apiRouter,
		proxyHandler: s.proxyHandler,
		simpleAuth:   s.simpleAuth,
	}
}

// combinedHandler routes requests between API and proxy handlers
type combinedHandler struct {
	apiRouter    http.Handler
	proxyHandler *proxy.Handler
	simpleAuth   *auth.SimpleAuth
}

func (h *combinedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CONNECT method is always proxy
	if r.Method == http.MethodConnect {
		if h.proxyHandler != nil {
			if h.simpleAuth != nil && !h.simpleAuth.ValidateProxyAuth(r) {
				auth.Reject407(w)
				return
			}
			h.proxyHandler.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Proxy not enabled", http.StatusServiceUnavailable)
		return
	}

	// Absolute URL = proxy request (e.g., GET http://example.com/path)
	if r.URL.IsAbs() && h.proxyHandler != nil {
		if h.simpleAuth != nil && !h.simpleAuth.ValidateProxyAuth(r) {
			auth.Reject407(w)
			return
		}
		h.proxyHandler.ServeHTTP(w, r)
		return
	}

	// Everything else → API router (gin)
	h.apiRouter.ServeHTTP(w, r)
}
