package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Engine *gin.Engine
}

func NewServer() *Server {
	return &Server{
		Engine: gin.Default(),
	}
}

func (s *Server) SetupFrontend(buildDir string) {
	absBuildPath, err := filepath.Abs(buildDir)
	if err != nil {
		log.Printf("WARN: Could not resolve absolute path for frontend build directory %s: %v", buildDir, err)
		s.Engine.Any("/app/*path", func(c *gin.Context) {
			c.String(http.StatusServiceUnavailable, "Frontend build path could not be resolved. Check server configuration.")
		})
		return
	}

	indexPath := filepath.Join(absBuildPath, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		log.Printf("WARN: Frontend build directory not found or incomplete at %s. Serving frontend will likely fail: %v", absBuildPath, err)
		s.Engine.Any("/app/*path", func(c *gin.Context) {
			c.String(http.StatusServiceUnavailable, "Frontend not built or incomplete. Run 'npm run build' in the frontend directory.")
		})
		return
	}

	s.Engine.StaticFS("/app", http.Dir(absBuildPath))

	s.Engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/app/") {
			http.ServeFile(c.Writer, c.Request, indexPath)
			return
		}
		c.Status(http.StatusNotFound)
	})

	log.Printf("Frontend serving from: %s", absBuildPath)
}

func (s *Server) Start(port string) error {
	if port == "" {
		port = "8123"
	}
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Server starting on %s", addr)
	return s.Engine.Run(addr)
}
