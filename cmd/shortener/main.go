package main

import (
	"go.uber.org/zap"
	"net/http"
	"shorturl/internal/app"
	"shorturl/internal/config"
	"shorturl/internal/logger"
)

func main() {
	cfg := config.Load()

	logger.InitializeLogger(cfg)

	application, err := app.New(cfg)
	if err != nil {
		logger.Logger.Fatal("failed to create app", zap.Error(err))
	}
	if application.Closer != nil {
		defer func() {
			if err := application.Closer.Close(); err != nil {
				logger.Logger.Error("failed to close resources", zap.Error(err))
			}
		}()
	}

	logger.Logger.Info("Starting server", zap.String("address", cfg.ServerAddress), zap.String("baseURL", cfg.BaseURL))
	if err := http.ListenAndServe(cfg.ServerAddress, application.Router); err != nil {
		logger.Logger.Fatal("Failed to start server", zap.Error(err))
	}
}
