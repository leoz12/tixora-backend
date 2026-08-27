package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/services"
	"tixora/internal/utils"
)

func newFileService(repo *mockFileRepo, storage *mockStorageService) services.IFileService {
	return services.NewFileService(repo, storage)
}

func TestFileService_PresignUpload_UnsupportedMimeTypeRejected(t *testing.T) {
	svc := newFileService(new(mockFileRepo), new(mockStorageService))

	_, _, err := svc.PresignUpload(context.Background(), "doc.pdf", "application/pdf", 1024)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestFileService_PresignUpload_OversizedRejected(t *testing.T) {
	svc := newFileService(new(mockFileRepo), new(mockStorageService))

	_, _, err := svc.PresignUpload(context.Background(), "cover.jpg", "image/jpeg", 10*1024*1024)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestFileService_PresignUpload_ZeroSizeRejected(t *testing.T) {
	svc := newFileService(new(mockFileRepo), new(mockStorageService))

	_, _, err := svc.PresignUpload(context.Background(), "cover.jpg", "image/jpeg", 0)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestFileService_PresignUpload_Success(t *testing.T) {
	repo := new(mockFileRepo)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*models.File")).
		Run(func(args mock.Arguments) {
			f := args.Get(1).(*models.File)
			assert.Equal(t, models.FileStatusPending, f.Status)
			assert.Contains(t, f.ObjectKey, ".jpg")
		}).
		Return(nil)

	storage := new(mockStorageService)
	storage.On("PresignPutURL", mock.Anything, mock.AnythingOfType("string"), "image/jpeg", services.PresignExpiry).
		Return("https://r2.example/presigned-put-url", nil)

	svc := newFileService(repo, storage)

	file, uploadURL, err := svc.PresignUpload(context.Background(), "cover.jpg", "image/jpeg", 1024)
	require.NoError(t, err)
	assert.Equal(t, "https://r2.example/presigned-put-url", uploadURL)
	assert.Equal(t, "cover.jpg", file.OriginalName)
	repo.AssertExpectations(t)
	storage.AssertExpectations(t)
}

func TestFileService_GetByID_EmptyIDRejected(t *testing.T) {
	svc := newFileService(new(mockFileRepo), new(mockStorageService))

	_, err := svc.GetByID(context.Background(), "")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestFileService_GetByID_NotFound(t *testing.T) {
	repo := new(mockFileRepo)
	repo.On("GetByID", mock.Anything, "missing").Return(nil, nil)

	svc := newFileService(repo, new(mockStorageService))

	_, err := svc.GetByID(context.Background(), "missing")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}
