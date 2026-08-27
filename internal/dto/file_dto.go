package dto

import "tixora/internal/models"

// PresignUploadRequest describes the file the client is about to upload
// directly to R2.
type PresignUploadRequest struct {
	Filename string `json:"filename" binding:"required"`
	MimeType string `json:"mime_type" binding:"required"`
	Size     int64  `json:"size" binding:"required,min=1"`
}

// PresignUploadResponse is returned so the client can PUT the file bytes
// straight to R2, then reference FileID when creating/updating the owning
// resource (e.g. an event's image_id).
type PresignUploadResponse struct {
	FileID           string `json:"file_id"`
	UploadURL        string `json:"upload_url"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

func NewPresignUploadResponse(file *models.File, uploadURL string, expiresInSeconds int) PresignUploadResponse {
	return PresignUploadResponse{
		FileID:           file.ID,
		UploadURL:        uploadURL,
		ExpiresInSeconds: expiresInSeconds,
	}
}
