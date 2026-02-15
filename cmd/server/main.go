package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Goga74/platform2/internal/common/config"
	"github.com/Goga74/platform2/internal/common/swagger"
	"github.com/Goga74/platform2/projects/strike2"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file FIRST
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: No .env file found")
	} else {
		log.Println("Loaded .env file")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Platform2 - Multi-project backend")

	// Initialize Gin router
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// Platform health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"platform": "platform2",
		})
	})

	// Swagger documentation
	swagger.RegisterRoutes(r)

	// --- Project: Strike2 ---
	s2Cfg := strike2.LoadConfig()
	s2, err := strike2.New(s2Cfg)
	if err != nil {
		log.Fatalf("Failed to initialize Strike2: %v", err)
	}

	// Register Strike2 API routes
	strike2Group := r.Group("/api/strike2")
	s2.RegisterRoutes(strike2Group)

	// Future projects:
	// scraperGroup := r.Group("/api/scraper")
	// scraper.RegisterRoutes(scraperGroup)

	// Wrap gin router with Strike2's combined handler (proxy + API)
	handler := s2.WrapHandler(r)

	// Create HTTP server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Platform2 listening on http://0.0.0.0:%s", cfg.Port)
		log.Printf("Strike2 proxy mode: Use this server as HTTP/HTTPS proxy")
		log.Printf("Strike2 API: http://0.0.0.0:%s/api/strike2/", cfg.Port)
		log.Printf("Swagger: http://0.0.0.0:%s/swagger", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}
