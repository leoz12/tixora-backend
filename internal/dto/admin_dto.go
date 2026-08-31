package dto

import (
	"time"

	"tixora/internal/models"
)

// AdminResponse is the API representation of an admin.
type AdminResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewAdminResponse(admin *models.Admin) AdminResponse {
	return AdminResponse{
		ID:        admin.ID,
		Email:     admin.Email,
		Name:      admin.Name,
		Role:      admin.Role,
		CreatedAt: admin.CreatedAt,
		UpdatedAt: admin.UpdatedAt,
	}
}

func NewAdminResponseList(admins []models.Admin) []AdminResponse {
	responses := make([]AdminResponse, 0, len(admins))
	for _, admin := range admins {
		responses = append(responses, NewAdminResponse(&admin))
	}
	return responses
}

// AdminLoginRequest is the request body for admin login.
type AdminLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AdminAuthResponse is the API response returned after a successful admin
// login. The access/refresh tokens themselves travel only as httpOnly
// cookies, not in this body.
type AdminAuthResponse struct {
	Admin AdminResponse `json:"admin"`
}

func NewAdminAuthResponse(admin *models.Admin) AdminAuthResponse {
	return AdminAuthResponse{
		Admin: NewAdminResponse(admin),
	}
}

// CreateAdminRequest is the request body for creating a new admin.
type CreateAdminRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=8"`
	Role     string `json:"role"`
}

// UpdateAdminRequest is the request body for updating an existing admin.
type UpdateAdminRequest struct {
	Name     string `json:"name" binding:"required,min=3"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// ChangeAdminPasswordRequest is the request body for the currently
// authenticated admin changing their own password.
type ChangeAdminPasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}
