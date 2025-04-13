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

// Handlers представляет собой структуру с обработчиками HTTP-запросов.
type Handlers struct {
	Service service.URLShortener // Интерфейс сервиса для работы с URL.
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
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("Error closing request body: %v", err)
			}
		}()

		originalURL := string(body)
		if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		shortID, err := h.Service.CreateShortURL(originalURL)
		if err != nil {
			http.Error(w, "Failed to create short URL", http.StatusBadRequest)
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

// HandleGet обрабатывает GET-запросы с параметром shortID
func (h *Handlers) HandleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortID := chi.URLParam(r, "shortID")
		if len(shortID) != 8 {
			http.Error(w, "Invalid short URL format", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				log.Printf("Error closing request body: %v", err)
			}
		}()
		originalURL, err := h.Service.GetOriginalURL(shortID)
		if err != nil {
			http.Error(w, "Invalid or non-existent short URL", http.StatusBadRequest)
			return
		}
		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}
