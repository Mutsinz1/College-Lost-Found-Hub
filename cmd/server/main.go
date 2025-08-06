package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001", "http://localhost:65063"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Building routes
		r.Route("/buildings", func(r chi.Router) {
			r.Get("/", handlers.GetBuildings)
			r.Get("/{id}", handlers.GetBuildingByID)
			r.Post("/", handlers.CreateBuilding) // Admin only
		})

		// Lost & Found Area routes
		r.Route("/areas", func(r chi.Router) {
			r.Get("/", handlers.GetLostFoundAreas)
			r.Get("/building/{buildingId}", handlers.GetLostFoundAreasByBuilding)
			r.Post("/", handlers.CreateLostFoundArea) // Admin only
		})

		// User routes
		r.Route("/users", func(r chi.Router) {
			r.Post("/sso", handlers.GetOrCreateUser)
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