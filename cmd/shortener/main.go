package main

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"math/rand"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/logger"
	"shorturl/internal/middleware"
	"shorturl/internal/service"
	"shorturl/internal/storage"
	"time"

	"go.uber.org/zap"
)

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	cfg := config.Load()

	logger.InitializeLogger(cfg)

	defer func() {
		if logger.Logger != nil {
			logger.Logger.Sync()
		}
	}()

	logger.Logger.Info("Loaded configuration", zap.String("config", cfg.String())) // Выводим конфигурацию одной строкой

	var urlStorage service.ShortURLCreatorGetter
	if cfg.FileStoragePath != "" {
		fileStorage := storage.NewFileStorage(cfg.FileStoragePath)
		urlStorage = fileStorage // FileStorage теперь неявно реализует ShortURLCreatorGetter
		logger.Logger.Info("Using file storage", zap.String("path", cfg.FileStoragePath))
	} else {
		urlStorage = storage.NewInMemoryStorage() // InMemoryStorage реализует ShortURLCreatorGetter
		logger.Logger.Info("Using in-memory storage")
	}

	svc := service.NewURLService(urlStorage)
	h := handlers.NewHandlers(svc)
	r := chi.NewRouter()

	r.Use(logger.Middleware(logger.Logger))
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

	logger.Logger.Info("Starting server", zap.String("address", cfg.ServerAddress), zap.String("baseURL", cfg.BaseURL))
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		logger.Logger.Fatal("Failed to start server", zap.Error(err))
	}
}
