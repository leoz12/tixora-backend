package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"tixora/internal/models"
	"tixora/internal/repository"
)

// seedUserAndEvent creates the User/Category/Event rows an Order references,
// returning their IDs, so order tests don't depend on the other repositories.
func seedUserAndEvent(t *testing.T, db *gorm.DB, suffix string) (userID, eventID string) {
	t.Helper()
	ctx := context.Background()

	userID = "user-" + suffix
	require.NoError(t, repository.NewUserRepository(db).Create(ctx, &models.User{
		ID: userID, Email: userID + "@example.com", Name: "Test User", GoogleID: "google-" + suffix,
	}))

	catID := "cat-" + suffix
	require.NoError(t, repository.NewCategoryRepository(db).Create(ctx, newTestCategory(catID, "Category "+suffix, "category-"+suffix, true)))

	eventID = "evt-" + suffix
	require.NoError(t, repository.NewEventRepository(db).Create(ctx, &models.Event{
		ID: eventID, Title: "Event " + suffix, EventDate: time.Now().Add(time.Hour), Price: 100000,
		TotalTickets: 100, AvailableTickets: 100, CategoryID: catID,
	}))
	return userID, eventID
}

func newTestOrder(id, orderID, userID, eventID string, status models.OrderStatus) *models.Order {
	return &models.Order{
		ID: id, OrderID: orderID, UserID: userID, EventID: eventID,
		Quantity: 2, BuyerName: "Buyer", BuyerEmail: "buyer@example.com",
		UnitPrice: 100000, Subtotal: 200000, AdminFee: 5000, TotalPrice: 205000,
		Status: status,
	}
}

func TestOrderRepository_CreateAndGetByID(t *testing.T) {
	db := requireDB(t)
	userID, eventID := seedUserAndEvent(t, db, "1")
	repo := repository.NewOrderRepository(db)
	ctx := context.Background()

	order := newTestOrder("row-1", "ORD-1", userID, eventID, models.OrderStatusPending)
	require.NoError(t, repo.Create(ctx, order))

	fetched, err := repo.GetByID(ctx, "row-1")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "ORD-1", fetched.OrderID)
}

func TestOrderRepository_GetByOrderID_PreloadsEventAndUser(t *testing.T) {
	db := requireDB(t)
	userID, eventID := seedUserAndEvent(t, db, "2")
	repo := repository.NewOrderRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestOrder("row-1", "ORD-2", userID, eventID, models.OrderStatusPending)))

	fetched, err := repo.GetByOrderID(ctx, "ORD-2")
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, "Event 2", fetched.Event.Title)
	assert.Equal(t, userID, fetched.User.ID)
}

func TestOrderRepository_GetByOrderID_NotFound(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewOrderRepository(db)

	fetched, err := repo.GetByOrderID(context.Background(), "ORD-missing")
	require.NoError(t, err)
	assert.Nil(t, fetched)
}

func TestOrderRepository_GetByUserID_FiltersByStatus(t *testing.T) {
	db := requireDB(t)
	userID, eventID := seedUserAndEvent(t, db, "3")
	repo := repository.NewOrderRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestOrder("row-1", "ORD-3a", userID, eventID, models.OrderStatusPending)))
	require.NoError(t, repo.Create(ctx, newTestOrder("row-2", "ORD-3b", userID, eventID, models.OrderStatusPaid)))

	pending, total, err := repo.GetByUserID(ctx, userID, models.OrderStatusPending, 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, pending, 1)
	assert.Equal(t, "ORD-3a", pending[0].OrderID)

	all, total, err := repo.GetByUserID(ctx, userID, "", 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, all, 2)
}

func TestOrderRepository_GetWithPagination_AcrossAllUsers(t *testing.T) {
	db := requireDB(t)
	user1, event1 := seedUserAndEvent(t, db, "4a")
	user2, event2 := seedUserAndEvent(t, db, "4b")
	repo := repository.NewOrderRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestOrder("row-1", "ORD-4a", user1, event1, models.OrderStatusPending)))
	require.NoError(t, repo.Create(ctx, newTestOrder("row-2", "ORD-4b", user2, event2, models.OrderStatusPending)))

	orders, total, err := repo.GetWithPagination(ctx, models.OrderStatusPending, 0, 10)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	assert.Len(t, orders, 2)
}

func TestOrderRepository_GetUserOrderStats_OnlyCountsPaidOrders(t *testing.T) {
	db := requireDB(t)
	userID, eventID := seedUserAndEvent(t, db, "5")
	repo := repository.NewOrderRepository(db)
	ctx := context.Background()

	paid := newTestOrder("row-1", "ORD-5a", userID, eventID, models.OrderStatusPaid)
	paid.TotalPrice = 205000
	require.NoError(t, repo.Create(ctx, paid))

	pending := newTestOrder("row-2", "ORD-5b", userID, eventID, models.OrderStatusPending)
	pending.TotalPrice = 300000
	require.NoError(t, repo.Create(ctx, pending))

	totalOrders, totalSpent, err := repo.GetUserOrderStats(ctx, userID)
	require.NoError(t, err)
	assert.EqualValues(t, 1, totalOrders)
	assert.EqualValues(t, 205000, totalSpent)
}

func TestOrderRepository_UpdateStatus(t *testing.T) {
	db := requireDB(t)
	userID, eventID := seedUserAndEvent(t, db, "6")
	repo := repository.NewOrderRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestOrder("row-1", "ORD-6", userID, eventID, models.OrderStatusPending)))
	require.NoError(t, repo.UpdateStatus(ctx, "row-1", models.OrderStatusPaid))

	fetched, err := repo.GetByID(ctx, "row-1")
	require.NoError(t, err)
	assert.Equal(t, models.OrderStatusPaid, fetched.Status)
}

func TestOrderRepository_UpdateStatus_NonexistentRowReturnsError(t *testing.T) {
	db := requireDB(t)
	repo := repository.NewOrderRepository(db)

	err := repo.UpdateStatus(context.Background(), "does-not-exist", models.OrderStatusPaid)
	assert.Error(t, err)
}

func TestOrderRepository_UniqueOrderIDConstraint(t *testing.T) {
	db := requireDB(t)
	userID, eventID := seedUserAndEvent(t, db, "7")
	repo := repository.NewOrderRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, newTestOrder("row-1", "ORD-DUP", userID, eventID, models.OrderStatusPending)))

	err := repo.Create(ctx, newTestOrder("row-2", "ORD-DUP", userID, eventID, models.OrderStatusPending))
	assert.Error(t, err)
}
