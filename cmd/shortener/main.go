package main

import (
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"math/rand"
	"net/http"
	"os"
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
	// Инициализируем InMemoryStorage как основное хранилище
	memStorage := storage.NewInMemoryStorage()
	var persistentStorage *storage.FileStorage
	fileStoragePath := cfg.FileStoragePath // Получаем значение из конфигурации как значение по умолчанию

	if envPath := os.Getenv("FILE_STORAGE_PATH"); envPath != "" {
		fileStoragePath = envPath // Перезаписываем, если переменная окружения установлена
	}
	if fileStoragePath != "" {
		persistentStorage = storage.NewFileStorage(fileStoragePath)
		logger.Logger.Info("Using FileStorage path:", zap.String("path", fileStoragePath))

		successful, failed, err := persistentStorage.LoadAllToMemory(memStorage)
		if err != nil {
			logger.Logger.Error("Error loading data from file to memory", zap.Error(err), zap.Int("successful", successful), zap.Int("failed", failed))
		} else if failed > 0 {
			logger.Logger.Warn("Loaded data from file with some errors", zap.Int("successful", successful), zap.Int("failed", failed))
		} else {
			logger.Logger.Info("Data loaded from file to in-memory storage")
		}
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				if err := persistentStorage.SaveAllFromMemory(memStorage); err != nil {
					logger.Logger.Error("Error saving data from memory to file", zap.Error(err))
				} else {
					logger.Logger.Info("Data saved from memory to file")
				}
			}
		}()
		defer func() {
			if err := persistentStorage.SaveAllFromMemory(memStorage); err != nil {
				logger.Logger.Error("Error saving data from memory to file on exit", zap.Error(err))
			} else {
				logger.Logger.Info("Data saved from memory to file on exit")
			}
		}()
	} else {
		logger.Logger.Info("Using only in-memory storage")
	}
	svc := service.NewURLService(memStorage)
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
	time.Sleep(time.Millisecond * 100)
}
