package strike2

import (
	"context"
	"net/http"
	"time"

	"github.com/Goga74/platform2/internal/transport"
	"github.com/Goga74/platform2/projects/strike2/captcha"
	"github.com/Goga74/platform2/projects/strike2/scraper"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all Strike2 API endpoints under the given router group.
// The router group should be mounted at /api/strike2.
func (s *Strike2) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", s.healthHandler)
	rg.POST("/fetch", s.fetchHandler)

	v1 := rg.Group("/v1")
	{
		v1.POST("/fetch", s.fetchHandler)
		v1.POST("/batch", s.batchHandler)
		v1.GET("/fingerprints", s.fingerprintsHandler)

		captchaGroup := v1.Group("/captcha")
		{
			captchaGroup.POST("/solve/amazon-waf", s.solveAmazonWAFHandler)
			captchaGroup.GET("/balance", s.getCaptchaBalanceHandler)
		}
	}
}

// healthHandler returns Strike2 service health status
func (s *Strike2) healthHandler(c *gin.Context) {
	status := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "strike2",
	}

	if s.captchaSolver != nil {
		status["captcha_enabled"] = true
	} else {
		status["captcha_enabled"] = false
	}

	status["proxy_enabled"] = s.proxyEnabled
	status["auth_enabled"] = s.simpleAuth != nil

	c.JSON(http.StatusOK, status)
}

// fetchHandler handles single URL fetch requests
func (s *Strike2) fetchHandler(c *gin.Context) {
	var req scraper.FetchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "url is required",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	result := s.scraper.FetchURL(ctx, req)

	c.JSON(http.StatusOK, result)
}

// batchHandler handles batch fetch requests
func (s *Strike2) batchHandler(c *gin.Context) {
	var batch scraper.BatchRequest
	if err := c.ShouldBindJSON(&batch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if len(batch.Requests) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "requests array is required and cannot be empty",
		})
		return
	}

	if len(batch.Requests) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "maximum 100 requests per batch",
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	result := s.scraper.FetchBatch(ctx, batch)
	c.JSON(http.StatusOK, result)
}

// fingerprintsHandler returns available fingerprints
func (s *Strike2) fingerprintsHandler(c *gin.Context) {
	fps := transport.GetFingerprints()

	list := make([]gin.H, len(fps))
	for i, fp := range fps {
		list[i] = gin.H{
			"name":       fp.Name,
			"user_agent": fp.UserAgent,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"fingerprints": list,
	})
}

// solveAmazonWAFHandler handles Amazon WAF captcha solving requests
func (s *Strike2) solveAmazonWAFHandler(c *gin.Context) {
	if s.captchaSolver == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "captcha solver not configured - provide CAPTCHA_API_KEY",
		})
		return
	}

	var req captcha.AmazonWAFRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body: " + err.Error(),
		})
		return
	}

	if req.SiteKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "sitekey is required",
		})
		return
	}

	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "url is required",
		})
		return
	}

	result, err := s.captchaSolver.SolveAmazonWAF(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"token":        result.Token,
		"solved_in_ms": result.SolvedIn.Milliseconds(),
		"cost":         result.Cost,
	})
}

// getCaptchaBalanceHandler returns 2Captcha account balance
func (s *Strike2) getCaptchaBalanceHandler(c *gin.Context) {
	if s.captchaSolver == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "captcha solver not configured - provide CAPTCHA_API_KEY",
		})
		return
	}

	balance, err := s.captchaSolver.GetBalance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get balance: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance":  balance,
		"currency": "USD",
	})
}
