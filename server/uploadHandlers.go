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
            logErrorAndRespond(c, "Unauthorized - Missing Token", errors.New("missing token", http.StatusBadGateway), http.StatusUnauthorized)
            return
        }

        // Log the access token retrieved from the header
        log.Println("Authorization Header:", c.Request.Header.Get("Authorization"))

        // Validate and decode the access token
        secret := s.Config.JWTSecret
        accessClaims, err := jwtPackage.ValidateAndGetClaims(accessToken, secret)
        if err != nil {
            log.Println("Token Validation Error:", err)
            logErrorAndRespond(c, "Unauthorized - Token Validation Failed", err, http.StatusUnauthorized)
            return
        }

        // Log claims to verify successful decoding
        log.Println("Access Claims:", accessClaims)

        // Extract userID from accessClaims
        userIDValue, ok := accessClaims["id"]
        if !ok {
            logErrorAndRespond(c, "UserID not found in claims", errors.New("userID missing", http.StatusBadGateway), http.StatusBadRequest)
            return
        }

        // Log user ID value before conversion
        log.Println("UserID Value from Claims:", userIDValue)

        // Convert userIDValue to uint
        var userID uint
        switch v := userIDValue.(type) {
        case float64:
            userID = uint(v)
        default:
            logErrorAndRespond(c, "Invalid userID format", errors.New("invalid userID format", http.StatusBadGateway), http.StatusBadRequest)
            return
        }

        // Log the converted userID
        log.Println("Converted UserID:", userID)

        // Parse multipart form data
        if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
            log.Println("Form Parsing Error:", err)
            logErrorAndRespond(c, "Failed to parse form data", err, http.StatusBadRequest)
            return
        }

        // Proceed with S3 upload logic
        s3Client, err := createS3Client()
        if err != nil {
            log.Println("S3 Client Creation Error:", err)
            logErrorAndRespond(c, "Failed to create S3 client", err, http.StatusInternalServerError)
            return
        }

        // Log S3 client creation success
        log.Println("S3 Client Created Successfully")
        sessionID := uuid.New().String()
        // Upload files and get URLs
        videoURLs, pictureURLs, err := uploadTrailerFiles(c, s3Client, "videos")
        if err != nil {
            log.Println("File Upload Error:", err)
            logErrorAndRespond(c, "Failed to upload files", err, http.StatusInternalServerError)
            return
        }

        // Log the uploaded URLs
        log.Println("Video URLs:", videoURLs)
        log.Println("Picture URLs:", pictureURLs)

        videoURLsStr := strings.Join(videoURLs, ",")
        pictureURLsStr := strings.Join(pictureURLs, ",")

        // Process form fields
        trailer, err := createTrailerFromForm(c, userID, videoURLsStr, pictureURLsStr)
        if err != nil {
            log.Println("Trailer Processing Error:", err)
            logErrorAndRespond(c, "Failed to process trailer data", err, http.StatusBadRequest)
            return
        }

        // Log the created trailer
        log.Println("Trailer Created:", trailer)

        // Save trailer to the database
        if err := s.MovieRepository.CreateTrailer(&trailer); err != nil {
            log.Println("Database Insertion Error:", err)
            logErrorAndRespond(c, "Failed to create trailer", err, http.StatusInternalServerError)
            return
        }

        // Log the successful trailer creation
        log.Println("Trailer Uploaded Successfully")

        response.JSON(c, "Trailer uploaded successfully", http.StatusCreated, gin.H{
            "trailer": trailer,
            "sessionID": sessionID,  
        }, nil)
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

func (s *Server) getUploadProgress() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Query("sessionID")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing sessionID"})
			return
		}

		progress, exists := uploadProgressStore[sessionID]
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "No progress found for sessionID"})
			return
		}

		c.JSON(http.StatusOK, progress)
	}
}



