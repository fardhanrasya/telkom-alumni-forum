package storage

import (
	"context"
	"fmt"
	"io"
	"os"
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

	resp, err := s.cld.Upload.Upload(ctx, r, params)
	if err != nil {
		return "", fmt.Errorf("failed to upload image to cloudinary: %w", err)
	}

	if resp.SecureURL == "" {
		return "", fmt.Errorf("cloudinary upload succeeded but secure URL is empty")
	}

	return resp.SecureURL, nil
}