package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"lostfound/internal/api"
	"lostfound/internal/config"
	"lostfound/internal/database"
	"lostfound/internal/image"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.NewConnection(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repository
	repo := database.NewRepository(db)

	// Initialize image processor
	imgProcessor := image.NewProcessor(cfg.Upload.Dir, cfg.Upload.MaxSize, cfg.Upload.AllowedTypes)

	// Initialize handlers
	handlers := api.NewHandler(repo, imgProcessor)
	authHandlers := api.NewAuthHandler(repo, cfg)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration (origins come from ALLOWED_ORIGINS env var)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.Server.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Edit-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Parse session tokens (when present) for all API routes
		r.Use(api.Middleware(cfg.JWT.Secret))

		// Auth routes
		r.Route("/auth", func(r chi.Router) {
			r.Post("/google", authHandlers.GoogleLogin)
			if cfg.Server.Environment == "development" {
				// Dev-only login that replaces the old unauthenticated
				// /users/sso endpoint (which trusted client-sent identities)
				r.Post("/dev-login", authHandlers.DevLogin)
			}
		})

		// Building routes
		r.Route("/buildings", func(r chi.Router) {
			r.Get("/", handlers.GetBuildings)
			r.Get("/{id}", handlers.GetBuildingByID)
			r.With(api.RequireAdmin).Post("/", handlers.CreateBuilding)
		})

		// Lost & Found Area routes
		r.Route("/areas", func(r chi.Router) {
			r.Get("/", handlers.GetLostFoundAreas)
			r.Get("/building/{buildingId}", handlers.GetLostFoundAreasByBuilding)
			r.With(api.RequireAdmin).Post("/", handlers.CreateLostFoundArea)
		})

		// User routes
		r.Route("/users", func(r chi.Router) {
			r.Get("/{id}", handlers.GetUserByID)
		})

		// Post routes
		r.Route("/posts", func(r chi.Router) {
			r.Get("/", handlers.SearchPosts)
			r.Post("/", handlers.CreatePost)
			r.Get("/{id}", handlers.GetPostByID)
			r.Put("/{id}", handlers.UpdatePost)
			r.Delete("/{id}", handlers.DeletePost)
			r.Post("/{id}/claim", handlers.ClaimPost)
			r.Post("/{id}/interactions", handlers.CreateInteraction)
			r.Get("/{id}/interactions", handlers.GetPostInteractions)
			r.Post("/{id}/reports", handlers.CreateReport)
		})

		// Interaction routes
		r.Put("/interactions/{id}", handlers.UpdateInteraction)

		// Moderation. Reports name other people's posts and carry the
		// reporter's address, so reading and triaging them is admin-only;
		// filing one is not (see /posts/{id}/reports above).
		r.Route("/reports", func(r chi.Router) {
			r.With(api.RequireAdmin).Get("/", handlers.GetReports)
			r.With(api.RequireAdmin).Put("/{id}", handlers.UpdateReport)
		})

		// Saved-search alerts
		r.Route("/alerts", func(r chi.Router) {
			r.Post("/", handlers.CreateAlert)
			r.Get("/", handlers.GetAlerts)
			r.Delete("/{id}", handlers.DeleteAlert)
		})
	})

	// Serve static files (uploads)
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir(cfg.Upload.Dir))))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Periodically clean up expired posts and their image files
	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	go func() {
		runCleanup := func() {
			ctx, cancel := context.WithTimeout(cleanupCtx, 2*time.Minute)
			defer cancel()
			count, imageURLs, err := repo.CleanupExpiredPosts(ctx)
			if err != nil {
				log.Printf("expired-post cleanup failed: %v", err)
				return
			}
			for _, url := range imageURLs {
				filename := strings.TrimPrefix(strings.TrimPrefix(url, "/uploads/"), "uploads/")
				if filename == "" {
					continue
				}
				if err := imgProcessor.DeleteImage(filename); err != nil {
					log.Printf("failed to delete image %s: %v", filename, err)
				}
			}
			if count > 0 {
				log.Printf("cleaned up %d expired posts", count)
			}
		}

		runCleanup() // run once at startup
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-cleanupCtx.Done():
				return
			case <-ticker.C:
				runCleanup()
			}
		}
	}()

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on %s", ":"+cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
