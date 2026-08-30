package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "tixora/docs"
	"tixora/internal/config"
	"tixora/internal/handlers"
	"tixora/internal/middleware"
	"tixora/internal/repository"
	"tixora/internal/services"
)

// @title						Tixora API
// @version						1.0
// @description					REST API for the Tixora event ticket marketplace.
// @contact.name				Tixora Support
// @license.name				Proprietary
// @host						localhost:8000
// @BasePath					/api
// @securityDefinitions.apikey	CookieAuth
// @in							cookie
// @name						tx_access_token
// @description					httpOnly access token cookie, set automatically on login (see /auth/oauth/google/callback, /admin/auth/login).
func main() {
	// Load config
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Create repositories
	userRepo := repository.NewUserRepository(db)
	adminRepo := repository.NewAdminRepository(db)
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	fileRepo := repository.NewFileRepository(db)
	eventRepo := repository.NewEventRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	// Create services
	authService := services.NewAuthService(userRepo, refreshTokenRepo, cfg)
	userService := services.NewUserService(userRepo)
	adminService := services.NewAdminService(adminRepo, refreshTokenRepo, cfg)
	categoryService := services.NewCategoryService(categoryRepo, eventRepo)
	storageService := services.NewStorageService(cfg)
	fileService := services.NewFileService(fileRepo, storageService)
	eventService := services.NewEventService(eventRepo, categoryRepo, fileRepo)
	paymentService := services.NewPaymentService(paymentRepo, orderRepo, eventRepo, cfg.MidtransServerKey, cfg.MidtransIsSandbox)
	orderService := services.NewOrderService(orderRepo, eventRepo, userRepo, paymentService)

	// Create the first admin from config on a fresh deploy. No-op once that
	// admin exists, or when the bootstrap env vars are unset.
	if err := adminService.EnsureBootstrapAdmin(
		context.Background(),
		cfg.AdminBootstrapEmail, cfg.AdminBootstrapName, cfg.AdminBootstrapPassword,
	); err != nil {
		log.Fatalf("Failed to bootstrap admin: %v", err)
	}

	// Create handlers
	authHandler := handlers.NewAuthHandler(authService, userService, cfg)
	userHandler := handlers.NewUserHandler(userService, orderService)
	adminHandler := handlers.NewAdminHandler(adminService, cfg)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	fileHandler := handlers.NewFileHandler(fileService)
	eventHandler := handlers.NewEventHandler(eventService, cfg.R2PublicBaseURL)
	orderHandler := handlers.NewOrderHandler(orderService, cfg.R2PublicBaseURL)
	paymentHandler := handlers.NewPaymentHandler(paymentService)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	router := gin.New()
	router.Use(gin.Recovery())

	// Trust only the configured proxies for X-Forwarded-For (empty => none),
	// so c.ClientIP() can't be spoofed from outside the platform edge.
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		log.Fatalf("Failed to set trusted proxies: %v", err)
	}

	// Liveness probe for the platform health check (Railway, k8s, ...).
	// Registered before the middleware chain so the platform's frequent polling
	// doesn't spam the request log.
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Middleware
	router.Use(middleware.CORSMiddleware(cfg.CORSOrigins))
	router.Use(middleware.LoggerMiddleware())
	router.Use(middleware.ErrorMiddleware())

	// Swagger docs - kept out of production, where the @host annotation points
	// at localhost and the spec would otherwise be served publicly.
	if cfg.Environment != "production" {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Routes
	api := router.Group("/api")
	{
		// Auth routes
		auth := api.Group("/auth")
		{
			auth.POST("/oauth/google/callback", authHandler.GoogleCallback)
			auth.POST("/refresh", authHandler.RefreshToken)

			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
			authProtected.Use(middleware.CSRFMiddleware(cfg.CORSOrigins))
			{
				authProtected.GET("/me", authHandler.GetCurrentUser)
				authProtected.POST("/logout", authHandler.Logout)
			}
		}

		// Event routes (public)
		events := api.Group("/events")
		{
			events.GET("", eventHandler.GetEvents)
			events.GET("/search", eventHandler.SearchEvents)
			events.GET("/:id", eventHandler.GetEventByID)

			// Event mutation routes (admin only)
			eventsProtected := events.Group("")
			eventsProtected.Use(middleware.AdminAuthMiddleware(cfg.JWTSecret))
			eventsProtected.Use(middleware.CSRFMiddleware(cfg.CORSOrigins))
			{
				eventsProtected.POST("", eventHandler.CreateEvent)
				eventsProtected.PUT("/:id", eventHandler.UpdateEvent)
				eventsProtected.DELETE("/:id", eventHandler.DeleteEvent)
			}
		}

		// File routes (admin only for now - only event covers use these today)
		files := api.Group("/files")
		files.Use(middleware.AdminAuthMiddleware(cfg.JWTSecret))
		files.Use(middleware.CSRFMiddleware(cfg.CORSOrigins))
		{
			files.POST("/presign-upload", fileHandler.PresignUpload)
		}

		// Category routes (public read, admin-only write)
		categories := api.Group("/categories")
		{
			categories.GET("", categoryHandler.ListCategories)
			categories.GET("/:id", categoryHandler.GetCategoryByID)

			categoriesProtected := categories.Group("")
			categoriesProtected.Use(middleware.AdminAuthMiddleware(cfg.JWTSecret))
			categoriesProtected.Use(middleware.CSRFMiddleware(cfg.CORSOrigins))
			{
				categoriesProtected.POST("", categoryHandler.CreateCategory)
				categoriesProtected.PUT("/:id", categoryHandler.UpdateCategory)
				categoriesProtected.DELETE("/:id", categoryHandler.DeleteCategory)
			}
		}

		// Order routes (protected)
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

		// Payment routes
		payments := api.Group("/payments")
		{
			payments.POST("/webhook", paymentHandler.WebhookHandler)

			paymentsProtected := payments.Group("")
			paymentsProtected.Use(middleware.AuthMiddleware(cfg.JWTSecret))
			paymentsProtected.Use(middleware.CSRFMiddleware(cfg.CORSOrigins))
			{
				paymentsProtected.GET("/status/:transactionId", paymentHandler.GetPaymentStatus)
			}
		}

		// User routes (protected)
		user := api.Group("/user")
		user.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		user.Use(middleware.CSRFMiddleware(cfg.CORSOrigins))
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.GET("/stats", userHandler.GetStats)
		}

		// Admin routes
		admin := api.Group("/admin")
		{
			admin.POST("/auth/login", adminHandler.Login)
			admin.POST("/auth/refresh", adminHandler.RefreshToken)

			adminProtected := admin.Group("")
			adminProtected.Use(middleware.AdminAuthMiddleware(cfg.JWTSecret))
			adminProtected.Use(middleware.CSRFMiddleware(cfg.CORSOrigins))
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
	}

	// Start the server, then block until an interrupt/termination signal.
	// Railway sends SIGTERM on every redeploy, so draining in-flight requests
	// (e.g. a payment webhook) before exiting avoids dropping them.
	srv := &http.Server{Addr: ":" + cfg.Port, Handler: router}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Starting server on port %s (%s)", cfg.Port, cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Println("Shutdown signal received, draining connections...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shut down: %v", err)
	}

	log.Println("Server stopped cleanly")
}
