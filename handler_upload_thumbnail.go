package main

import (
	// "encoding/base64"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")

	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not read random bytes", err)
		return
	}
	randomURL := base64.RawURLEncoding.EncodeToString(key)
	fmt.Println(randomURL)

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

	const maxMemory = 10 << 20
	r.ParseMultipartForm(maxMemory)

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
		return
	}

	fmt.Println(mediaType)

	var fileExtension string

	if contentType == "image/png" {
		fileExtension = ".png"
	} else if contentType == "image/jpeg" {
		fileExtension = ".jpeg"
	} else {
		respondWithError(w, http.StatusBadRequest, "Not a jpeg or a png", err)
		return
	}

	// data, err := io.ReadAll(file)
	// if err != nil {
	// 	respondWithError(w, http.StatusBadRequest, "Unable to parse form file", err)
	// 	return
	// }

	videoData, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Unable to get video metadata", err)
		return
	}

	if videoData.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Authenticated user is not video owner", err)
		return
	}

	// encodedData := base64.StdEncoding.EncodeToString(data)
	// thumbnail_url := fmt.Sprintf("data:%s;base64,%s", contentType, encodedData)
	// videoData.ThumbnailURL = &thumbnail_url

	// fileURL := fmt.Sprintf("/assets/%s.%s", videoID, fileExtension)

	fileName := randomURL + fileExtension
	fileURL := filepath.Join(cfg.assetsRoot, fileName)

	thumbnailFile, err := os.Create(fileURL)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Could not save to file url", err)
		return
	}
	defer thumbnailFile.Close()

	_, err = file.Seek(0, 0)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not reset file position", err)
		return
	}

	bytesCopied, err := io.Copy(thumbnailFile, file)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Copy to file failed", err)
		return
	}

	fmt.Println("Copied bytes", bytesCopied)
	fmt.Println(fileURL)

	thumbnail_url := fmt.Sprintf("http://localhost:%s/assets/%s%s", cfg.port, randomURL, fileExtension)

	fmt.Println(thumbnail_url)

	videoData.ThumbnailURL = &thumbnail_url

	err = cfg.db.UpdateVideo(videoData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, videoData)
}
