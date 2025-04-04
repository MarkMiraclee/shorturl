package router

import (
	"github.com/go-chi/chi/v5"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
)

func NewRouter(store *handlers.URLStore, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", handlers.HandlePost(store, cfg))
	r.Get("/{shortID}", handlers.HandleGet(store))
	return r
}
