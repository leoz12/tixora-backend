package services

import (
	"context"
	"fmt"

	"tixora/internal/models"
	"tixora/internal/repository"
	"tixora/internal/utils"
)

const (
	defaultEventPageLimit = 12
	maxEventPageLimit     = 50
)

// IEventService contains event catalog business logic.
type IEventService interface {
	GetEvents(ctx context.Context, page, limit int, categoryID, search string, includePast bool) ([]models.Event, int64, error)
	GetEventByID(ctx context.Context, id string) (*models.Event, error)
	SearchEvents(ctx context.Context, query string, page, limit int, includePast bool) ([]models.Event, int64, error)
	CreateEvent(ctx context.Context, event *models.Event) error
	UpdateEvent(ctx context.Context, event *models.Event) error
	DeleteEvent(ctx context.Context, id string) error
}

type EventService struct {
	repo         repository.IEventRepository
	categoryRepo repository.ICategoryRepository
	fileRepo     repository.IFileRepository
}

func NewEventService(repo repository.IEventRepository, categoryRepo repository.ICategoryRepository, fileRepo repository.IFileRepository) IEventService {
	return &EventService{repo: repo, categoryRepo: categoryRepo, fileRepo: fileRepo}
}

func (s *EventService) GetEvents(ctx context.Context, page, limit int, categoryID, search string, includePast bool) ([]models.Event, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxEventPageLimit {
		limit = defaultEventPageLimit
	}

	offset := (page - 1) * limit
	events, total, err := s.repo.GetWithPagination(ctx, offset, limit, categoryID, search, includePast)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch events: %w", err)
	}

	return events, total, nil
}

func (s *EventService) SearchEvents(ctx context.Context, query string, page, limit int, includePast bool) ([]models.Event, int64, error) {
	if query == "" {
		return nil, 0, fmt.Errorf("%w: search query is required", utils.ErrInvalidInput)
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > maxEventPageLimit {
		limit = defaultEventPageLimit
	}

	offset := (page - 1) * limit
	events, total, err := s.repo.Search(ctx, query, offset, limit, includePast)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search events: %w", err)
	}

	return events, total, nil
}

func (s *EventService) GetEventByID(ctx context.Context, id string) (*models.Event, error) {
	if id == "" {
		return nil, fmt.Errorf("%w: event id is required", utils.ErrInvalidInput)
	}

	event, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event: %w", err)
	}
	if event == nil {
		return nil, fmt.Errorf("%w: event", utils.ErrNotFound)
	}

	return event, nil
}

func (s *EventService) CreateEvent(ctx context.Context, event *models.Event) error {
	if err := validateEvent(event); err != nil {
		return err
	}
	if err := s.validateCategory(ctx, event.CategoryID); err != nil {
		return err
	}
	if err := s.validateImage(ctx, event.ImageID); err != nil {
		return err
	}

	if event.ID == "" {
		event.ID = utils.GenerateUUID()
	}
	if event.AvailableTickets == 0 {
		event.AvailableTickets = event.TotalTickets
	}

	if err := s.repo.Create(ctx, event); err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	return nil
}

func (s *EventService) UpdateEvent(ctx context.Context, event *models.Event) error {
	if event.ID == "" {
		return fmt.Errorf("%w: event id is required", utils.ErrInvalidInput)
	}
	if err := validateEvent(event); err != nil {
		return err
	}
	if err := s.validateCategory(ctx, event.CategoryID); err != nil {
		return err
	}
	if err := s.validateImage(ctx, event.ImageID); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("failed to fetch event: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("%w: event", utils.ErrNotFound)
	}

	existing.Title = event.Title
	existing.Description = event.Description
	existing.EventDate = event.EventDate
	existing.Location = event.Location
	existing.ImageID = event.ImageID
	existing.Price = event.Price
	existing.TotalTickets = event.TotalTickets
	existing.AvailableTickets = event.AvailableTickets
	existing.CategoryID = event.CategoryID

	if err := s.repo.Update(ctx, existing); err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// validateCategory checks that categoryID refers to an existing, active category.
func (s *EventService) validateCategory(ctx context.Context, categoryID string) error {
	category, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("failed to fetch category: %w", err)
	}
	if category == nil {
		return fmt.Errorf("%w: category not found", utils.ErrInvalidInput)
	}
	if !category.IsActive {
		return fmt.Errorf("%w: category is inactive", utils.ErrInvalidInput)
	}
	return nil
}

// validateImage checks that imageID (when set) refers to an existing file,
// and marks that file as attached so it's known to be in use.
func (s *EventService) validateImage(ctx context.Context, imageID *string) error {
	if imageID == nil || *imageID == "" {
		return nil
	}

	file, err := s.fileRepo.GetByID(ctx, *imageID)
	if err != nil {
		return fmt.Errorf("failed to fetch image file: %w", err)
	}
	if file == nil {
		return fmt.Errorf("%w: image file not found", utils.ErrInvalidInput)
	}

	if file.Status != models.FileStatusAttached {
		file.Status = models.FileStatusAttached
		if err := s.fileRepo.Update(ctx, file); err != nil {
			return fmt.Errorf("failed to update image file status: %w", err)
		}
	}

	return nil
}

func (s *EventService) DeleteEvent(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("%w: event id is required", utils.ErrInvalidInput)
	}

	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to fetch event: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("%w: event", utils.ErrNotFound)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

func validateEvent(event *models.Event) error {
	if event.Title == "" {
		return fmt.Errorf("%w: title is required", utils.ErrInvalidInput)
	}
	if event.EventDate.IsZero() {
		return fmt.Errorf("%w: event date is required", utils.ErrInvalidInput)
	}
	if event.Price < 0 {
		return fmt.Errorf("%w: price cannot be negative", utils.ErrInvalidInput)
	}
	if event.TotalTickets < 1 {
		return fmt.Errorf("%w: total tickets must be at least 1", utils.ErrInvalidInput)
	}
	if event.AvailableTickets < 0 || event.AvailableTickets > event.TotalTickets {
		return fmt.Errorf("%w: available tickets must be between 0 and total tickets", utils.ErrInvalidInput)
	}
	if event.CategoryID == "" {
		return fmt.Errorf("%w: category id is required", utils.ErrInvalidInput)
	}

	return nil
}
