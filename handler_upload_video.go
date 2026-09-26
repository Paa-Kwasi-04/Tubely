package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	// Limit the size of the request body to 1GB
	r.Body = http.MaxBytesReader(w, r.Body, 1<<30)

	// get the video ID from the URL path
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// get the JWT token from the Authorization header
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	// validate the JWT token and get the user ID
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	// get the video metadata from the database
	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video meta data", err)
		return
	}

	// check if the user ID from the JWT matches the user ID from the video metadata
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized access to video", fmt.Errorf("401 Unauthorized"))
		return
	}

	// parse the multipart form data
	const maxMemory = 10 << 20 // 10 MB
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to read file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse Content-Type", err)
		return
	}

	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Wrong file type", fmt.Errorf("400 Bad Request"))
		return
	}

	// Create a temporary file to store the uploaded video
	tempFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating temp file", err)
		return
	} // defer is LIFO
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy the uploaded video to the temporary file
	_, err = io.Copy(tempFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save thumbnail file", err)
		return
	}

	// Reset the file pointer to the beginning of the file
	_, err = tempFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Issue with file read", err)
		return
	}

	// Process the video for fast start using ffmpeg
	fastStartFilePath, err := processVideoForFastStart(tempFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error processing video for fast start", err)
		return
	}
	defer os.Remove(fastStartFilePath) // Clean up the processed file after we're done

	// Open the processed video file for reading
	content, err := os.Open(fastStartFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error opening processed video file", err)
		return
	}
	defer content.Close()

	// Get the aspect ratio of the video
	aspectRatio, err := getVideoAspectRatio(content.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error getting video aspect ratio", err)
		return
	}

	var videoFormat string

	// Determine the video format based on the aspect ratio
	switch aspectRatio {
	case "16:9":
		videoFormat = "landscape"
	case "9:16":
		videoFormat = "portrait"
	default:
		videoFormat = "other"
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate file key", err)
		return
	}
	objectKey := videoFormat + "/" + hex.EncodeToString(key) + "." + strings.Split(mediaType, "/")[1]

	// Upload the processed video to S3
	object := &s3.PutObjectInput{
		Bucket:      &cfg.s3Bucket,
		Key:         &objectKey,
		Body:        content,
		ContentType: &mediaType,
	}

	// Upload the processed video to S3
	_, err = cfg.s3Client.PutObject(r.Context(), object)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to upload object to S3", err)
		return
	}

	// Update the video metadata in the database with the S3 URL
	videoURL := cfg.s3CfDistribution + "/" + objectKey
	video.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video url in db", err)
		return
	}

	// // Generate a signed URL for the video and return it in the response
	// video, err = cfg.DBVideoToSignedVideo(video)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Couldn't generate signed video URL", err)
	// 	return
	// }

	respondWithJSON(w, http.StatusOK, video)
}

// // dbVideoToSignedVideo generates a presigned URL for the video stored in S3 and returns the updated video object.
// func (cfg *apiConfig) DBVideoToSignedVideo(video database.Video) (database.Video, error) {
// 	// If the video URL is empty, return the video as is
// 	if video.VideoURL == nil || *video.VideoURL == "" {
// 		return video, nil
// 	}

// 	// Split the video URL into bucket and key
// 	bucket, key, ok := strings.Cut(*video.VideoURL, ",")
// 	if !ok || bucket == "" || key == "" {
// 		return video, fmt.Errorf("invalid video URL format")
// 	}

// 	url, err := generatePresignedURL(cfg.s3Client, bucket, key, 24*time.Hour)
// 	if err != nil {
// 		return video, fmt.Errorf("failed to generate presigned URL: %w", err)
// 	}
// 	video.VideoURL = &url
// 	return video, nil
// }

// // generatePresignedURL generates a presigned URL for the given S3 object key and bucket.
// func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
// 	// Create a presigned URL for the uploaded video
// 	presignedClient := s3.NewPresignClient(s3Client)

// 	// Generate a presigned URL for the uploaded video
// 	getObjectRequest, err := presignedClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
// 		Bucket: &bucket,
// 		Key:    &key,
// 	}, s3.WithPresignExpires(expireTime))
// 	if err != nil {
// 		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
// 	}
// 	return getObjectRequest.URL, nil
// }
