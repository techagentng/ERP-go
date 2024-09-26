package server

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"time"
    
	"github.com/gin-gonic/gin"
	errs "github.com/techagentng/telair-erp/errors"
	"github.com/techagentng/telair-erp/models"
	"github.com/techagentng/telair-erp/server/response"
	"github.com/techagentng/telair-erp/services/jwt"
	"gorm.io/gorm"
)

func (s *Server) Authorize() gin.HandlerFunc {
    return func(c *gin.Context) {
        accessToken := getTokenFromHeader(c)
        if accessToken == "" {
            respondAndAbort(c, "", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
            return
        }

        if s.AuthRepository.IsTokenInBlacklist(accessToken) {
            respondAndAbort(c, "Access token is blacklisted", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
            return
        }

        secret := s.Config.JWTSecret
        accessClaims, err := jwt.ValidateAndGetClaims(accessToken, secret)
        if err != nil {
            respondAndAbort(c, "", http.StatusUnauthorized, nil, errs.New("Unauthorized", http.StatusUnauthorized))
            return
        }

        userIDValue := accessClaims["id"]
        var userID uint
        switch v := userIDValue.(type) {
        case float64:
            userID = uint(v)
        default:
            respondAndAbort(c, "", http.StatusBadRequest, nil, errs.New("Invalid userID format", http.StatusBadRequest))
            return
        }

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

				// Extract role from claims
				role, ok := accessClaims["role"].(string)
				if !ok {
					respondAndAbort(c, "invalid role information", http.StatusBadRequest, nil, errs.New("Invalid role in token", http.StatusBadRequest))
					return
				}

        c.Set("user", user)
        c.Set("userID", userID)
        c.Set("access_token", accessToken)
        c.Set("fullName", user.Fullname)
        c.Set("username", user.Username)
		c.Set("profile_image", user.ThumbNailURL)
		c.Set("user_role", role)
		// Log to check if values are set
		log.Printf("Username in middleware: %v", c.Value("username"))
		log.Printf("FullName in middleware: %v", c.Value("fullName"))
		log.Printf("Profile image in middleware: %v", c.Value("profileImage"))
        c.Next()
    }
}

func keyFunc(c *gin.Context) string {
	//TODO Handle when email isn't sent successfully in any of the three tries
	//b1, err := c.Request.GetBody()
	buf, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		response.JSON(c, "", http.StatusBadRequest, nil, err)
		return ""
	}

	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(buf))

	var foundUser models.ForgotPassword
	err = decode(c, &foundUser)
	if err != nil {
		response.JSON(c, "", http.StatusBadRequest, nil, err)
		return ""
	}

	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	return foundUser.Email
}

func keyFuncMacAddress(c *gin.Context) string {
	// Extract MAC address from the request
	macAddress := c.PostForm("mac_address")
	return macAddress
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

// Function to check if a string exists in a slice of strings
func containsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
