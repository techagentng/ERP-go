package server

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/techagentng/telair-erp/errors"
	"github.com/techagentng/telair-erp/models"
	"github.com/techagentng/telair-erp/server/response"
	jwtPackage "github.com/techagentng/telair-erp/services/jwt"
)

// Define allowed file types and maximum size
var (
	allowedFileTypes = map[string]bool{
		"video/mp4":  true,
		"video/avi":  true,
		"image/jpeg": true,
		"image/png":  true,
		// Add more allowed types as needed
	}
	maxFileSize int64 = 50 << 20 // 50 MB
)

// Function to validate file types and sizes
func validateFile(fileHeader *multipart.FileHeader) error {
	if _, ok := allowedFileTypes[fileHeader.Header.Get("Content-Type")]; !ok {
		return fmt.Errorf("invalid file type: %s", fileHeader.Filename)
	}
	if fileHeader.Size > maxFileSize {
		return fmt.Errorf("file %s exceeds the maximum allowed size of %d bytes", fileHeader.Filename, maxFileSize)
	}
	return nil
}

// Gin handler for uploading a trailer
func (s *Server) handleUploadTrailer() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get the access token from the authorization header
        accessToken := getTokenFromHeader(c)
        if accessToken == "" {
            logErrorAndRespond(c, "Unauthorized", errors.New("missing token", http.StatusUnauthorized), http.StatusUnauthorized)
            return
        }

        // Validate and decode the access token
        secret := s.Config.JWTSecret
        accessClaims, err := jwtPackage.ValidateAndGetClaims(accessToken, secret)
        if err != nil {
            logErrorAndRespond(c, "Unauthorized", err, http.StatusUnauthorized)
            return
        }

        // Extract userID from accessClaims
        userIDValue, ok := accessClaims["id"]
        if !ok {
            logErrorAndRespond(c, "UserID not found in claims", errors.New("userID missing", http.StatusFailedDependency), http.StatusBadRequest)
            return
        }

        // Convert userIDValue to uint
        var userID uint
        switch v := userIDValue.(type) {
        case float64:
            userID = uint(v)
        default:
            logErrorAndRespond(c, "Invalid userID format", errors.New("invalid userID format", http.StatusUnauthorized), http.StatusBadRequest)
            return
        }

        // Parse multipart form data
        if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
            logErrorAndRespond(c, "Failed to parse form data", err, http.StatusBadRequest)
            return
        }

        // Proceed with S3 upload logic (as you already have)
        s3Client, err := createS3Client()
        if err != nil {
            logErrorAndRespond(c, "Failed to create S3 client", err, http.StatusInternalServerError)
            return
        }

        // Upload files and get URLs
        videoURLs, pictureURLs, err := uploadTrailerFiles(c, s3Client, "videos")
        if err != nil {
            logErrorAndRespond(c, "Failed to upload files", err, http.StatusInternalServerError)
            return
        }

        videoURLsStr := strings.Join(videoURLs, ",")
        pictureURLsStr := strings.Join(pictureURLs, ",")

        // Process form fields
        trailer, err := createTrailerFromForm(c, userID, videoURLsStr, pictureURLsStr)
        if err != nil {
            logErrorAndRespond(c, "Failed to process trailer data", err, http.StatusBadRequest)
            return
        }

        // Save trailer to the database
        if err := s.MovieRepository.CreateTrailer(&trailer); err != nil {
            logErrorAndRespond(c, "Failed to create trailer", err, http.StatusInternalServerError)
            return
        }

        response.JSON(c, "Trailer uploaded successfully", http.StatusCreated, trailer, nil)
    }
}

// Mock in-memory store for upload progress tracking (replace with a persistent store in production)
var uploadProgressStore = make(map[string]*models.UploadProgress)

// Function to track progress for a specific user upload session
func trackProgress(sessionID string, uploadedFiles int, totalFiles int) {
	progress := uploadProgressStore[sessionID]
	if progress == nil {
		progress = &models.UploadProgress{TotalFiles: totalFiles}
		uploadProgressStore[sessionID] = progress
	}
	progress.UploadedFiles = uploadedFiles
	progress.Percentage = float64(uploadedFiles) / float64(totalFiles) * 100

	// You could broadcast this progress to the client using WebSockets or a similar mechanism
}

func uploadTrailerFiles(c *gin.Context, s3Client *s3.Client, folder string) (videoURLs []string, pictureURLs []string, err error) {
    sessionID := uuid.New().String()
    // Handle video file upload concurrently
    videoFilePaths, err := uploadFilesConcurrently(c.Request.MultipartForm.File["videos"], folder+"/videos", s3Client, sessionID)
    if err != nil {
        return nil, nil, err
    }
    videoURLs = videoFilePaths

    // Handle picture file upload concurrently
    pictureFilePaths, err := uploadFilesConcurrently(c.Request.MultipartForm.File["pictures"], folder+"/pictures", s3Client, sessionID)
    if err != nil {
        return nil, nil, err
    }
    pictureURLs = pictureFilePaths

    return videoURLs, pictureURLs, nil
}

// Helper function to extract userID from context
func getUserIDFromContext(c *gin.Context) (uint, error) {
    userIDCtx, exists := c.Get("userID")
    if !exists {
        return 0, errors.New("userID not found in context", errors.ErrBadRequest.Status)
    }

    userID, ok := userIDCtx.(uint)
    if !ok {
        return 0, errors.New("invalid userID format", errors.ErrBadRequest.Status)
    }
    return userID, nil
}

// Function to upload files concurrently
func uploadFilesConcurrently(files []*multipart.FileHeader, folder string, s3Client *s3.Client, sessionID string) ([]string, error) {
    var filePaths []string
    var wg sync.WaitGroup
    var uploadErr error
    var mu sync.Mutex

    totalFiles := len(files)

    for i, file := range files {
        wg.Add(1)
        go func(file *multipart.FileHeader, index int) {
            defer wg.Done()

            // Open the file
            fileReader, err := file.Open()
            if err != nil {
                mu.Lock() // Protect shared variable
                uploadErr = err
                mu.Unlock()
                return
            }
            defer fileReader.Close()

            // Construct the filename for S3
            filename := fmt.Sprintf("%s/%s", folder, file.Filename)
            filePath, err := uploadFileToS3(s3Client, fileReader, os.Getenv("AWS_BUCKET"), filename)
            if err != nil {
                mu.Lock() // Protect shared variable
                uploadErr = err
                mu.Unlock()
                return
            }

            // Store the uploaded file path
            mu.Lock() // Protect shared variable
            filePaths = append(filePaths, filePath)
            mu.Unlock()

            // Track progress
            trackProgress(sessionID, index+1, totalFiles)
        }(file, i)
    }

    // Wait for all goroutines to finish
    wg.Wait()

    // Return the uploaded file paths and any error that occurred
    return filePaths, uploadErr
}


// Helper function to create trailer from form data
func createTrailerFromForm(c *gin.Context, userID uint, videoURLs, pictureURLs string) (models.Trailer, error) {
    durationStr := c.PostForm("duration")
    duration, err := strconv.Atoi(durationStr)
    if err != nil {
        log.Printf("Invalid duration value: %v", err)
        duration = 0
    }

    trailer := models.Trailer{
        MovieBase: models.MovieBase{
            Title:       c.PostForm("title"),
            Description: c.PostForm("description"),
            Duration:    duration,
        },
        LogLine:     c.PostForm("log_line"),
        ProductYear: c.PostForm("product_year"),
        Star1:       c.PostForm("star1"),
        Star2:       c.PostForm("star2"),
        Star3:       c.PostForm("star3"),
        VideoURLs:   videoURLs,
        PictureURLs: pictureURLs,
        UserID:      userID,
    }

    return trailer, nil
}

// Helper function to log errors and send JSON response
func logErrorAndRespond(c *gin.Context, message string, err error, statusCode int) {
    log.Printf("%s: %v", message, err)
    response.JSON(c, "", statusCode, nil, fmt.Errorf("%s: %w", message, err))
}

var (
	// To store progress of uploads
	progressMap = make(map[string]int)
	mu          sync.Mutex
)

func (s *Server) checkUploadProgress() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Get the sessionID from the URL parameters
        sessionID := c.Param("sessionID")

        // Lock the mutex to safely access the progress map
        mu.Lock()
        defer mu.Unlock()

        // Get the progress value from the map
        progress, exists := progressMap[sessionID]
        if !exists {
            // If the sessionID does not exist, return a 404 status
            c.JSON(http.StatusNotFound, gin.H{"error": "Session ID not found"})
            return
        }

        // Return the progress value as a JSON response
        c.JSON(http.StatusOK, gin.H{"sessionID": sessionID, "progress": progress})
    }
}



