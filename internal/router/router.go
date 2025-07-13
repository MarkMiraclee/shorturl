package router

import (
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/logger"
	"shorturl/internal/middleware"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func New(h *handlers.Handlers, cfg *config.Config) http.Handler {
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
		r.Delete("/api/user/urls", h.HandleDeleteUserURLs())
	})
	r.Get("/api/user/urls", h.HandleGetUserURLs(cfg))
	r.Get("/{shortID}", h.HandleGet())
	r.Get("/ping", h.HandlePing())

	return r
}
