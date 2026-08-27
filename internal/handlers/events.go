package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"tixora/internal/dto"
	"tixora/internal/models"
	"tixora/internal/services"
)

// EventHandler handles event catalog endpoints.
type EventHandler struct {
	eventService services.IEventService
	imageBaseURL string
}

// NewEventHandler builds an EventHandler. imageBaseURL is the public/CDN
// base URL event images are served from, used to build absolute image_url
// values in responses.
func NewEventHandler(eventService services.IEventService, imageBaseURL string) *EventHandler {
	return &EventHandler{eventService: eventService, imageBaseURL: imageBaseURL}
}

// GetEvents handles GET /api/events
//
//	@Summary		List events
//	@Description	Returns a paginated list of events, optionally filtered by category and/or a free-text search on title/location.
//	@Tags			events
//	@Produce		json
//	@Param			page		query		int		false	"Page number"		default(1)
//	@Param			limit		query		int		false	"Items per page"	default(12)
//	@Param			category_id	query		string	false	"Exact category ID filter"
//	@Param			search		query		string	false	"Free-text search on title/location"
//	@Success		200			{object}	dto.ListResponse{data=[]dto.EventResponse}
//	@Failure		500			{object}	dto.ErrorResponse
//	@Router			/events [get]
func (h *EventHandler) GetEvents(c *gin.Context) {
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 12)
	categoryID := c.Query("category_id")
	search := c.Query("search")

	events, total, err := h.eventService.GetEvents(c.Request.Context(), page, limit, categoryID, search)
	if err != nil {
		respondError(c, err, "Failed to fetch events")
		return
	}

	c.JSON(http.StatusOK, dto.NewListResponse(dto.NewEventResponseList(events, h.imageBaseURL), page, limit, int(total)))
}

// GetEventByID handles GET /api/events/:id
//
//	@Summary		Get event by ID
//	@Description	Returns a single event by its ID.
//	@Tags			events
//	@Produce		json
//	@Param			id	path		string	true	"Event ID"
//	@Success		200	{object}	dto.SuccessResponse{data=dto.EventResponse}
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/events/{id} [get]
func (h *EventHandler) GetEventByID(c *gin.Context) {
	id := c.Param("id")

	event, err := h.eventService.GetEventByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err, "Failed to fetch event")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("", dto.NewEventResponse(event, h.imageBaseURL)))
}

// SearchEvents handles GET /api/events/search
//
//	@Summary		Search events
//	@Description	Searches events by a free-text query.
//	@Tags			events
//	@Produce		json
//	@Param			q		query		string	true	"Search query"
//	@Param			page	query		int		false	"Page number"		default(1)
//	@Param			limit	query		int		false	"Items per page"	default(12)
//	@Success		200		{object}	dto.ListResponse{data=[]dto.EventResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Router			/events/search [get]
func (h *EventHandler) SearchEvents(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Query parameter 'q' is required", nil))
		return
	}

	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 12)

	events, total, err := h.eventService.SearchEvents(c.Request.Context(), query, page, limit)
	if err != nil {
		respondError(c, err, "Failed to search events")
		return
	}

	c.JSON(http.StatusOK, dto.NewListResponse(dto.NewEventResponseList(events, h.imageBaseURL), page, limit, int(total)))
}

// CreateEvent handles POST /api/events (admin only)
//
//	@Summary		Create event
//	@Description	Creates a new event. Requires admin authentication.
//	@Tags			events
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			request	body		dto.EventRequest	true	"Event details"
//	@Success		201		{object}	dto.SuccessResponse{data=dto.EventResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Router			/events [post]
func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req dto.EventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	event := eventFromRequest(&req)
	if err := h.eventService.CreateEvent(c.Request.Context(), event); err != nil {
		respondError(c, err, "Failed to create event")
		return
	}

	created, err := h.eventService.GetEventByID(c.Request.Context(), event.ID)
	if err != nil {
		respondError(c, err, "Failed to fetch created event")
		return
	}

	c.JSON(http.StatusCreated, dto.NewSuccessResponse("Event created", dto.NewEventResponse(created, h.imageBaseURL)))
}

// UpdateEvent handles PUT /api/events/:id (admin only)
//
//	@Summary		Update event
//	@Description	Updates an existing event. Requires admin authentication.
//	@Tags			events
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id		path		string				true	"Event ID"
//	@Param			request	body		dto.EventRequest	true	"Event details"
//	@Success		200		{object}	dto.SuccessResponse{data=dto.EventResponse}
//	@Failure		400		{object}	dto.ErrorResponse
//	@Failure		401		{object}	dto.ErrorResponse
//	@Failure		403		{object}	dto.ErrorResponse
//	@Failure		404		{object}	dto.ErrorResponse
//	@Router			/events/{id} [put]
func (h *EventHandler) UpdateEvent(c *gin.Context) {
	id := c.Param("id")

	var req dto.EventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.NewErrorResponse("Invalid request body", err))
		return
	}

	event := eventFromRequest(&req)
	event.ID = id
	if err := h.eventService.UpdateEvent(c.Request.Context(), event); err != nil {
		respondError(c, err, "Failed to update event")
		return
	}

	updated, err := h.eventService.GetEventByID(c.Request.Context(), id)
	if err != nil {
		respondError(c, err, "Failed to fetch updated event")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Event updated", dto.NewEventResponse(updated, h.imageBaseURL)))
}

// DeleteEvent handles DELETE /api/events/:id (admin only)
//
//	@Summary		Delete event
//	@Description	Deletes an event. Requires admin authentication.
//	@Tags			events
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		string	true	"Event ID"
//	@Success		200	{object}	dto.SuccessResponse
//	@Failure		401	{object}	dto.ErrorResponse
//	@Failure		403	{object}	dto.ErrorResponse
//	@Failure		404	{object}	dto.ErrorResponse
//	@Router			/events/{id} [delete]
func (h *EventHandler) DeleteEvent(c *gin.Context) {
	id := c.Param("id")

	if err := h.eventService.DeleteEvent(c.Request.Context(), id); err != nil {
		respondError(c, err, "Failed to delete event")
		return
	}

	c.JSON(http.StatusOK, dto.NewSuccessResponse("Event deleted", nil))
}

func eventFromRequest(req *dto.EventRequest) *models.Event {
	return &models.Event{
		Title:        req.Title,
		Description:  req.Description,
		EventDate:    req.EventDate,
		Location:     req.Location,
		ImageID:      req.ImageID,
		Price:        req.Price,
		TotalTickets: req.TotalTickets,
		CategoryID:   req.CategoryID,
	}
}

// parseIntQuery reads an integer query parameter, falling back to defaultValue
// when it is missing or malformed. Range validation is left to the service layer.
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return defaultValue
	}
	return value
}
