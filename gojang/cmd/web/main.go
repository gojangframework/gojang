package main

import (
	"context"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gojangframework/gojang/app/pages"
	"github.com/gojangframework/gojang/app/posts"
	"github.com/gojangframework/gojang/gojang/utils"

	"github.com/gojangframework/gojang/gojang/admin"
	"github.com/gojangframework/gojang/gojang/config"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/http/routes"
	"github.com/gojangframework/gojang/gojang/models/db"
	"github.com/gojangframework/gojang/gojang/views"
	"github.com/gojangframework/gojang/gojang/views/renderers"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/justinas/nosurf"
)

func main() {
	// Load config from .env
	cfg := config.MustLoad()

	// Initialize global logging
	// Use LOG_LEVEL env var or infer from cfg.Debug/ENV
	lvl := ""
	if cfg.Debug {
		lvl = "debug"
	}
	if err := utils.Init(lvl); err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	// Setup database
	client, err := db.NewClient(cfg.DatabaseURL)
	if err != nil {
		utils.Errorf("Failed to connect to database: %v", err)
		os.Exit(1)
	}
	defer client.Close()

	// Run auto-migrations
	ctx := context.Background()
	if err := db.AutoMigrate(ctx, client); err != nil {
		utils.Errorf("Failed to run migrations: %v", err)
		os.Exit(1)
	}

	// Setup session manager
	sessionManager := middleware.NewSessionManager(cfg)

	// Setup email service when SMTP is configured.
	var emailService *utils.EmailService
	if strings.TrimSpace(cfg.SMTPHost) != "" {
		emailService, err = utils.NewEmailService(utils.EmailConfig{
			SMTPHost:        cfg.SMTPHost,
			SMTPPort:        cfg.SMTPPort,
			SMTPUser:        cfg.SMTPUser,
			SMTPPass:        cfg.SMTPPass,
			FromAddress:     cfg.SMTPFrom,
			FromDisplayName: cfg.SMTPFromName,
			MaxSendRate:     cfg.EmailSendRate,
			QueueSize:       cfg.EmailQueueSize,
			WorkerCount:     cfg.EmailWorkerCount,
			SendTimeout:     cfg.EmailSendTimeout,
		})
		if err != nil {
			utils.Errorf("Failed to setup email service: %v", err)
			os.Exit(1)
		}
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := emailService.Shutdown(shutdownCtx); err != nil {
				utils.Warnw("email.shutdown_incomplete", "error", err)
			}
		}()
		utils.Infof("Email service configured")
	} else {
		utils.Warnf("Email service disabled: SMTP_HOST is not configured")
	}

	// Setup renderers
	// Public renderer: Handles public site pages with base.html wrapper
	publicRenderer, err := renderers.NewRenderer(cfg.Debug)
	if err != nil {
		utils.Errorf("Failed to setup public renderer: %v", err)
		os.Exit(1)
	}

	// Admin renderer: Handles admin panel (always fragments, no base.html)
	adminRenderer, err := admin.NewAdminRenderer(cfg.Debug)
	if err != nil {
		utils.Errorf("Failed to setup admin renderer: %v", err)
		os.Exit(1)
	}

	// Setup handlers
	authHandler := handlers.NewAuthHandler(client, sessionManager, publicRenderer)
	userHandler := handlers.NewUserHandler(client, publicRenderer)
	postHandler := posts.NewPostHandler(client, publicRenderer)
	pageHandler := pages.NewPageHandler(publicRenderer)

	// Setup admin registry and handler
	adminRegistry := admin.NewRegistry(client)
	// Register models with the admin system
	admin.RegisterModels(adminRegistry)
	adminHandler := admin.NewHandler(adminRegistry, adminRenderer, client)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.EnforceHTTPS(cfg))
	r.Use(middleware.SecurityHeaders(cfg))
	r.Use(sessionManager.LoadAndSave)
	r.Use(middleware.LoadUser(sessionManager, client)) // Load user from session on all pages
	r.Use(nosurf.NewPure)                              // CSRF protection on all routes

	// Static files (CSS and assets in views/static)
	staticFS, err := fs.Sub(views.StaticFiles, "static")
	if err != nil {
		utils.Errorf("Failed to create static sub-filesystem: %v", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// Admin static files (keep admin assets in admin folder)
	adminFS, err := fs.Sub(admin.ViewFiles, "views")
	if err != nil {
		utils.Errorf("Failed to create admin static sub-filesystem: %v", err)
		os.Exit(1)
	}
	adminFileServer := http.FileServer(http.FS(adminFS))
	r.Handle("/admin/static/*", http.StripPrefix("/admin/static", adminFileServer))

	// Well-known files (security.txt, etc.)
	wellKnownServer := http.FileServer(http.Dir("."))
	r.Handle("/.well-known/*", http.StripPrefix("/", wellKnownServer))

	// Auth routes (must be mounted before "/" to avoid conflicts)
	authLimiter := middleware.AuthRateLimiter()

	// Start cleanup routine for rate limiter (cleanup every 5 minutes)
	cleanupDone := make(chan struct{})
	defer close(cleanupDone)
	go authLimiter.StartCleanupRoutine(5*time.Minute, cleanupDone)

	r.Group(func(auth chi.Router) {
		auth.Get("/login", authHandler.LoginGET)
		auth.With(middleware.RateLimit(authLimiter)).Post("/login", authHandler.LoginPOST)
		auth.Get("/register", authHandler.RegisterGET)
		auth.With(middleware.RateLimit(authLimiter)).Post("/register", authHandler.RegisterPOST)
		auth.Post("/logout", authHandler.LogoutPOST)
	})

	// Mount routes (organized by resource)
	r.Mount("/", pages.PageRoutes(pageHandler, sessionManager, client))
	r.Mount("/posts", posts.PostRoutes(postHandler, sessionManager, client))
	r.Mount("/users", routes.UserRoutes(userHandler, sessionManager, client))
	r.Mount("/admin", admin.AdminRoutes(adminHandler, sessionManager, client))

	// 404 handler for unmatched routes
	r.NotFound(pageHandler.NotFound)

	// Start server
	addr := net.JoinHostPort(cfg.DevHost, cfg.Port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		utils.Infof("🚀 Server starting on http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Errorf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	utils.Infof("🛑 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		utils.Errorf("Server forced to shutdown: %v", err)
		os.Exit(1)
	}

	utils.Infof("✅ Server stopped")
}
