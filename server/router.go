package server

import (
	"fmt"

	// rateLimit "github.com/JGLTechnologies/gin-rate-limit"
	// "net/http"
	"os"
	// "path/filepath"
	// "runtime"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func (s *Server) setupRouter() *gin.Engine {
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "test" {
		r := gin.New()
		s.defineRoutes(r)
		return r
	}

	r := gin.New()
	// r.Static("/static", "./build/static")

	// staticFiles := "server/templates/static"
	// htmlFiles := "server/templates/*.html"
	// if s.Config.Env == "test" {
	// 	_, b, _, _ := runtime.Caller(0)
	// 	basepath := filepath.Dir(b)
	// 	staticFiles = basepath + "/templates/static"
	// 	htmlFiles = basepath + "/templates/*.html"
	// }
	// r.StaticFS("static", http.Dir(staticFiles))
	// r.LoadHTMLGlob(htmlFiles)

	// LoggerWithFormatter middleware will write the logs to gin.DefaultWriter
	// By default gin.DefaultWriter = os.Stdout
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// your custom format
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))
	r.Use(gin.Recovery())

	// allowedOrigins := []string{"http://localhost:3001"}
	// if os.Getenv("GIN_MODE") == "release" {
	// 	allowedOrigins = []string{"https://citizenx-dashboard-sbqx.onrender.com"} 
	// }
	// Use CORS middleware with appropriate configuration
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	r.MaxMultipartMemory = 32 << 20
	s.defineRoutes(r)

	return r
}

func (s *Server) defineRoutes(router *gin.Engine) {
	// store := rateLimit.InMemoryStore(&rateLimit.InMemoryOptions{})
	// limitRate := limitRateForPasswordReset(store)

	apirouter := router.Group("/api/v1")
	apirouter.POST("/auth/signup", s.handleSignup())
	apirouter.POST("/auth/login", s.handleLogin())
	apirouter.POST("/no-cred/login", restrictAccessToProtectedRoutes(), s.handleNonCredentialLogin())

	authorized := apirouter.Group("/")
	authorized.Use(s.Authorize())
	// Upload endpoint
	authorized.GET("/logout", s.handleLogout())
	authorized.GET("/users/online", s.handleGetOnlineUsers())

}
