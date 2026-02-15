package strike2

import (
	"net/http"

	"github.com/Goga74/platform2/internal/common/database"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all Strike2 endpoints under the given router group.
// The router group should be mounted at /api/strike2.
func RegisterRoutes(rg *gin.RouterGroup, db *database.DB) {
	rg.GET("/health", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "unhealthy",
				"project": "strike2",
				"error":   "database unreachable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"project": "strike2",
		})
	})

	// TODO: Migrate full Strike2 implementation from igorzamiatin-hireright/strike2
	// Endpoints to be added:
	// - POST /check       - Submit a check request
	// - GET  /check/:id   - Get check result
	// - GET  /tokens      - List API tokens
	// - POST /tokens      - Create API token
}
