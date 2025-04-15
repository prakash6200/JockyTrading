package utils

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

func SaveUploadedFile(file *multipart.FileHeader, destDir string) (string, error) {
	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", err
	}

	// Create a unique filename
	ext := filepath.Ext(file.Filename)
	newFilename := time.Now().Format("20060102150405") + ext
	filePath := filepath.Join(destDir, newFilename)

	// Create destination file
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copy the file content
	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}

	return filePath, nil
}

func GetFileURL(filePath string) string {
	if filePath == "" {
		return ""
	}
	// Adjust this based on your actual file serving setup
	return "/uploads/" + filePath
}
