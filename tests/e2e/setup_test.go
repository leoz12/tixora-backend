// Package e2e drives the full HTTP stack (router -> middleware -> handlers
// -> services -> repositories -> a real MySQL database) the same way
// cmd/server/main.go wires it, verifying that every layer is connected
// correctly end to end. It intentionally does not hit Google or Midtrans:
//   - Google OAuth login can't be exercised without a live external account,
//     so user sessions are established by minting a JWT/cookie directly for a
//     seeded user (bypassing HandleGoogleCallback, not GetCurrentUser/orders/etc).
//   - Midtrans's CreateTransaction call is replaced with a stub that returns a
//     canned Snap URL/token, so order creation doesn't depend on Midtrans's
//     sandbox being reachable; ProcessWebhook (the inbound side) is real and is
//     exercised directly with a correctly-signed notification payload.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"tixora/internal/config"
	"tixora/internal/handlers"
	"tixora/internal/middleware"
	"tixora/internal/models"
	"tixora/internal/repository"
	"tixora/internal/services"
	"tixora/internal/utils"
)

const testMidtransServerKey = "SB-Mid-server-e2e-test-key"

var sharedDB *gorm.DB

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx := context.Background()

	container, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("tixora_e2e"),
		mysql.WithUsername("tixora"),
		mysql.WithPassword("tixora"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e tests: failed to start MySQL container (is Docker running?): %v\n", err)
		fmt.Fprintln(os.Stderr, "skipping e2e tests - run `go test -short ./...` to skip explicitly next time")
		os.Exit(0)
	}
	defer func() { _ = container.Terminate(ctx) }()

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "charset=utf8mb4", "loc=Local")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e tests: failed to build DSN: %v\n", err)
		os.Exit(1)
	}

	db, err := gorm.Open(gormmysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e tests: failed to connect: %v\n", err)
		os.Exit(1)
	}

	if err := db.AutoMigrate(
		&models.User{}, &models.Admin{}, &models.RefreshToken{},
		&models.Category{}, &models.File{}, &models.Event{},
		&models.Order{}, &models.Payment{},
	); err != nil {
		fmt.Fprintf(os.Stderr, "e2e tests: failed to migrate: %v\n", err)
		os.Exit(1)
	}

	sharedDB = db
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

// testApp is a fully wired router plus the config/db it was built with, for
// tests that need to seed data or mint tokens directly.
type testApp struct {
	router *gin.Engine
	db     *gorm.DB
	cfg    *config.Config
}

// fakePaymentService wraps a real IPaymentService but replaces
// CreateTransaction with a stub, so order creation doesn't call the real
// Midtrans sandbox. ProcessWebhook/GetPaymentStatus are the real
// implementation, so webhook e2e tests still exercise real logic.
type fakePaymentService struct {
	services.IPaymentService
}

func (f *fakePaymentService) CreateTransaction(ctx context.Context, order *models.Order, itemName, customerName, customerEmail string) (string, string, error) {
	return "https://snap.example.test/redirect/" + order.OrderID, "fake-snap-token-" + order.OrderID, nil
}

// newTestApp builds a full router - identical wiring to cmd/server/main.go -
// backed by a transaction on the shared test database, rolled back
// automatically when the test ends so every test starts from a clean slate
// without the cost of a fresh container or schema per test.
func newTestApp(t *testing.T) *testApp {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping e2e test in -short mode")
	}
	if sharedDB == nil {
		t.Skip("no test database available (container failed to start - see TestMain output)")
	}

	tx := sharedDB.Begin()
	t.Cleanup(func() { tx.Rollback() })

	cfg := &config.Config{
		JWTSecret:         "e2e_test_secret_min_32_chars_long",
		JWTAccessExpiry:   15 * time.Minute,
		JWTRefreshExpiry:  168 * time.Hour,
		MidtransServerKey: testMidtransServerKey,
		MidtransIsSandbox: true,
		CookieSecure:      false,
		CORSOrigins:       []string{"http://localhost:3000"},
		R2PublicBaseURL:   "https://cdn.test.local",
	}

	userRepo := repository.NewUserRepository(tx)
	adminRepo := repository.NewAdminRepository(tx)
	refreshTokenRepo := repository.NewRefreshTokenRepository(tx)
	categoryRepo := repository.NewCategoryRepository(tx)
	fileRepo := repository.NewFileRepository(tx)
	eventRepo := repository.NewEventRepository(tx)
	orderRepo := repository.NewOrderRepository(tx)
	paymentRepo := repository.NewPaymentRepository(tx)

	authService := services.NewAuthService(userRepo, refreshTokenRepo, cfg)
	userService := services.NewUserService(userRepo)
	adminService := services.NewAdminService(adminRepo, refreshTokenRepo, cfg)
	categoryService := services.NewCategoryService(categoryRepo, eventRepo)
	realPaymentService := services.NewPaymentService(paymentRepo, orderRepo, eventRepo, cfg.MidtransServerKey, cfg.MidtransIsSandbox)
	paymentService := &fakePaymentService{IPaymentService: realPaymentService}
	eventService := services.NewEventService(eventRepo, categoryRepo, fileRepo)
	orderService := services.NewOrderService(orderRepo, eventRepo, userRepo, paymentService)

	authHandler := handlers.NewAuthHandler(authService, userService, cfg)
	userHandler := handlers.NewUserHandler(userService, orderService)
	adminHandler := handlers.NewAdminHandler(adminService, cfg)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	eventHandler := handlers.NewEventHandler(eventService, cfg.R2PublicBaseURL)
	orderHandler := handlers.NewOrderHandler(orderService, cfg.R2PublicBaseURL)
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	router := gin.New()
	router.Use(gin.Recovery())
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		t.Fatalf("newTestApp: set trusted proxies: %v", err)
	}
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.Use(middleware.CORSMiddleware(cfg.CORSOrigins))
	router.Use(middleware.ErrorMiddleware())

	api := router.Group("/api")
	{
		auth := api.Group("/auth")
		auth.POST("/oauth/google/callback", authHandler.GoogleCallback)
		auth.POST("/refresh", authHandler.RefreshToken)
		authProtected := auth.Group("")
		authProtected.Use(middleware.AuthMiddleware(cfg.JWTSecret), middleware.CSRFMiddleware(cfg.CORSOrigins))
		{
			authProtected.GET("/me", authHandler.GetCurrentUser)
			authProtected.POST("/logout", authHandler.Logout)
		}

		events := api.Group("/events")
		events.GET("", eventHandler.GetEvents)
		events.GET("/search", eventHandler.SearchEvents)
		events.GET("/:id", eventHandler.GetEventByID)
		eventsProtected := events.Group("")
		eventsProtected.Use(middleware.AdminAuthMiddleware(cfg.JWTSecret), middleware.CSRFMiddleware(cfg.CORSOrigins))
		{
			eventsProtected.POST("", eventHandler.CreateEvent)
			eventsProtected.PUT("/:id", eventHandler.UpdateEvent)
			eventsProtected.DELETE("/:id", eventHandler.DeleteEvent)
		}

		categories := api.Group("/categories")
		categories.GET("", categoryHandler.ListCategories)
		categories.GET("/:id", categoryHandler.GetCategoryByID)
		categoriesProtected := categories.Group("")
		categoriesProtected.Use(middleware.AdminAuthMiddleware(cfg.JWTSecret), middleware.CSRFMiddleware(cfg.CORSOrigins))
		{
			categoriesProtected.POST("", categoryHandler.CreateCategory)
			categoriesProtected.PUT("/:id", categoryHandler.UpdateCategory)
			categoriesProtected.DELETE("/:id", categoryHandler.DeleteCategory)
		}

		orders := api.Group("/orders")
		orders.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		orders.Use(middleware.CSRFMiddleware(cfg.CORSOrigins))
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.GetUserOrders)
			orders.GET("/:id", orderHandler.GetOrderByID)
			orders.GET("/:id/status", orderHandler.GetOrderStatus)
			orders.GET("/:id/ticket/download", orderHandler.DownloadTicket)
			orders.POST("/:id/cancel", orderHandler.CancelOrder)
			orders.POST("/:id/pay", orderHandler.ContinuePayment)
		}

		payments := api.Group("/payments")
		payments.POST("/webhook", paymentHandler.WebhookHandler)
		paymentsProtected := payments.Group("")
		paymentsProtected.Use(middleware.AuthMiddleware(cfg.JWTSecret), middleware.CSRFMiddleware(cfg.CORSOrigins))
		{
			paymentsProtected.GET("/status/:transactionId", paymentHandler.GetPaymentStatus)
		}

		user := api.Group("/user")
		user.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		user.Use(middleware.CSRFMiddleware(cfg.CORSOrigins))
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.GET("/stats", userHandler.GetStats)
		}

		admin := api.Group("/admin")
		admin.POST("/auth/login", adminHandler.Login)
		admin.POST("/auth/refresh", adminHandler.RefreshToken)
		adminProtected := admin.Group("")
		adminProtected.Use(middleware.AdminAuthMiddleware(cfg.JWTSecret), middleware.CSRFMiddleware(cfg.CORSOrigins))
		{
			adminProtected.GET("/auth/me", adminHandler.GetCurrentAdmin)
			adminProtected.POST("/auth/logout", adminHandler.Logout)
			adminProtected.GET("/admins", adminHandler.ListAdmins)
			adminProtected.POST("/admins", adminHandler.CreateAdmin)
			adminProtected.PUT("/admins/:id", adminHandler.UpdateAdmin)
			adminProtected.DELETE("/admins/:id", adminHandler.DeleteAdmin)
			adminProtected.GET("/orders", orderHandler.AdminListOrders)
			adminProtected.GET("/orders/:id", orderHandler.AdminGetOrderByID)
			adminProtected.GET("/users", userHandler.AdminListUsers)
			adminProtected.GET("/users/:id", userHandler.AdminGetUserByID)
		}
	}

	return &testApp{router: router, db: tx, cfg: cfg}
}

// --- seeding + request helpers shared by every *_e2e_test.go file ---

func (a *testApp) seedCategory(t *testing.T, id, name, slug string) *models.Category {
	t.Helper()
	category := &models.Category{ID: id, Name: name, Slug: slug, IsActive: true}
	if err := repository.NewCategoryRepository(a.db).Create(context.Background(), category); err != nil {
		t.Fatalf("seedCategory: %v", err)
	}
	return category
}

func (a *testApp) seedEvent(t *testing.T, id, categoryID string, availableTickets int) *models.Event {
	t.Helper()
	event := &models.Event{
		ID: id, Title: "Test Event " + id, EventDate: time.Now().Add(24 * time.Hour),
		Location: "Jakarta", Price: 100000, TotalTickets: availableTickets,
		AvailableTickets: availableTickets, CategoryID: categoryID,
	}
	if err := repository.NewEventRepository(a.db).Create(context.Background(), event); err != nil {
		t.Fatalf("seedEvent: %v", err)
	}
	return event
}

func (a *testApp) seedUser(t *testing.T, id, email string) *models.User {
	t.Helper()
	user := &models.User{ID: id, Email: email, Name: "Test User", GoogleID: "google-" + id}
	if err := repository.NewUserRepository(a.db).Create(context.Background(), user); err != nil {
		t.Fatalf("seedUser: %v", err)
	}
	return user
}

func (a *testApp) seedAdmin(t *testing.T, id, email, password, role string) *models.Admin {
	t.Helper()
	hash, err := utils.HashPassword(password)
	if err != nil {
		t.Fatalf("seedAdmin: hash password: %v", err)
	}
	admin := &models.Admin{ID: id, Email: email, Name: "Test Admin", PasswordHash: hash, Role: role}
	if err := repository.NewAdminRepository(a.db).Create(context.Background(), admin); err != nil {
		t.Fatalf("seedAdmin: %v", err)
	}
	return admin
}

// authedCookies mints a valid access-token cookie directly (bypassing the
// Google OAuth / admin-password login flow) for a seeded principal, ready to
// attach to an *http.Request. Set origin to a value to simulate a browser
// request from that page - CSRF protection is an Origin/Referer allowlist
// check, so a disallowed origin is how a cross-site (CSRF) attempt is modeled.
type authedCookies struct {
	accessCookie *http.Cookie
	origin       string
}

func (c authedCookies) attach(req *http.Request) {
	req.AddCookie(c.accessCookie)
	if c.origin != "" {
		req.Header.Set("Origin", c.origin)
	}
}

func (a *testApp) userCookies(t *testing.T, userID, email string) authedCookies {
	t.Helper()
	token, err := utils.GenerateJWT(userID, email, a.cfg.JWTSecret, a.cfg.JWTAccessExpiry)
	if err != nil {
		t.Fatalf("userCookies: %v", err)
	}
	return authedCookies{
		accessCookie: &http.Cookie{Name: utils.UserAccessCookie, Value: token},
	}
}

func (a *testApp) adminCookies(t *testing.T, adminID, email, role string) authedCookies {
	t.Helper()
	token, err := utils.GenerateAdminJWT(adminID, email, role, a.cfg.JWTSecret, a.cfg.JWTAccessExpiry)
	if err != nil {
		t.Fatalf("adminCookies: %v", err)
	}
	return authedCookies{
		accessCookie: &http.Cookie{Name: utils.AdminAccessCookie, Value: token},
	}
}

// do issues an HTTP request against the app's router and decodes the JSON
// response body into out (when out is non-nil).
func (a *testApp) do(t *testing.T, method, path string, body interface{}, cookies *authedCookies, out interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var req *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("do: marshal body: %v", err)
		}
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if cookies != nil {
		cookies.attach(req)
	}

	rec := httptest.NewRecorder()
	a.router.ServeHTTP(rec, req)

	if out != nil && rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("do: unmarshal response body %q: %v", rec.Body.String(), err)
		}
	}

	return rec
}
