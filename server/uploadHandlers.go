package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings" 
	"github.com/gin-gonic/gin"
	"github.com/techagentng/telair-erp/models"
	"github.com/techagentng/telair-erp/server/response"
)

// func getContentType(fileName string) string {
//     ext := filepath.Ext(fileName)
//     switch ext {
//     case ".jpg", ".jpeg":
//         return "image/jpeg"
//     case ".png":
//         return "image/png"
//     case ".gif":
//         return "image/gif"
//     case ".mp4":
//         return "video/mp4"
//     case ".avi":
//         return "video/x-msvideo"
//     default:
//         return "application/octet-stream"
//     }
// }

// func uploadFileToS3t(client *s3.S3, file multipart.File, bucketName, key, region string) (string, error) {
//     defer file.Close()

//     // Read the file content
//     fileContent, err := io.ReadAll(file)
//     if err != nil {
//         return "", fmt.Errorf("failed to read file content: %v", err)
//     }

//     // Upload the file to S3
//     _, err = client.PutObject(&s3.PutObjectInput{
//         Bucket:      aws.String(bucketName),
//         Key:         aws.String(key),
//         Body:        bytes.NewReader(fileContent),
//         ContentType: aws.String(getContentType(key)), 
// 		ACL:         aws.String("public-read"),
//     })
//     if err != nil {
//         return "", fmt.Errorf("failed to upload file to S3: %v", err)
//     }

//     // Return the S3 URL of the uploaded file
//     fileURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucketName, region, key)
//     return fileURL, nil
// }


func (s *Server) handleUploadTrailer() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Parse multipart form data with a 50 MB limit
        if err := c.Request.ParseMultipartForm(50 << 20); err != nil {
            log.Printf("Failed to parse form data: %v", err)
            response.JSON(c, "", http.StatusBadRequest, nil, fmt.Errorf("failed to parse form data: %w", err))
            return
        }

        // Get userID from context
        userIDCtx, exists := c.Get("userID")
        if !exists {
            log.Println("Unauthorized: userID not found in context")
            response.JSON(c, "", http.StatusUnauthorized, nil, fmt.Errorf("unauthorized"))
            return
        }

        // Assert userID as uint
        userID, ok := userIDCtx.(uint)
        if !ok {
            log.Println("Invalid userID format in context")
            response.JSON(c, "", http.StatusBadRequest, nil, fmt.Errorf("invalid userID format"))
            return
        }

        // Initialize file paths for S3
        var videoFilePaths []string
        var pictureFilePaths []string

        // Process video files
        videoFiles := c.Request.MultipartForm.File["videos"]
        for _, videoFile := range videoFiles {
            file, err := videoFile.Open()
            if err != nil {
                log.Printf("Failed to open video file: %v", err)
                response.JSON(c, "", http.StatusInternalServerError, nil, fmt.Errorf("failed to open video file: %w", err))
                return
            }
            defer file.Close()

            // Generate unique filename and upload to S3
            filename := fmt.Sprintf("videos/%s", videoFile.Filename)
            s3Client, err := createS3Client()
            if err != nil {
                log.Printf("Failed to create S3 client: %v", err)
                response.JSON(c, "", http.StatusInternalServerError, nil, fmt.Errorf("failed to create S3 client: %w", err))
                return
            }

            filePath, err := uploadFileToS3(s3Client, file, os.Getenv("AWS_BUCKET"), filename)
            if err != nil {
                log.Printf("Failed to upload video file to S3: %v", err)
                response.JSON(c, "", http.StatusInternalServerError, nil, fmt.Errorf("failed to upload video file to S3: %w", err))
                return
            }

            videoFilePaths = append(videoFilePaths, filePath)
        }

        // Process picture files
        pictureFiles := c.Request.MultipartForm.File["pictures"]
        for _, pictureFile := range pictureFiles {
            file, err := pictureFile.Open()
            if err != nil {
                log.Printf("Failed to open picture file: %v", err)
                response.JSON(c, "", http.StatusInternalServerError, nil, fmt.Errorf("failed to open picture file: %w", err))
                return
            }
            defer file.Close()

            // Generate unique filename and upload to S3
            filename := fmt.Sprintf("pictures/%s", pictureFile.Filename)
            s3Client, err := createS3Client()
            if err != nil {
                log.Printf("Failed to create S3 client: %v", err)
                response.JSON(c, "", http.StatusInternalServerError, nil, fmt.Errorf("failed to create S3 client: %w", err))
                return
            }

            filePath, err := uploadFileToS3(s3Client, file, os.Getenv("AWS_BUCKET"), filename)
            if err != nil {
                log.Printf("Failed to upload picture file to S3: %v", err)
                response.JSON(c, "", http.StatusInternalServerError, nil, fmt.Errorf("failed to upload picture file to S3: %w", err))
                return
            }
            pictureFilePaths = append(pictureFilePaths, filePath)
        }

        // Convert video and picture file paths to comma-separated strings
        videoURLsStr := strings.Join(videoFilePaths, ",")
        pictureURLsStr := strings.Join(pictureFilePaths, ",")

        // Process other form fields
        logLine := c.PostForm("log_line")
        productYear := c.PostForm("product_year")

        // Retrieve stars from form
        star1 := c.PostForm("star1")
        star2 := c.PostForm("star2")
        star3 := c.PostForm("star3")

        // Convert duration from string to int
        durationStr := c.PostForm("duration")
        duration, err := strconv.Atoi(durationStr)
        if err != nil {
            log.Printf("Invalid duration value: %v", err)
            duration = 0
        }

        // Create and populate Trailer struct
        trailer := models.Trailer{
            MovieBase: models.MovieBase{
                Title:       c.PostForm("title"),
                Description: c.PostForm("description"),
                Duration:    duration,
            },
            LogLine:     logLine,
            ProductYear: productYear,
            Star1:       star1,
            Star2:       star2,
            Star3:       star3,
            VideoURLs:   videoURLsStr,
            PictureURLs: pictureURLsStr,
            UserID:      userID,
        }

        // Save trailer to database
        if err := s.MovieRepository.CreateTrailer(&trailer); err != nil {
            log.Printf("Failed to create trailer: %v", err)
            response.JSON(c, "", http.StatusInternalServerError, nil, fmt.Errorf("failed to create trailer: %w", err))
            return
        }

        response.JSON(c, "Trailer uploaded successfully", http.StatusCreated, trailer, nil)
    }
}





