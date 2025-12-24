package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// ImageStorage defines contract for image storage provider (Cloudinary implementation).
type ImageStorage interface {
	// UploadImage uploads image from reader and returns the secure URL.
	// folder is optional logical folder in storage (e.g. "avatars").
	UploadImage(ctx context.Context, r io.Reader, folder, fileName string) (string, error)
	// DeleteImage deletes image from storage using its URL.
	DeleteImage(ctx context.Context, fileURL string) error
}

type cloudinaryStorage struct {
	cld *cloudinary.Cloudinary
}

// NewCloudinaryStorage creates Cloudinary-backed implementation of ImageStorage.
// It expects CLOUDINARY_URL or individual CLOUDINARY_CLOUD_NAME / CLOUDINARY_API_KEY / CLOUDINARY_API_SECRET
// to be configured in environment variables (see Cloudinary Go SDK docs).
func NewCloudinaryStorage() (ImageStorage, error) {
	// cloudinary.New() automatically reads CLOUDINARY_URL from environment if present.
	cld, err := cloudinary.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cloudinary client: %w", err)
	}

	// Ensure HTTPS URLs by default.
	cld.Config.URL.Secure = true

	// Optional: allow overriding cloud name via env if needed.
	if cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME"); cloudName != "" {
		cld.Config.Cloud.CloudName = cloudName
	}

	return &cloudinaryStorage{cld: cld}, nil
}

// UploadImage uploads an image to Cloudinary and returns the secure URL.
func (s *cloudinaryStorage) UploadImage(ctx context.Context, r io.Reader, folder, fileName string) (string, error) {
	if s == nil || s.cld == nil {
		return "", fmt.Errorf("cloudinary storage is not initialized")
	}

	if ctx == nil {
		ctx = context.Background()
	}

	publicID := fmt.Sprintf("%d-%s", time.Now().UnixNano(), fileName)

	params := uploader.UploadParams{
		Folder:         folder,
		UseFilename:    api.Bool(true),
		UniqueFilename: api.Bool(true),
		PublicID:       publicID,
		Overwrite:      api.Bool(false),
	}

	// Apply WebP conversion and compression only for images
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".bmp", ".tiff", ".gif", ".webp":
		params.Format = "webp"
		params.Transformation = "q_auto"
	}

	resp, err := s.cld.Upload.Upload(ctx, r, params)
	if err != nil {
		return "", fmt.Errorf("failed to upload image to cloudinary: %w", err)
	}

	if resp.SecureURL == "" {
		return "", fmt.Errorf("cloudinary upload succeeded but secure URL is empty")
	}

	return resp.SecureURL, nil
}

// DeleteImage deletes image from Cloudinary.
func (s *cloudinaryStorage) DeleteImage(ctx context.Context, fileURL string) error {
	if s == nil || s.cld == nil {
		return fmt.Errorf("cloudinary storage is not initialized")
	}

	publicID := s.extractPublicID(fileURL)
	if publicID == "" {
		// If we can't extract public ID, we can't delete it.
		// We could return error, but maybe just log it. Returns error for now.
		return fmt.Errorf("could not extract public ID from URL: %s", fileURL)
	}

	// Invalidate: true helps to clear CDN cache
	params := uploader.DestroyParams{
		PublicID:   publicID,
		Invalidate: api.Bool(true),
	}

	resp, err := s.cld.Upload.Destroy(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to delete image from cloudinary: %w", err)
	}

	if resp.Result != "ok" && resp.Result != "not found" {
		return fmt.Errorf("cloudinary destroy api returned result: %s", resp.Result)
	}

	return nil
}

// extractPublicID attempts to extract the public ID from a Cloudinary URL.
// Example: https://res.cloudinary.com/demo/image/upload/v123456789/folder/sample.jpg -> folder/sample
func (s *cloudinaryStorage) extractPublicID(fileURL string) string {
	u, err := url.Parse(fileURL)
	if err != nil {
		return ""
	}

	path := u.Path
	// Path is roughly /<cloud_name>/image/upload/v<version>/<folder>/<file>.<ext>
	// or /<cloud_name>/image/upload/<folder>/<file>.<ext>

	// Find the "upload" segment
	parts := strings.Split(path, "/")
	uploadIndex := -1
	for i, p := range parts {
		if p == "upload" {
			uploadIndex = i
			break
		}
	}

	if uploadIndex == -1 || uploadIndex+1 >= len(parts) {
		return ""
	}

	// Everything after "upload" is potential [version/]public_id.ext
	relevantParts := parts[uploadIndex+1:]

	// Check if the first part is a version (starts with 'v' and is numeric)
	// Cloudinary versions start with 'v' followed by numbers.
	if len(relevantParts) > 0 && strings.HasPrefix(relevantParts[0], "v") {
		// weak check, but okay for cloudinary
		relevantParts = relevantParts[1:] // skip version
	}

	if len(relevantParts) == 0 {
		return ""
	}

	// Join the rest back to get folder/filename.ext
	publicIDWithExt := strings.Join(relevantParts, "/")

	// Strip extension
	ext := filepath.Ext(publicIDWithExt)
	return strings.TrimSuffix(publicIDWithExt, ext)
}
