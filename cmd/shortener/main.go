package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"log"
	"math/rand"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/logger"
	"shorturl/internal/middleware"
	"shorturl/internal/service"
	"shorturl/internal/storage"
	"time"
)

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
		return
	}
	defer func() {
		if err := zapLogger.Sync(); err != nil {
			log.Printf("failed to sync zap logger: %v", err)
		}
	}()
	cfg := config.Load()

	var urlStorage storage.URLStorage
	if cfg.FileStoragePath != "" {
		urlStorage = storage.NewFileStorage(cfg.FileStoragePath)
		log.Printf("Using file storage at: %s", cfg.FileStoragePath)
	} else {
		urlStorage = storage.NewInMemoryStorage()
		log.Println("Using in-memory storage")
	}

	svc := service.NewURLService(urlStorage)
	h := handlers.NewHandlers(svc)
	r := chi.NewRouter()

	r.Use(logger.Middleware(zapLogger))
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))
	r.Use(middleware.GzipResponse)

	r.Route("/", func(r chi.Router) {
		r.Use(middleware.GzipRequest)
		r.Post("/", h.HandlePost(cfg))
		r.Post("/api/shorten", h.HandleAPIShorten(cfg))
	})
	r.Get("/{shortID}", h.HandleGet())

	fmt.Printf("Server address from config: %s\n", cfg.ServerAddress) // ПЕРЕНЕСЛИ СЮДА
	fmt.Printf("Starting server on %s\n", cfg.BaseURL)                // ПЕРЕНЕСЛИ СЮДА
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
