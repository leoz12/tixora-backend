package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"tixora/internal/models"
)

type IEventRepository interface {
	GetWithPagination(ctx context.Context, offset, limit int, categoryID, search string) ([]models.Event, int64, error)
	GetByID(ctx context.Context, id string) (*models.Event, error)
	Search(ctx context.Context, query string, offset, limit int) ([]models.Event, int64, error)
	CountByCategoryID(ctx context.Context, categoryID string) (int64, error)
	Create(ctx context.Context, event *models.Event) error
	Update(ctx context.Context, event *models.Event) error
	Delete(ctx context.Context, id string) error
}

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) IEventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) GetWithPagination(ctx context.Context, offset, limit int, categoryID, search string) ([]models.Event, int64, error) {
	var events []models.Event
	var total int64

	countQuery := applyEventFilters(r.db.WithContext(ctx).Model(&models.Event{}), categoryID, search)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	findQuery := applyEventFilters(r.db.WithContext(ctx).Preload("Category").Preload("Image"), categoryID, search)
	if err := findQuery.
		Offset(offset).
		Limit(limit).
		Order("event_date asc").
		Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch events: %w", err)
	}

	return events, total, nil
}

// applyEventFilters applies an exact category match and a free-text search
// (matched against title/location) to the given query builder.
func applyEventFilters(db *gorm.DB, categoryID, search string) *gorm.DB {
	if categoryID != "" {
		db = db.Where("category_id = ?", categoryID)
	}
	if search != "" {
		like := "%" + search + "%"
		db = db.Where("title LIKE ? OR location LIKE ?", like, like)
	}
	return db
}

func (r *EventRepository) Search(ctx context.Context, query string, offset, limit int) ([]models.Event, int64, error) {
	var events []models.Event
	var total int64

	like := "%" + query + "%"
	condition := "title LIKE ? OR location LIKE ?"

	if err := r.db.WithContext(ctx).Model(&models.Event{}).
		Where(condition, like, like).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	if err := r.db.WithContext(ctx).Preload("Category").Preload("Image").
		Where(condition, like, like).
		Offset(offset).
		Limit(limit).
		Order("event_date asc").
		Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search events: %w", err)
	}

	return events, total, nil
}

func (r *EventRepository) CountByCategoryID(ctx context.Context, categoryID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&models.Event{}).
		Where("category_id = ?", categoryID).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count events by category: %w", err)
	}
	return count, nil
}

func (r *EventRepository) GetByID(ctx context.Context, id string) (*models.Event, error) {
	var event models.Event
	if err := r.db.WithContext(ctx).Preload("Category").Preload("Image").Where("id = ?", id).First(&event).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get event by id: %w", err)
	}
	return &event, nil
}

func (r *EventRepository) Create(ctx context.Context, event *models.Event) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}
	return nil
}

func (r *EventRepository) Update(ctx context.Context, event *models.Event) error {
	if err := r.db.WithContext(ctx).Save(event).Error; err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}
	return nil
}

func (r *EventRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Event{}).Error; err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	return nil
}
