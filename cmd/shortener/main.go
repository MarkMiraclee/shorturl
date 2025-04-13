package main

import (
	"fmt"
	"log"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/logger"
	"shorturl/internal/service"
	"shorturl/internal/storage"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

func main() {
	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
		return
	}
	defer zapLogger.Sync()
	cfg := config.Load()
	urlStorage := storage.NewInMemoryStorage()
	svc := service.NewURLService(urlStorage)
	h := handlers.NewHandlers(svc)
	r := chi.NewRouter()

	r.Use(logger.Middleware(zapLogger))

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(middleware.Timeout(60 * time.Second))

	r.Post("/", h.HandlePost(cfg))
	r.Get("/{shortID}", h.HandleGet())

	fmt.Printf("Server address from config: %s\n", cfg.ServerAddress)
	fmt.Printf("Starting server on %s\n", cfg.BaseURL)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
