package swagger

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>Platform2 API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        SwaggerUIBundle({
            url: '/swagger/openapi.yaml',
            dom_id: '#swagger-ui',
            presets: [
                SwaggerUIBundle.presets.apis,
                SwaggerUIBundle.SwaggerUIStandalonePreset
            ],
            layout: "BaseLayout"
        });
    </script>
</body>
</html>`

func RegisterRoutes(r *gin.Engine) {
	r.GET("/swagger", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIHTML))
	})

	r.GET("/swagger/openapi.yaml", func(c *gin.Context) {
		yamlPath := findOpenAPISpec()
		if yamlPath == "" {
			c.String(http.StatusNotFound, "openapi.yaml not found")
			return
		}
		c.File(yamlPath)
	})
}

func findOpenAPISpec() string {
	paths := []string{
		"api/openapi.yaml",
		"/app/api/openapi.yaml",
	}

	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "api/openapi.yaml"))
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
