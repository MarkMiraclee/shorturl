package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/logger"
	"shorturl/internal/service"
	"strings"

	"go.uber.org/zap"
)

const shortURLLength = 8

// Handlers представляет собой структуру с обработчиками HTTP-запросов.
type Handlers struct {
	Service service.URLShortener
}

// NewHandlers создает и возвращает новый экземпляр Handlers с заданным сервисом.
func NewHandlers(svc service.URLShortener) *Handlers {
	return &Handlers{Service: svc}
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

// HandleAPIShorten обрабатывает POST-запросы к /api/shorten для сокращения URL (JSON).
func (h *Handlers) HandleAPIShorten(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ShortenRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer func() {
			if errClose := r.Body.Close(); errClose != nil {
				logger.Logger.Error("Error closing request body", zap.Error(errClose))
			}
		}()

		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		originalURL := req.URL
		if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}

		shortID, err := h.Service.CreateShortURL(originalURL) // Используем метод интерфейса
		if err != nil {
			http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
			return
		}

		response := ShortenResponse{
			Result: fmt.Sprintf("%s/%s", cfg.BaseURL, shortID),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Logger.Error("Error writing JSON response", zap.Error(err))
		}
	}
}

// HandlePost обрабатывает POST-запросы (текст).
func (h *Handlers) HandlePost(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer func() {
			if errClose := r.Body.Close(); errClose != nil {
				logger.Logger.Error("Error closing request body", zap.Error(errClose))
			}
		}()

		originalURL := string(body)
		if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		shortID, err := h.Service.CreateShortURL(originalURL) // Используем метод интерфейса
		if err != nil {
			http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		_, err = fmt.Fprintf(w, "%s/%s", cfg.BaseURL, shortID)
		if err != nil {
			logger.Logger.Error("Error writing response", zap.Error(err))
		}
	}
}

// HandleGet обрабатывает GET-запросы с параметром shortID
func (h *Handlers) HandleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortID := chi.URLParam(r, "shortID")
		if len(shortID) != shortURLLength {
			http.Error(w, fmt.Sprintf("Invalid short URL format (expected %d characters)", shortURLLength), http.StatusBadRequest)
			return
		}
		defer func() {
			if errClose := r.Body.Close(); errClose != nil {
				logger.Logger.Error("Error closing request body", zap.Error(errClose))
			}
		}()
		originalURL, err := h.Service.GetOriginalURL(shortID) // Используем метод интерфейса
		if err != nil {
			http.Error(w, "Invalid or non-existent short URL", http.StatusBadRequest)
			return
		}
		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}
