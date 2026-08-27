package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tixora/internal/dto"
	"tixora/internal/services"
)

// FileHandler handles direct-upload endpoints backing the generic file registry.
type FileHandler struct {
	fileService services.IFileService
}

func NewFileHandler(fileService services.IFileService) *FileHandler {
	return &FileHandler{fileService: fileService}
}

// PresignUpload handles POST /api/files/presign-upload (admin only)
//
//	@Summary		Request a direct upload URL
//	@Description	Registers a pending file record and returns a presigned R2 URL. The client uploads the file bytes directly to that URL (PUT, with the same Content-Type), then references the returned file_id when creating/updating the owning resource (e.g. an event's image_id). Requires admin authentication.
//	@Tags			files
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			request	body		dto.PresignUploadRequest	true	"Upload metadata"
//	@Success		201		{object}	dto.SuccessResponse{data=dto.PresignUploadResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Router			/files/presign-upload [post]
func (h *FileHandler) PresignUpload(c *gin.Context) {
	var req dto.PresignUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	file, uploadURL, err := h.fileService.PresignUpload(c.Request.Context(), req.Filename, req.MimeType, req.Size)
	if err != nil {
		respondError(c, err, "Failed to create upload url")
		return
	}

	resp := dto.NewPresignUploadResponse(file, uploadURL, int(services.PresignExpiry.Seconds()))
	c.JSON(http.StatusCreated, dto.NewSuccessResponse("Upload URL created", resp))
}
