package handlers

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"log"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/service"
	"strings"
)

type Handlers struct {
	Service service.URLShortener
}

// NewHandlers создает и возвращает новый экземпляр Handlers с заданным сервисом.
func NewHandlers(svc service.URLShortener) *Handlers {
	return &Handlers{Service: svc}
}

// HandlePost обрабатывает POST-запросы
func (h *Handlers) HandlePost(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		originalURL := string(body)

		if !strings.HasPrefix(originalURL, "http") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		shortID, err := h.Service.CreateShortURL(originalURL)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		_, err = fmt.Fprintf(w, "%s/%s", cfg.BaseURL, shortID)
		if err != nil {
			log.Printf("Error writing response: %v", err)
		}
	}
}

// HandleGet обрабатывает GET-запросы
func (h *Handlers) HandleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortID := chi.URLParam(r, "shortID")
		originalURL, err := h.Service.GetOriginalURL(shortID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}
