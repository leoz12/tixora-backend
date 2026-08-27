package dto

import (
	"time"

	"tixora/internal/models"
)

// CategoryRequest is the request body for creating or updating a category.
type CategoryRequest struct {
	Name     string `json:"name" binding:"required,min=3,max=100"`
	IsActive *bool  `json:"is_active"`
}

// CategoryResponse is the API representation of a category.
type CategoryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewCategoryResponse(category *models.Category) CategoryResponse {
	return CategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		Slug:      category.Slug,
		IsActive:  category.IsActive,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}
}

func NewCategoryResponseList(categories []models.Category) []CategoryResponse {
	responses := make([]CategoryResponse, 0, len(categories))
	for _, category := range categories {
		responses = append(responses, NewCategoryResponse(&category))
	}
	return responses
}
