package image

import (
	"crypto/rand"
	"fmt"
	"image"
	// Register decoders so image.DecodeConfig can validate uploads
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

type Processor struct {
	uploadDir     string
	maxFileSize   int64
	allowedTypes  []string
	thumbnailSize int
}

func NewProcessor(uploadDir string, maxFileSize int64, allowedTypes []string) *Processor {
	return &Processor{
		uploadDir:     uploadDir,
		maxFileSize:   maxFileSize,
		allowedTypes:  allowedTypes,
		thumbnailSize: 300, // 300x300 thumbnail
	}
}

// ProcessUpload handles file upload, validation, and thumbnail generation
func (p *Processor) ProcessUpload(file *multipart.FileHeader) (string, string, error) {
	// Validate file size
	if file.Size > p.maxFileSize {
		return "", "", fmt.Errorf("file size exceeds maximum allowed size of %d bytes", p.maxFileSize)
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !p.isAllowedType(ext) {
		return "", "", fmt.Errorf("file type %s is not allowed", ext)
	}

	// Validate actual file content, not just the extension
	if err := p.ValidateImage(file); err != nil {
		return "", "", err
	}

	// Generate unique filename
	filename := p.generateUniqueFilename(ext)
	filePath := filepath.Join(p.uploadDir, filename)

	// Create upload directory if it doesn't exist
	if err := os.MkdirAll(p.uploadDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	// Save original file
	if err := p.saveFile(file, filePath); err != nil {
		return "", "", fmt.Errorf("failed to save file: %w", err)
	}

	// Generate thumbnail
	thumbnailFilename := p.generateThumbnailFilename(filename)
	thumbnailPath := filepath.Join(p.uploadDir, thumbnailFilename)

	if err := p.generateThumbnail(filePath, thumbnailPath); err != nil {
		// Remove original file if thumbnail generation fails
		os.Remove(filePath)
		return "", "", fmt.Errorf("failed to generate thumbnail: %w", err)
	}

	return filename, thumbnailFilename, nil
}

// saveFile saves the uploaded file to disk
func (p *Processor) saveFile(file *multipart.FileHeader, filePath string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// generateThumbnail creates a thumbnail version of the uploaded image
func (p *Processor) generateThumbnail(originalPath, thumbnailPath string) error {
	// Open original image
	img, err := imaging.Open(originalPath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Resize image to thumbnail size while maintaining aspect ratio
	thumbnail := imaging.Fit(img, p.thumbnailSize, p.thumbnailSize, imaging.Lanczos)

	// Save thumbnail
	if err := imaging.Save(thumbnail, thumbnailPath); err != nil {
		return fmt.Errorf("failed to save thumbnail: %w", err)
	}

	return nil
}

// generateUniqueFilename creates a unique filename with random bytes
func (p *Processor) generateUniqueFilename(ext string) string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// crypto/rand failing is unrecoverable; fall back to a timestamp-based name
		return fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	}
	return fmt.Sprintf("%x%s", bytes, ext)
}

// generateThumbnailFilename creates a thumbnail filename from the original filename
func (p *Processor) generateThumbnailFilename(filename string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	return fmt.Sprintf("%s_thumb%s", name, ext)
}

// isAllowedType checks if the file extension is in the allowed types list
func (p *Processor) isAllowedType(ext string) bool {
	for _, allowedType := range p.allowedTypes {
		if ext == allowedType {
			return true
		}
	}
	return false
}

// ValidateImage validates that the file is a valid image
func (p *Processor) ValidateImage(file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Try to decode the image to validate it
	_, _, err = image.DecodeConfig(src)
	if err != nil {
		return fmt.Errorf("invalid image file: %w", err)
	}

	return nil
}

// GetImageInfo returns basic information about an image file
func GetImageInfo(filePath string) (width, height int, format string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, "", err
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, "", err
	}

	return config.Width, config.Height, format, nil
}

// DeleteImage deletes an image file and its thumbnail
func (p *Processor) DeleteImage(filename string) error {
	filePath := filepath.Join(p.uploadDir, filename)
	thumbnailFilename := p.generateThumbnailFilename(filename)
	thumbnailPath := filepath.Join(p.uploadDir, thumbnailFilename)

	// Delete original file
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete original file: %w", err)
	}

	// Delete thumbnail
	if err := os.Remove(thumbnailPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete thumbnail: %w", err)
	}

	return nil
}

// GetImageURL returns the URL path for an image
func (p *Processor) GetImageURL(filename string) string {
	return fmt.Sprintf("/uploads/%s", filename)
}

// GetThumbnailURL returns the URL path for a thumbnail
func (p *Processor) GetThumbnailURL(filename string) string {
	thumbnailFilename := p.generateThumbnailFilename(filename)
	return fmt.Sprintf("/uploads/%s", thumbnailFilename)
}
