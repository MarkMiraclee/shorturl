package app

import (
	"go.uber.org/zap"
	"io"
	"math/rand"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/logger"
	"shorturl/internal/router"
	"shorturl/internal/service"
	"shorturl/internal/storage"
	"time"
)

type App struct {
	Router http.Handler
	Closer io.Closer
}

func New(cfg *config.Config) (*App, error) {
	rand.New(rand.NewSource(time.Now().UnixNano()))

	logger.Logger.Info("Loaded configuration", zap.String("config", cfg.String()))
	logger.Logger.Info("Config values:",
		zap.String("ServerAddress", cfg.ServerAddress),
		zap.String("BaseURL", cfg.BaseURL),
		zap.String("FileStoragePath", cfg.FileStoragePath),
		zap.String("LogLevel", cfg.LogLevel),
		zap.String("LogFormat", cfg.LogFormat),
		zap.String("DatabaseDSN", cfg.DatabaseDSN),
	)

	var store service.ShortURLCreatorGetter
	var pinger service.Pinger
	var closer io.Closer

	if cfg.DatabaseDSN != "" {
		dbStorage, err := storage.NewDatabaseStorage(cfg.DatabaseDSN)
		if err == nil {
			store = dbStorage
			pinger = dbStorage
			closer = dbStorage
			logger.Logger.Info("Using PostgreSQL database storage")
		} else {
			logger.Logger.Error("Failed to initialize database storage, falling back to file or memory", zap.Error(err))
		}
	}

	if store == nil && cfg.FileStoragePath != "" {
		fileStorage, err := storage.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			return nil, err
		}
		store = fileStorage
		logger.Logger.Info("Using file storage")
	}

	if store == nil {
		memStorage := storage.NewInMemoryStorage()
		store = memStorage
		logger.Logger.Info("Using only in-memory storage")
	}

	svc := service.NewURLService(store, pinger)
	h := handlers.NewHandlers(svc)
	r := router.New(h, cfg)

	return &App{Router: r, Closer: closer}, nil
}
