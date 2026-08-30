package services_test

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"tixora/internal/models"
	"tixora/internal/services"
)

// Shared testify/mock doubles for every repository interface (and the
// cross-service IPaymentService/IStorageService dependencies), reused across
// this package's *_test.go files.

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*models.User, error) {
	args := m.Called(ctx, id)
	return userOrNil(args.Get(0)), args.Error(1)
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	return userOrNil(args.Get(0)), args.Error(1)
}
func (m *mockUserRepo) GetByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	args := m.Called(ctx, googleID)
	return userOrNil(args.Get(0)), args.Error(1)
}
func (m *mockUserRepo) List(ctx context.Context, search string, offset, limit int) ([]models.User, int64, error) {
	args := m.Called(ctx, search, offset, limit)
	return sliceOrNil[models.User](args.Get(0)), int64(args.Int(1)), args.Error(2)
}
func (m *mockUserRepo) Create(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}
func (m *mockUserRepo) Update(ctx context.Context, user *models.User) error {
	return m.Called(ctx, user).Error(0)
}

func userOrNil(v interface{}) *models.User {
	if v == nil {
		return nil
	}
	return v.(*models.User)
}

func sliceOrNil[T any](v interface{}) []T {
	if v == nil {
		return nil
	}
	return v.([]T)
}

type mockAdminRepo struct{ mock.Mock }

func (m *mockAdminRepo) GetByID(ctx context.Context, id string) (*models.Admin, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}
func (m *mockAdminRepo) GetByEmail(ctx context.Context, email string) (*models.Admin, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Admin), args.Error(1)
}
func (m *mockAdminRepo) List(ctx context.Context, offset, limit int) ([]models.Admin, int64, error) {
	args := m.Called(ctx, offset, limit)
	return sliceOrNil[models.Admin](args.Get(0)), int64(args.Int(1)), args.Error(2)
}
func (m *mockAdminRepo) Create(ctx context.Context, admin *models.Admin) error {
	return m.Called(ctx, admin).Error(0)
}
func (m *mockAdminRepo) Update(ctx context.Context, admin *models.Admin) error {
	return m.Called(ctx, admin).Error(0)
}
func (m *mockAdminRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

type mockRefreshTokenRepo struct{ mock.Mock }

func (m *mockRefreshTokenRepo) Create(ctx context.Context, token *models.RefreshToken) error {
	return m.Called(ctx, token).Error(0)
}
func (m *mockRefreshTokenRepo) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}
func (m *mockRefreshTokenRepo) Revoke(ctx context.Context, tokenHash string, replacedBy *string) error {
	return m.Called(ctx, tokenHash, replacedBy).Error(0)
}
func (m *mockRefreshTokenRepo) RevokeAllForSubject(ctx context.Context, subjectID string, subjectType models.SubjectType) error {
	return m.Called(ctx, subjectID, subjectType).Error(0)
}

type mockCategoryRepo struct{ mock.Mock }

func (m *mockCategoryRepo) List(ctx context.Context, includeInactive bool) ([]models.Category, error) {
	args := m.Called(ctx, includeInactive)
	return sliceOrNil[models.Category](args.Get(0)), args.Error(1)
}
func (m *mockCategoryRepo) ListWithPagination(ctx context.Context, offset, limit int) ([]models.Category, int64, error) {
	args := m.Called(ctx, offset, limit)
	return sliceOrNil[models.Category](args.Get(0)), int64(args.Int(1)), args.Error(2)
}
func (m *mockCategoryRepo) GetByID(ctx context.Context, id string) (*models.Category, error) {
	args := m.Called(ctx, id)
	return categoryOrNil(args.Get(0)), args.Error(1)
}
func (m *mockCategoryRepo) GetByName(ctx context.Context, name string) (*models.Category, error) {
	args := m.Called(ctx, name)
	return categoryOrNil(args.Get(0)), args.Error(1)
}
func (m *mockCategoryRepo) GetBySlug(ctx context.Context, slug string) (*models.Category, error) {
	args := m.Called(ctx, slug)
	return categoryOrNil(args.Get(0)), args.Error(1)
}
func (m *mockCategoryRepo) Create(ctx context.Context, category *models.Category) error {
	return m.Called(ctx, category).Error(0)
}
func (m *mockCategoryRepo) Update(ctx context.Context, category *models.Category) error {
	return m.Called(ctx, category).Error(0)
}
func (m *mockCategoryRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func categoryOrNil(v interface{}) *models.Category {
	if v == nil {
		return nil
	}
	return v.(*models.Category)
}

type mockEventRepo struct{ mock.Mock }

func (m *mockEventRepo) GetWithPagination(ctx context.Context, offset, limit int, categoryID, search string, includePast bool) ([]models.Event, int64, error) {
	args := m.Called(ctx, offset, limit, categoryID, search, includePast)
	return sliceOrNil[models.Event](args.Get(0)), int64(args.Int(1)), args.Error(2)
}
func (m *mockEventRepo) GetByID(ctx context.Context, id string) (*models.Event, error) {
	args := m.Called(ctx, id)
	return eventOrNil(args.Get(0)), args.Error(1)
}
func (m *mockEventRepo) Search(ctx context.Context, query string, offset, limit int, includePast bool) ([]models.Event, int64, error) {
	args := m.Called(ctx, query, offset, limit, includePast)
	return sliceOrNil[models.Event](args.Get(0)), int64(args.Int(1)), args.Error(2)
}
func (m *mockEventRepo) CountByCategoryID(ctx context.Context, categoryID string) (int64, error) {
	args := m.Called(ctx, categoryID)
	return int64(args.Int(0)), args.Error(1)
}
func (m *mockEventRepo) Create(ctx context.Context, event *models.Event) error {
	return m.Called(ctx, event).Error(0)
}
func (m *mockEventRepo) Update(ctx context.Context, event *models.Event) error {
	return m.Called(ctx, event).Error(0)
}
func (m *mockEventRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func eventOrNil(v interface{}) *models.Event {
	if v == nil {
		return nil
	}
	return v.(*models.Event)
}

type mockFileRepo struct{ mock.Mock }

func (m *mockFileRepo) Create(ctx context.Context, file *models.File) error {
	return m.Called(ctx, file).Error(0)
}
func (m *mockFileRepo) GetByID(ctx context.Context, id string) (*models.File, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.File), args.Error(1)
}
func (m *mockFileRepo) Update(ctx context.Context, file *models.File) error {
	return m.Called(ctx, file).Error(0)
}

type mockOrderRepo struct{ mock.Mock }

func (m *mockOrderRepo) Create(ctx context.Context, order *models.Order) error {
	return m.Called(ctx, order).Error(0)
}
func (m *mockOrderRepo) GetByID(ctx context.Context, id string) (*models.Order, error) {
	args := m.Called(ctx, id)
	return orderOrNil(args.Get(0)), args.Error(1)
}
func (m *mockOrderRepo) GetByOrderID(ctx context.Context, orderID string) (*models.Order, error) {
	args := m.Called(ctx, orderID)
	return orderOrNil(args.Get(0)), args.Error(1)
}
func (m *mockOrderRepo) GetByUserID(ctx context.Context, userID string, status models.OrderStatus, offset, limit int) ([]models.Order, int64, error) {
	args := m.Called(ctx, userID, status, offset, limit)
	return sliceOrNil[models.Order](args.Get(0)), int64(args.Int(1)), args.Error(2)
}
func (m *mockOrderRepo) GetWithPagination(ctx context.Context, status models.OrderStatus, offset, limit int) ([]models.Order, int64, error) {
	args := m.Called(ctx, status, offset, limit)
	return sliceOrNil[models.Order](args.Get(0)), int64(args.Int(1)), args.Error(2)
}
func (m *mockOrderRepo) GetUserOrderStats(ctx context.Context, userID string) (int64, int64, error) {
	args := m.Called(ctx, userID)
	return int64(args.Int(0)), int64(args.Int(1)), args.Error(2)
}
func (m *mockOrderRepo) Update(ctx context.Context, order *models.Order) error {
	return m.Called(ctx, order).Error(0)
}
func (m *mockOrderRepo) UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error {
	return m.Called(ctx, id, status).Error(0)
}

func orderOrNil(v interface{}) *models.Order {
	if v == nil {
		return nil
	}
	return v.(*models.Order)
}

type mockPaymentRepo struct{ mock.Mock }

func (m *mockPaymentRepo) Create(ctx context.Context, payment *models.Payment) error {
	return m.Called(ctx, payment).Error(0)
}
func (m *mockPaymentRepo) GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error) {
	args := m.Called(ctx, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}
func (m *mockPaymentRepo) UpdateStatus(ctx context.Context, id string, status models.PaymentStatus) error {
	return m.Called(ctx, id, status).Error(0)
}

// mockPaymentService doubles IPaymentService, the dependency OrderService
// uses to create a Midtrans Snap transaction.
type mockPaymentService struct{ mock.Mock }

func (m *mockPaymentService) CreateTransaction(ctx context.Context, order *models.Order, itemName, customerName, customerEmail string) (string, string, error) {
	args := m.Called(ctx, order, itemName, customerName, customerEmail)
	return args.String(0), args.String(1), args.Error(2)
}
func (m *mockPaymentService) ProcessWebhook(ctx context.Context, notification services.MidtransNotification) error {
	return m.Called(ctx, notification).Error(0)
}
func (m *mockPaymentService) GetPaymentStatus(ctx context.Context, userID, transactionID string) (*models.Payment, error) {
	args := m.Called(ctx, userID, transactionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Payment), args.Error(1)
}

// mockStorageService doubles IStorageService, the dependency FileService
// uses to presign an R2 upload URL.
type mockStorageService struct{ mock.Mock }

func (m *mockStorageService) PresignPutURL(ctx context.Context, objectKey, contentType string, expiry time.Duration) (string, error) {
	args := m.Called(ctx, objectKey, contentType, expiry)
	return args.String(0), args.Error(1)
}
