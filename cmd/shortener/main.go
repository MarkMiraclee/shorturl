package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
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

	cfg := config.Load()

	logger.InitializeLogger(cfg)
	logger.Logger.Info("Loaded configuration", zap.String("config", cfg.String()))
	logger.Logger.Info("Config values:",
		zap.String("ServerAddress", cfg.ServerAddress),
		zap.String("BaseURL", cfg.BaseURL),
		zap.String("FileStoragePath", cfg.FileStoragePath),
		zap.String("LogLevel", cfg.LogLevel),
		zap.String("LogFormat", cfg.LogFormat),
		zap.String("DatabaseDSN", cfg.DatabaseDSN),
	)

	var store service.ShortURLCreatorGetter // Интерфейс для хранилища

	if cfg.DatabaseDSN != "" {
		dbStorage, err := storage.NewDatabaseStorage(cfg.DatabaseDSN)
		if err == nil {
			store = dbStorage
			defer func() {
				if err := dbStorage.Close(); err != nil {
					logger.Logger.Error("Error closing database connection", zap.Error(err))
				}
			}()
			logger.Logger.Info("Using PostgreSQL database storage")
		} else {
			logger.Logger.Error("Failed to initialize database storage, falling back to file or memory", zap.Error(err))
		}
	}

	if store == nil && cfg.FileStoragePath != "" {
		fileStorage, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			logger.Logger.Fatal("failed to initialize file storage", zap.Error(err))
		}
		store = fileStorage
		logger.Logger.Info("Using file storage")
	}

	if store == nil {
		memStorage := storage.NewInMemoryStorage()
		store = memStorage // По умолчанию используем in-memory
		logger.Logger.Info("Using only in-memory storage")
	}

	svc := service.NewURLService(store) // Инициализируем сервис с выбранным хранилищем
	h := handlers.NewHandlers(svc)
	r := chi.NewRouter()

	r.Use(logger.Middleware(logger.Logger))
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(60 * time.Second))
	r.Use(middleware.GzipResponse)
	r.Use(middleware.Auth)

	r.Group(func(r chi.Router) {
		r.Use(middleware.GzipRequest)
		r.Post("/", h.HandlePost(cfg))
		r.Post("/api/shorten", h.HandleAPIShorten(cfg))
		r.Post("/api/shorten/batch", h.HandleAPIShortenBatch(cfg))
	})
	r.Get("/api/user/urls", h.HandleGetUserURLs(cfg))
	r.Get("/{shortID}", h.HandleGet())

	// Добавляем новый хендлер /ping
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()
		if dbStorage, ok := store.(*storage.DatabaseStorage); ok {
			if err := dbStorage.PingContext(ctx); err != nil {
				logger.Logger.Error("Database ping failed", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		// Если используется in-memory или file storage, возвращаем 200 OK
		w.WriteHeader(http.StatusOK)
	})

	logger.Logger.Info("Starting server", zap.String("address", cfg.ServerAddress), zap.String("baseURL", cfg.BaseURL))
	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		logger.Logger.Fatal("Failed to start server", zap.Error(err))
	}
	time.Sleep(time.Millisecond * 100)
}
