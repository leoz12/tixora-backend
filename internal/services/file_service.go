package services

import (
	"context"
	"fmt"
	"time"

	"tixora/internal/models"
	"tixora/internal/repository"
	"tixora/internal/utils"
)

// PresignExpiry is how long a presigned upload URL stays valid for.
const PresignExpiry = 15 * time.Minute

const maxUploadSize = 5 * 1024 * 1024 // 5MB

// allowedUploadMimeTypes maps accepted MIME types to their stored extension.
// Only images are accepted for now (event covers); widen this map when
// other upload use cases (e.g. review attachments) are added.
var allowedUploadMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// IFileService manages the generic file registry and issues direct upload URLs.
type IFileService interface {
	PresignUpload(ctx context.Context, originalName, mimeType string, size int64) (*models.File, string, error)
	GetByID(ctx context.Context, id string) (*models.File, error)
}

type FileService struct {
	repo    repository.IFileRepository
	storage IStorageService
}

func NewFileService(repo repository.IFileRepository, storage IStorageService) IFileService {
	return &FileService{repo: repo, storage: storage}
}

// PresignUpload registers a pending file record and returns a presigned R2
// URL the caller uploads the bytes to directly. The file starts as
// "pending" and is only marked "attached" once a consumer (e.g. an event)
// actually references its ID.
func (s *FileService) PresignUpload(ctx context.Context, originalName, mimeType string, size int64) (*models.File, string, error) {
	ext, ok := allowedUploadMimeTypes[mimeType]
	if !ok {
		return nil, "", fmt.Errorf("%w: unsupported file type %q", utils.ErrInvalidInput, mimeType)
	}
	if size <= 0 || size > maxUploadSize {
		return nil, "", fmt.Errorf("%w: file size must be between 1 and %d bytes", utils.ErrInvalidInput, maxUploadSize)
	}

	file := &models.File{
		ID:           utils.GenerateUUID(),
		ObjectKey:    fmt.Sprintf("uploads/%s%s", utils.GenerateUUID(), ext),
		OriginalName: originalName,
		Provider:     "r2",
		MimeType:     mimeType,
		Size:         size,
		Status:       models.FileStatusPending,
	}

	if err := s.repo.Create(ctx, file); err != nil {
		return nil, "", fmt.Errorf("failed to create file record: %w", err)
	}

	uploadURL, err := s.storage.PresignPutURL(ctx, file.ObjectKey, mimeType, PresignExpiry)
	if err != nil {
		return nil, "", err
	}

	return file, uploadURL, nil
}

func (s *FileService) GetByID(ctx context.Context, id string) (*models.File, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: file id is required", utils.ErrInvalidInput)
	}

	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch file: %w", err)
	}
	if file == nil {
		return nil, fmt.Errorf("%w: file", utils.ErrNotFound)
	}

	return file, nil
}
