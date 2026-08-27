package dto

// SuccessResponse is the standard envelope for successful API responses.
type SuccessResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse is the standard envelope for failed API responses.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// ListResponse is the standard envelope for paginated list responses.
type ListResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination describes pagination metadata for list responses.
type Pagination struct {
	CurrentPage int `json:"current_page"`
	TotalPages  int `json:"total_pages"`
	TotalItems  int `json:"total_items"`
	PerPage     int `json:"per_page"`
}

func NewSuccessResponse(message string, data interface{}) SuccessResponse {
	return SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

func NewErrorResponse(message string, err error) ErrorResponse {
	resp := ErrorResponse{
		Success: false,
		Message: message,
	}
	if err != nil {
		resp.Error = err.Error()
	}
	return resp
}

func NewListResponse(data interface{}, page, limit, totalItems int) ListResponse {
	totalPages := 0
	if limit > 0 {
		totalPages = (totalItems + limit - 1) / limit
	}

	return ListResponse{
		Success: true,
		Data:    data,
		Pagination: Pagination{
			CurrentPage: page,
			TotalPages:  totalPages,
			TotalItems:  totalItems,
			PerPage:     limit,
		},
	}
}
