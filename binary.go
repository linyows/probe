package probe

import (
	"crypto/rand"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// IsTextualMimeType determines if the given MIME type represents textual data
func IsTextualMimeType(contentType string) bool {
	if contentType == "" {
		return true // Default to text if no content type
	}

	// Extract the main MIME type (before semicolon if present)
	mimeType := strings.Split(contentType, ";")[0]
	mimeType = strings.TrimSpace(strings.ToLower(mimeType))

	// Textual MIME types
	textualTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-javascript",
		"application/x-www-form-urlencoded",
		"application/xhtml+xml",
		"application/rss+xml",
		"application/atom+xml",
		"application/ld+json",
	}

	for _, textType := range textualTypes {
		if strings.HasPrefix(mimeType, textType) {
			return true
		}
	}

	return false
}

// SaveBinaryToTempFile saves binary data to a temporary file and returns the file path
func SaveBinaryToTempFile(data []byte, contentType string) (string, error) {
	if len(data) == 0 {
		return "", nil
	}

	// Generate a unique filename
	randBytes := make([]byte, 8)
	if _, err := rand.Read(randBytes); err != nil {
		return "", fmt.Errorf("failed to generate random filename: %w", err)
	}

	filename := fmt.Sprintf("probe_binary_%x", randBytes)

	// Determine file extension from MIME type
	if ext := getExtensionFromMimeType(contentType); ext != "" {
		filename += ext
	}

	// Create file in temporary directory
	tempDir := os.TempDir()
	filePath := filepath.Join(tempDir, filename)

	// Write data to file
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := file.Write(data); err != nil {
		_ = os.Remove(filePath) // Clean up on error
		return "", fmt.Errorf("failed to write data to temp file: %w", err)
	}

	return filePath, nil
}

// getExtensionFromMimeType returns the appropriate file extension for a MIME type
func getExtensionFromMimeType(contentType string) string {
	if contentType == "" {
		return ""
	}

	mimeType := strings.Split(contentType, ";")[0]
	mimeType = strings.TrimSpace(strings.ToLower(mimeType))

	// Preferred extensions for common types (override mime package defaults)
	extensionMap := map[string]string{
		"image/jpeg":       ".jpg",
		"image/jpg":        ".jpg",
		"image/png":        ".png",
		"image/gif":        ".gif",
		"image/webp":       ".webp",
		"image/svg+xml":    ".svg",
		"video/mp4":        ".mp4",
		"video/webm":       ".webm",
		"video/quicktime":  ".mov",
		"audio/mp3":        ".mp3",
		"audio/mpeg":       ".mp3",
		"audio/wav":        ".wav",
		"application/pdf":  ".pdf",
		"application/zip":  ".zip",
		"application/gzip": ".gz",
		"text/plain":       ".txt",
		"application/json": ".json",
	}

	// Check our preferred extensions first
	if ext, exists := extensionMap[mimeType]; exists {
		return ext
	}

	// Fallback to mime package for other types
	if exts, err := mime.ExtensionsByType(mimeType); err == nil && len(exts) > 0 {
		return exts[0] // Return the first/most common extension
	}

	return ""
}

// ProcessHttpBody processes HTTP body data based on Content-Type
// Returns (bodyString, filePath, error)
func ProcessHttpBody(data []byte, contentType string) (string, string, error) {
	if len(data) == 0 {
		return "", "", nil
	}

	if IsTextualMimeType(contentType) {
		// Return as text body
		return string(data), "", nil
	} else {
		// Save as binary file
		filePath, err := SaveBinaryToTempFile(data, contentType)
		if err != nil {
			return "", "", err
		}
		return "", filePath, nil
	}
}
