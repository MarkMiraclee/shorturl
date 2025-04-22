package main

import (
	"fmt"
	"log"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/logger"
	"shorturl/internal/middleware"
	"shorturl/internal/service"
	"shorturl/internal/storage"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
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
	urlStorage := storage.NewInMemoryStorage()
	svc := service.NewURLService(urlStorage)
	h := handlers.NewHandlers(svc)
	r := chi.NewRouter()

	r.Use(logger.Middleware(zapLogger))
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))
	r.Use(middleware.GzipResponse) // Применяем middleware для сжатия ответов

	r.Route("/", func(r chi.Router) {
		r.Use(middleware.GzipRequest) // Применяем middleware для распаковки запросов
		r.Post("/", h.HandlePost(cfg))
		r.Post("/api/shorten", h.HandleAPIShorten(cfg))
	})
	r.Get("/{shortID}", h.HandleGet())

	fmt.Printf("Server address from config: %s\n", cfg.ServerAddress)
	fmt.Printf("Starting server on %s\n", cfg.BaseURL)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
