package dto

import (
	"time"

	"tixora/internal/models"
)

// UserResponse is the API representation of a user.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		AvatarURL: user.AvatarURL,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func NewUserResponseList(users []models.User) []UserResponse {
	responses := make([]UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, NewUserResponse(&user))
	}
	return responses
}

// UserStatsResponse is the API representation of a user's order stats.
type UserStatsResponse struct {
	TotalOrders int64 `json:"total_orders"`
	TotalSpent  int64 `json:"total_spent"`
}

// UpdateProfileRequest is the request body for updating the current user's profile.
type UpdateProfileRequest struct {
	Name      string `json:"name" binding:"required,min=3"`
	AvatarURL string `json:"avatar_url"`
}

// GoogleCallbackRequest is the request body for the Google OAuth callback.
type GoogleCallbackRequest struct {
	Code string `json:"code" binding:"required"`
}

// AuthResponse is the API response returned after a successful login.
// The access/refresh tokens themselves travel only as httpOnly cookies, not
// in this body.
type AuthResponse struct {
	User UserResponse `json:"user"`
}

func NewAuthResponse(user *models.User) AuthResponse {
	return AuthResponse{
		User: NewUserResponse(user),
	}
}
