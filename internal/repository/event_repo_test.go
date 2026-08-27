package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/repository"
)

func TestEventRepository_CreateAndGetByID(t *testing.T) {
	db := requireDB(t)
	ctx := context.Background()
	catRepo := repository.NewCategoryRepository(db)
	require.NoError(t, catRepo.Create(ctx, newTestCategory("cat-1", "Music", "music", true)))

	eventRepo := repository.NewEventRepository(db)
	event := &models.Event{
		ID: "evt-1", Title: "Concert", EventDate: time.Now().Add(24 * time.Hour),
		Location: "Jakarta", Price: 100000, TotalTickets: 100, AvailableTickets: 100, CategoryID: "cat-1",
	}
	require.NoError(t, eventRepo.Create(ctx, event))

	fetched, err := eventRepo.GetByID(ctx, "evt-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "Concert", fetched.Title)
	assert.Equal(t, "Music", fetched.Category.Name, "GetByID must preload Category")
}

func TestEventRepository_GetByID_NotFound(t *testing.T) {
	db := requireDB(t)
	eventRepo := repository.NewEventRepository(db)

	fetched, err := eventRepo.GetByID(context.Background(), "missing")
	require.NoError(t, err)
	assert.Nil(t, fetched)
}

func TestEventRepository_GetWithPagination_FiltersByCategory(t *testing.T) {
	db := requireDB(t)
	ctx := context.Background()
	catRepo := repository.NewCategoryRepository(db)
	require.NoError(t, catRepo.Create(ctx, newTestCategory("cat-1", "Music", "music", true)))
	require.NoError(t, catRepo.Create(ctx, newTestCategory("cat-2", "Sports", "sports", true)))

	eventRepo := repository.NewEventRepository(db)
	require.NoError(t, eventRepo.Create(ctx, &models.Event{
		ID: "evt-1", Title: "Concert", EventDate: time.Now().Add(time.Hour), Price: 1000,
		TotalTickets: 10, AvailableTickets: 10, CategoryID: "cat-1",
	}))
	require.NoError(t, eventRepo.Create(ctx, &models.Event{
		ID: "evt-2", Title: "Match", EventDate: time.Now().Add(2 * time.Hour), Price: 1000,
		TotalTickets: 10, AvailableTickets: 10, CategoryID: "cat-2",
	}))

	events, total, err := eventRepo.GetWithPagination(ctx, 0, 10, "cat-1", "")
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, events, 1)
	assert.Equal(t, "evt-1", events[0].ID)
}

func TestEventRepository_GetWithPagination_OrdersByEventDateAscending(t *testing.T) {
	db := requireDB(t)
	ctx := context.Background()
	catRepo := repository.NewCategoryRepository(db)
	require.NoError(t, catRepo.Create(ctx, newTestCategory("cat-1", "Music", "music", true)))

	eventRepo := repository.NewEventRepository(db)
	later := time.Now().Add(48 * time.Hour)
	sooner := time.Now().Add(1 * time.Hour)
	require.NoError(t, eventRepo.Create(ctx, &models.Event{
		ID: "evt-later", Title: "Later Show", EventDate: later, Price: 1000,
		TotalTickets: 10, AvailableTickets: 10, CategoryID: "cat-1",
	}))
	require.NoError(t, eventRepo.Create(ctx, &models.Event{
		ID: "evt-sooner", Title: "Sooner Show", EventDate: sooner, Price: 1000,
		TotalTickets: 10, AvailableTickets: 10, CategoryID: "cat-1",
	}))

	events, _, err := eventRepo.GetWithPagination(ctx, 0, 10, "", "")
	require.NoError(t, err)
	require.Len(t, events, 2)
	assert.Equal(t, "evt-sooner", events[0].ID)
	assert.Equal(t, "evt-later", events[1].ID)
}

func TestEventRepository_Search_MatchesTitleOrLocation(t *testing.T) {
	db := requireDB(t)
	ctx := context.Background()
	catRepo := repository.NewCategoryRepository(db)
	require.NoError(t, catRepo.Create(ctx, newTestCategory("cat-1", "Music", "music", true)))

	eventRepo := repository.NewEventRepository(db)
	require.NoError(t, eventRepo.Create(ctx, &models.Event{
		ID: "evt-1", Title: "Jazz Night", Location: "Bandung", EventDate: time.Now().Add(time.Hour),
		Price: 1000, TotalTickets: 10, AvailableTickets: 10, CategoryID: "cat-1",
	}))
	require.NoError(t, eventRepo.Create(ctx, &models.Event{
		ID: "evt-2", Title: "Rock Fest", Location: "Jakarta", EventDate: time.Now().Add(time.Hour),
		Price: 1000, TotalTickets: 10, AvailableTickets: 10, CategoryID: "cat-1",
	}))

	byTitle, total, err := eventRepo.Search(ctx, "Jazz", 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byTitle, 1)
	assert.Equal(t, "evt-1", byTitle[0].ID)

	byLocation, total, err := eventRepo.Search(ctx, "Jakarta", 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, byLocation, 1)
	assert.Equal(t, "evt-2", byLocation[0].ID)
}

func TestEventRepository_CountByCategoryID(t *testing.T) {
	db := requireDB(t)
	ctx := context.Background()
	catRepo := repository.NewCategoryRepository(db)
	require.NoError(t, catRepo.Create(ctx, newTestCategory("cat-1", "Music", "music", true)))

	eventRepo := repository.NewEventRepository(db)
	require.NoError(t, eventRepo.Create(ctx, &models.Event{
		ID: "evt-1", Title: "Show 1", EventDate: time.Now().Add(time.Hour), Price: 1000,
		TotalTickets: 10, AvailableTickets: 10, CategoryID: "cat-1",
	}))
	require.NoError(t, eventRepo.Create(ctx, &models.Event{
		ID: "evt-2", Title: "Show 2", EventDate: time.Now().Add(time.Hour), Price: 1000,
		TotalTickets: 10, AvailableTickets: 10, CategoryID: "cat-1",
	}))

	count, err := eventRepo.CountByCategoryID(ctx, "cat-1")
	require.NoError(t, err)
	assert.EqualValues(t, 2, count)
}

func TestEventRepository_UpdateAndDelete(t *testing.T) {
	db := requireDB(t)
	ctx := context.Background()
	catRepo := repository.NewCategoryRepository(db)
	require.NoError(t, catRepo.Create(ctx, newTestCategory("cat-1", "Music", "music", true)))

	eventRepo := repository.NewEventRepository(db)
	event := &models.Event{
		ID: "evt-1", Title: "Original Title", EventDate: time.Now().Add(time.Hour), Price: 1000,
		TotalTickets: 10, AvailableTickets: 10, CategoryID: "cat-1",
	}
	require.NoError(t, eventRepo.Create(ctx, event))

	event.Title = "Updated Title"
	require.NoError(t, eventRepo.Update(ctx, event))

	fetched, err := eventRepo.GetByID(ctx, "evt-1")
	require.NoError(t, err)
	assert.Equal(t, "Updated Title", fetched.Title)

	require.NoError(t, eventRepo.Delete(ctx, "evt-1"))
	fetched, err = eventRepo.GetByID(ctx, "evt-1")
	require.NoError(t, err)
	assert.Nil(t, fetched)
}
