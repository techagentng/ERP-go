package server

import (
	"errors"
	"os"
	"log"
	"net/http"
	"time"

	"gorm.io/gorm"

	// ratelimit "github.com/JGLTechnologies/gin-rate-limit"
	"github.com/gin-gonic/gin"
	errs "github.com/techagentng/telair-erp/errors"
	"github.com/techagentng/telair-erp/server/response"
	"github.com/techagentng/telair-erp/services/jwt"
)

func (s *Server) Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract access token from header
		accessToken := getTokenFromHeader(c)
		if accessToken == "" {
			respondAndAbort(c, "", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		// Check if the access token is present in the blacklist
		if s.AuthRepository.IsTokenInBlacklist(accessToken) {
			respondAndAbort(c, "Access token is blacklisted", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		// Validate and decode the access token to get the claims
		secret := s.Config.JWTSecret
		accessClaims, err := jwt.ValidateAndGetClaims(accessToken, secret)
		if err != nil {
			respondAndAbort(c, "", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
			return
		}

		// Extract MAC address from claims
		_, isMACAddressUser := accessClaims["mac_address"]
		if isMACAddressUser {
			// Handle the case for MAC address user
			response.JSON(c, "", http.StatusForbidden, nil, errs.New("Forbidden: Access restricted for MAC address users", http.StatusForbidden))
			c.Abort()
			return
		}

		// Extract userID from claims
		userIDValue := accessClaims["id"]
		var userID uint
		switch v := userIDValue.(type) {
		case float64:
			userID = uint(v)
		default:
			respondAndAbort(c, "", http.StatusBadRequest, nil, errs.New("Invalid userID format", http.StatusBadRequest))
			return
		}

		// Retrieve user from service using userID
		user, err := s.AuthRepository.FindUserByID(userID)
		if err != nil {
			switch {
			case errors.Is(err, errs.InActiveUserError):
				respondAndAbort(c, "inactive user", http.StatusUnauthorized, nil, errs.New(err.Error(), http.StatusUnauthorized))
				return
			case errors.Is(err, gorm.ErrRecordNotFound):
				respondAndAbort(c, "user not found", http.StatusUnauthorized, nil, errs.New(err.Error(), http.StatusUnauthorized))
				return
			default:
				respondAndAbort(c, "unable to find entity", http.StatusInternalServerError, nil, errs.New("internal server error", http.StatusInternalServerError))
				return
			}
		}

		// Check if user's email is active
		// if !user.IsEmailActive {
		// 	respondAndAbort(c, "user needs to be verified", http.StatusUnauthorized, nil, errs.New("User needs to be verified", http.StatusUnauthorized))
		// 	return
		// }

		// Set the retrieved user in the context for downstream handlers to access
		c.Set("user", user)
		c.Set("userID", userID)
		c.Set("access_token", accessToken)
		// Continue handling the request
		c.Next()
	}
}

// respondAndAbort calls response.JSON and aborts the Context
func respondAndAbort(c *gin.Context, message string, status int, data interface{}, e *errs.Error) {
	response.JSON(c, message, status, data, e)
	c.Abort()
}

func Logger(inner http.Handler, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		inner.ServeHTTP(w, r)

		log.Printf(
			"%s %s %s %s",
			r.Method,
			r.RequestURI,
			name,
			time.Since(start),
		)
	})
}

// getTokenFromHeader returns the token string in the authorization header
func getTokenFromHeader(c *gin.Context) string {
	authHeader := c.Request.Header.Get("Authorization")
	if len(authHeader) > 8 {
		return authHeader[7:]
	}
	return ""
}

// Middleware for Restricting Access to Protected Routes
func restrictAccessToProtectedRoutes() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if the user is non-credential
		_, exists := c.Get("user")
		if !exists {
			// Check if the user is a MAC address user
			accessToken := getTokenFromHeader(c)
			if accessToken != "" {
				// Validate and decode the access token to get the claims
				secret := os.Getenv("JWT_SECRET")
				accessClaims, err := jwt.ValidateAndGetClaims(accessToken, secret)
				if err == nil {
					_, isMACAddressUser := accessClaims["mac_address"]
					if isMACAddressUser {
						// Handle the case for MAC address user
						response.JSON(c, "", http.StatusForbidden, nil, errs.New("Forbidden: Access restricted for MAC address users", http.StatusForbidden))
						c.Abort()
						return
					}
				}
			}

			// User is non-credential and not a MAC address user, restrict access to protected routes
			restrictedRoutes := []string{"/user/:reportID/like", "/user/:reportID/bookmark"}
			if containsString(restrictedRoutes, c.Request.URL.Path) {
				response.JSON(c, "", http.StatusForbidden, nil, errs.New("Forbidden: Access restricted for non-credential users", http.StatusForbidden))
				c.Abort()
				return
			}
		}

		// User is authenticated or not accessing a protected route, continue with the request
		c.Next()
	}
}

// Function to check if a string exists in a slice of strings
func containsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
