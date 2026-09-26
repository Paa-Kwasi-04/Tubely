package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here
	// Parse the multipart form data
	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	// Get the file from the form data
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	// Get the file extension from the Content-Type header
	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse Content-Type", err)
		return
	}

	if mediaType != "image/jpeg" && mediaType != "image/png" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", err)
		return
	}

	extension := strings.Split(mediaType, "/")[1]

	// Get the video metadata from the database
	metadata, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get video meta data", err)
		return
	}

	// Check if the user is authorized to upload the thumbnail
	if metadata.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "UnAuthorized User", err)
		return
	}

	// Generate a random filename for the thumbnail
	key := make([]byte, 32)
	rand.Read(key)
	encodedStr := base64.RawURLEncoding.EncodeToString(key)

	// Create the file path for the thumbnail
	filePath := fmt.Sprintf("%s.%s", filepath.Join(cfg.assetsRoot, encodedStr), extension)

	// Create the file on disk
	videoFile, err := os.Create(filePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to get save thumbnail file", err)
		return
	}
	defer videoFile.Close()

	// Write the file to disk
	if _, err = io.Copy(videoFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to save thumbnail file", err)
		return
	}

	// Update the video metadata with the thumbnail URL
	ThumbnailURL := "http://localhost:" + cfg.port + "/assets/" + encodedStr + "." + extension
	metadata.ThumbnailURL = &ThumbnailURL

	// Update the video metadata in the database
	err = cfg.db.UpdateVideo(metadata)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to update thumbnail url", err)
		return
	}

	// Respond with the updated video metadata
	respondWithJSON(w, http.StatusOK, metadata)
}
