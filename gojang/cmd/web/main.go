package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gojangframework/gojang/gojang/utils"

	"github.com/gojangframework/gojang/gojang/admin"
	"github.com/gojangframework/gojang/gojang/config"
	"github.com/gojangframework/gojang/gojang/http/handlers"
	"github.com/gojangframework/gojang/gojang/http/middleware"
	"github.com/gojangframework/gojang/gojang/http/routes"
	"github.com/gojangframework/gojang/gojang/models/db"
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
	postHandler := handlers.NewPostHandler(client, publicRenderer)
	pageHandler := handlers.NewPageHandler(publicRenderer)

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

	// Static files (CSS and assets in views/static)
	fileServer := http.FileServer(http.Dir("./gojang/views/static"))
	r.Handle("/static/*", http.StripPrefix("/static", fileServer))

	// Admin static files (keep admin assets in admin folder)
	adminFileServer := http.FileServer(http.Dir("./gojang/admin/views"))
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
		auth.Use(nosurf.NewPure)
		auth.Get("/login", authHandler.LoginGET)
		auth.With(middleware.RateLimit(authLimiter)).Post("/login", authHandler.LoginPOST)
		auth.Get("/register", authHandler.RegisterGET)
		auth.With(middleware.RateLimit(authLimiter)).Post("/register", authHandler.RegisterPOST)
		auth.Post("/logout", authHandler.LogoutPOST)
	})

	// Mount routes (organized by resource)
	r.Mount("/", routes.PageRoutes(pageHandler, sessionManager, client))
	r.Mount("/posts", routes.PostRoutes(postHandler, sessionManager, client))
	r.Mount("/users", routes.UserRoutes(userHandler, sessionManager, client))
	r.Mount("/admin", admin.AdminRoutes(adminHandler, sessionManager, client))

	// 404 handler for unmatched routes
	r.NotFound(pageHandler.NotFound)

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		utils.Infof("🚀 Server starting on http://localhost%s", addr)
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
