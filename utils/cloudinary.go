package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/sirupsen/logrus"
)

type CloudinaryService struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinaryService() (*CloudinaryService, error) {
	cloudinaryURL := os.Getenv("CLOUDINARY_URL")

	if cloudinaryURL == "" {
		return nil, fmt.Errorf("CLOUDINARY_URL environment variable is not set")
	}

	cld, err := cloudinary.NewFromURL(cloudinaryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary: %w", err)
	}

	return &CloudinaryService{cld: cld}, nil
}

func (s *CloudinaryService) UploadImage(ctx context.Context, file io.Reader, publicID string) (string, error) {
	overwrite := true
	uploadResult, err := s.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		PublicID:       publicID,
		Folder:         "bikinota/company-logos",
		AllowedFormats: []string{"jpg", "jpeg", "png", "gif", "webp"},
		ResourceType:   "image",
		Overwrite:      &overwrite,
	})

	if err != nil {
		logrus.Errorf("Cloudinary upload error: %v", err)
		return "", fmt.Errorf("failed to upload image: %w", err)
	}

	return uploadResult.SecureURL, nil
}

// GetPublicIDFromURL extracts the public ID from a Cloudinary URL
func GetPublicIDFromURL(url string) string {
	// Cloudinary URL format: https://res.cloudinary.com/{cloud_name}/image/upload/v{version}/{folder}/{public_id}.{format}
	// We need to extract: {folder}/{public_id}
	// For our case: bikinota/company-logos/company-logo-{user_id}
	// This is a simplified extraction - in production, you might want to store the public_id separately
	if url == "" {
		return ""
	}
	// Extract the path after /upload/
	parts := strings.Split(url, "/upload/")
	if len(parts) < 2 {
		return ""
	}
	// Get the part after version (v{number}/)
	pathParts := strings.Split(parts[1], "/")
	if len(pathParts) < 2 {
		return ""
	}
	// Skip version and get folder + filename
	// Remove file extension
	filename := pathParts[len(pathParts)-1]
	extIndex := strings.LastIndex(filename, ".")
	if extIndex > 0 {
		filename = filename[:extIndex]
	}
	// Reconstruct: folder + filename
	if len(pathParts) > 2 {
		return strings.Join(pathParts[1:len(pathParts)-1], "/") + "/" + filename
	}
	return filename
}

func (s *CloudinaryService) DeleteImage(ctx context.Context, publicID string) error {
	_, err := s.cld.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID:     publicID,
		ResourceType: "image",
	})

	if err != nil {
		logrus.Errorf("Cloudinary delete error: %v", err)
		return fmt.Errorf("failed to delete image: %w", err)
	}

	return nil
}
