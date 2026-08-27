package dto

import (
	"time"

	"tixora/internal/models"
)

// AdminFeeFlat is the flat platform fee charged per order, in Rupiah.
// Exposed on EventResponse so the frontend can display it without
// hardcoding a value that has to stay in sync with the order service.
const AdminFeeFlat int64 = 5000

// EventRequest is the request body for creating or updating an event.
type EventRequest struct {
	Title        string    `json:"title" binding:"required,min=3,max=255"`
	Description  string    `json:"description"`
	EventDate    time.Time `json:"event_date" binding:"required"`
	Location     string    `json:"location" binding:"required"`
	ImageID      *string   `json:"image_id"`
	Price        int64     `json:"price" binding:"required,min=0"`
	TotalTickets int       `json:"total_tickets" binding:"required,min=1"`
	CategoryID   string    `json:"category_id" binding:"required"`
}

// EventResponse is the API representation of an event.
type EventResponse struct {
	ID               string            `json:"id"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	EventDate        time.Time         `json:"event_date"`
	Location         string            `json:"location"`
	ImageURL         string            `json:"image_url,omitempty"`
	Price            int64             `json:"price"`
	AdminFee         int64             `json:"admin_fee"`
	TotalTickets     int               `json:"total_tickets"`
	AvailableTickets int               `json:"available_tickets"`
	CategoryID       string            `json:"category_id"`
	Category         *CategoryResponse `json:"category,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// NewEventResponse builds an EventResponse from event. imageBaseURL is the
// public/CDN base URL images are served from (e.g. https://cdn.tixora.com);
// it's prepended to the linked file's object key to build ImageURL, keeping
// the stored key provider-agnostic.
func NewEventResponse(event *models.Event, imageBaseURL string) EventResponse {
	resp := EventResponse{
		ID:               event.ID,
		Title:            event.Title,
		Description:      event.Description,
		EventDate:        event.EventDate,
		Location:         event.Location,
		Price:            event.Price,
		AdminFee:         AdminFeeFlat,
		TotalTickets:     event.TotalTickets,
		AvailableTickets: event.AvailableTickets,
		CategoryID:       event.CategoryID,
		CreatedAt:        event.CreatedAt,
		UpdatedAt:        event.UpdatedAt,
	}

	if event.Image != nil {
		resp.ImageURL = imageBaseURL + "/" + event.Image.ObjectKey
	}

	if event.Category.ID != "" {
		category := NewCategoryResponse(&event.Category)
		resp.Category = &category
	}

	return resp
}

func NewEventResponseList(events []models.Event, imageBaseURL string) []EventResponse {
	responses := make([]EventResponse, 0, len(events))
	for _, event := range events {
		responses = append(responses, NewEventResponse(&event, imageBaseURL))
	}
	return responses
}
