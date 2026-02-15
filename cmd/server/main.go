package main

import (
	"log"
	"net/http"

	"github.com/Goga74/platform2/internal/common/config"
	"github.com/Goga74/platform2/internal/common/database"
	"github.com/Goga74/platform2/internal/common/swagger"
	"github.com/Goga74/platform2/projects/strike2"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	r := gin.Default()

	// Global health check
	r.GET("/health", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"error":  "database unreachable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":   "healthy",
			"platform": "platform2",
		})
	})

	// Swagger documentation
	swagger.RegisterRoutes(r)

	// Project: Strike2
	strike2Group := r.Group("/api/strike2")
	strike2.RegisterRoutes(strike2Group, db)

	// Future projects:
	// scraperGroup := r.Group("/api/scraper")
	// scraper.RegisterRoutes(scraperGroup, db)

	log.Printf("Platform2 starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
