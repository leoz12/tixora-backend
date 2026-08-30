package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"tixora/internal/models"
	"tixora/internal/services"
	"tixora/internal/utils"
)

func newEventService(eventRepo *mockEventRepo, categoryRepo *mockCategoryRepo, fileRepo *mockFileRepo) services.IEventService {
	return services.NewEventService(eventRepo, categoryRepo, fileRepo)
}

func TestEventService_GetEvents_DefaultsInvalidPagination(t *testing.T) {
	eventRepo := new(mockEventRepo)
	eventRepo.On("GetWithPagination", mock.Anything, 0, 12, "", "", true).
		Return([]models.Event{{ID: "evt-1"}}, 1, nil)

	svc := newEventService(eventRepo, new(mockCategoryRepo), new(mockFileRepo))

	events, total, err := svc.GetEvents(context.Background(), 0, 0, "", "", true)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.EqualValues(t, 1, total)
	eventRepo.AssertExpectations(t)
}

func TestEventService_GetEvents_ClampsOversizedLimit(t *testing.T) {
	eventRepo := new(mockEventRepo)
	eventRepo.On("GetWithPagination", mock.Anything, 0, 12, "", "", false).
		Return([]models.Event{}, 0, nil)

	svc := newEventService(eventRepo, new(mockCategoryRepo), new(mockFileRepo))

	_, _, err := svc.GetEvents(context.Background(), 1, 500, "", "", false)
	require.NoError(t, err)
	eventRepo.AssertExpectations(t)
}

func TestEventService_SearchEvents_EmptyQueryRejected(t *testing.T) {
	svc := newEventService(new(mockEventRepo), new(mockCategoryRepo), new(mockFileRepo))

	_, _, err := svc.SearchEvents(context.Background(), "", 1, 12, true)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_GetEventByID_EmptyIDRejected(t *testing.T) {
	svc := newEventService(new(mockEventRepo), new(mockCategoryRepo), new(mockFileRepo))

	_, err := svc.GetEventByID(context.Background(), "")
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_GetEventByID_NotFound(t *testing.T) {
	eventRepo := new(mockEventRepo)
	eventRepo.On("GetByID", mock.Anything, "missing").Return(nil, nil)

	svc := newEventService(eventRepo, new(mockCategoryRepo), new(mockFileRepo))

	_, err := svc.GetEventByID(context.Background(), "missing")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func validEventInput() *models.Event {
	return &models.Event{
		Title:            "Concert Night",
		EventDate:        time.Now().Add(24 * time.Hour),
		Location:         "Jakarta",
		Price:            100000,
		TotalTickets:     100,
		AvailableTickets: 100,
		CategoryID:       "cat-1",
	}
}

func TestEventService_CreateEvent_Success(t *testing.T) {
	eventRepo := new(mockEventRepo)
	categoryRepo := new(mockCategoryRepo)
	fileRepo := new(mockFileRepo)

	categoryRepo.On("GetByID", mock.Anything, "cat-1").
		Return(&models.Category{ID: "cat-1", IsActive: true}, nil)
	eventRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Event")).Return(nil)

	svc := newEventService(eventRepo, categoryRepo, fileRepo)

	event := validEventInput()
	err := svc.CreateEvent(context.Background(), event)
	require.NoError(t, err)
	assert.NotEmpty(t, event.ID)
	eventRepo.AssertExpectations(t)
	categoryRepo.AssertExpectations(t)
}

func TestEventService_CreateEvent_MissingTitle(t *testing.T) {
	svc := newEventService(new(mockEventRepo), new(mockCategoryRepo), new(mockFileRepo))

	event := validEventInput()
	event.Title = ""

	err := svc.CreateEvent(context.Background(), event)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_CreateEvent_NegativePrice(t *testing.T) {
	svc := newEventService(new(mockEventRepo), new(mockCategoryRepo), new(mockFileRepo))

	event := validEventInput()
	event.Price = -1

	err := svc.CreateEvent(context.Background(), event)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_CreateEvent_ZeroTotalTickets(t *testing.T) {
	svc := newEventService(new(mockEventRepo), new(mockCategoryRepo), new(mockFileRepo))

	event := validEventInput()
	event.TotalTickets = 0

	err := svc.CreateEvent(context.Background(), event)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_CreateEvent_AvailableExceedsTotal(t *testing.T) {
	svc := newEventService(new(mockEventRepo), new(mockCategoryRepo), new(mockFileRepo))

	event := validEventInput()
	event.AvailableTickets = event.TotalTickets + 1

	err := svc.CreateEvent(context.Background(), event)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_CreateEvent_CategoryNotFound(t *testing.T) {
	categoryRepo := new(mockCategoryRepo)
	categoryRepo.On("GetByID", mock.Anything, "cat-1").Return(nil, nil)

	svc := newEventService(new(mockEventRepo), categoryRepo, new(mockFileRepo))

	err := svc.CreateEvent(context.Background(), validEventInput())
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_CreateEvent_InactiveCategoryRejected(t *testing.T) {
	categoryRepo := new(mockCategoryRepo)
	categoryRepo.On("GetByID", mock.Anything, "cat-1").
		Return(&models.Category{ID: "cat-1", IsActive: false}, nil)

	svc := newEventService(new(mockEventRepo), categoryRepo, new(mockFileRepo))

	err := svc.CreateEvent(context.Background(), validEventInput())
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_CreateEvent_ImageNotFound(t *testing.T) {
	categoryRepo := new(mockCategoryRepo)
	categoryRepo.On("GetByID", mock.Anything, "cat-1").
		Return(&models.Category{ID: "cat-1", IsActive: true}, nil)
	fileRepo := new(mockFileRepo)
	imageID := "missing-file"
	fileRepo.On("GetByID", mock.Anything, imageID).Return(nil, nil)

	svc := newEventService(new(mockEventRepo), categoryRepo, fileRepo)

	event := validEventInput()
	event.ImageID = &imageID

	err := svc.CreateEvent(context.Background(), event)
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_CreateEvent_MarksImageAttached(t *testing.T) {
	categoryRepo := new(mockCategoryRepo)
	categoryRepo.On("GetByID", mock.Anything, "cat-1").
		Return(&models.Category{ID: "cat-1", IsActive: true}, nil)

	fileRepo := new(mockFileRepo)
	imageID := "file-1"
	pendingFile := &models.File{ID: imageID, Status: models.FileStatusPending}
	fileRepo.On("GetByID", mock.Anything, imageID).Return(pendingFile, nil)
	fileRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.File")).
		Run(func(args mock.Arguments) {
			f := args.Get(1).(*models.File)
			assert.Equal(t, models.FileStatusAttached, f.Status)
		}).
		Return(nil)

	eventRepo := new(mockEventRepo)
	eventRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Event")).Return(nil)

	svc := newEventService(eventRepo, categoryRepo, fileRepo)

	event := validEventInput()
	event.ImageID = &imageID

	err := svc.CreateEvent(context.Background(), event)
	require.NoError(t, err)
	fileRepo.AssertExpectations(t)
}

func TestEventService_UpdateEvent_NotFound(t *testing.T) {
	categoryRepo := new(mockCategoryRepo)
	categoryRepo.On("GetByID", mock.Anything, "cat-1").
		Return(&models.Category{ID: "cat-1", IsActive: true}, nil)
	eventRepo := new(mockEventRepo)
	eventRepo.On("GetByID", mock.Anything, "evt-1").Return(nil, nil)

	svc := newEventService(eventRepo, categoryRepo, new(mockFileRepo))

	event := validEventInput()
	event.ID = "evt-1"

	err := svc.UpdateEvent(context.Background(), event)
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestEventService_UpdateEvent_MissingIDRejected(t *testing.T) {
	svc := newEventService(new(mockEventRepo), new(mockCategoryRepo), new(mockFileRepo))

	err := svc.UpdateEvent(context.Background(), validEventInput())
	assert.ErrorIs(t, err, utils.ErrInvalidInput)
}

func TestEventService_DeleteEvent_NotFound(t *testing.T) {
	eventRepo := new(mockEventRepo)
	eventRepo.On("GetByID", mock.Anything, "evt-1").Return(nil, nil)

	svc := newEventService(eventRepo, new(mockCategoryRepo), new(mockFileRepo))

	err := svc.DeleteEvent(context.Background(), "evt-1")
	assert.ErrorIs(t, err, utils.ErrNotFound)
}

func TestEventService_DeleteEvent_Success(t *testing.T) {
	eventRepo := new(mockEventRepo)
	eventRepo.On("GetByID", mock.Anything, "evt-1").Return(&models.Event{ID: "evt-1"}, nil)
	eventRepo.On("Delete", mock.Anything, "evt-1").Return(nil)

	svc := newEventService(eventRepo, new(mockCategoryRepo), new(mockFileRepo))

	err := svc.DeleteEvent(context.Background(), "evt-1")
	require.NoError(t, err)
	eventRepo.AssertExpectations(t)
}
