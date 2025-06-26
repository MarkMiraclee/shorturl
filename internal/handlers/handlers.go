package handlers

import (
	"encoding/json"
	"errors"
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

type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
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

		shortID, err := h.Service.CreateShortURL(r.Context(), originalURL) // Используем метод интерфейса
		var conflictErr *service.ErrConflict                               // Объявляем conflictErr здесь

		if err != nil {
			if errors.As(err, &conflictErr) {
				w.Header().Set("Content-Type", "application/json") // Устанавливаем заголовок
				w.WriteHeader(http.StatusConflict)
				response := ShortenResponse{
					Result: fmt.Sprintf("%s/%s", cfg.BaseURL, shortID),
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					logger.Logger.Error("Error writing JSON response", zap.Error(err))
				}
				return
			} else {
				http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
				return
			}
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

// HandleAPIShortenBatch обрабатывает POST-запросы к /api/shorten/batch для пакетного сокращения URL (JSON).
func (h *Handlers) HandleAPIShortenBatch(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var requests []BatchShortenRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request body", http.StatusBadRequest)
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				logger.Logger.Error("error closing request body", zap.Error(err))
			}
		}()

		if err := json.Unmarshal(body, &requests); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		responses := make([]BatchShortenResponse, len(requests))
		for i, req := range requests {
			if !strings.HasPrefix(req.OriginalURL, "http://") && !strings.HasPrefix(req.OriginalURL, "https://") {
				http.Error(w, fmt.Sprintf("Invalid URL format for correlation_id: %s", req.CorrelationID), http.StatusBadRequest)
				return // Прерываем обработку всего пакета, если хотя бы один URL невалиден
			}

			shortID, err := h.Service.CreateShortURL(r.Context(), req.OriginalURL)
			if err != nil {
				logger.Logger.Error("Failed to create short URL for batch", zap.Error(err), zap.String("correlation_id", req.CorrelationID), zap.String("original_url", req.OriginalURL))
				http.Error(w, "Failed to create short URL for batch", http.StatusInternalServerError)
				return // Прерываем обработку всего пакета при ошибке создания.
			}

			responses[i] = BatchShortenResponse{
				CorrelationID: req.CorrelationID,
				ShortURL:      fmt.Sprintf("%s/%s", cfg.BaseURL, shortID),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(responses); err != nil {
			logger.Logger.Error("Error writing JSON response for batch", zap.Error(err))
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
		shortID, err := h.Service.CreateShortURL(r.Context(), originalURL) // Используем метод интерфейса
		if err != nil {
			var conflictErr *service.ErrConflict
			if errors.As(err, &conflictErr) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusConflict)
				_, err = fmt.Fprintf(w, "%s/%s", cfg.BaseURL, shortID)
				if err != nil {
					logger.Logger.Error("Error writing conflict response", zap.Error(err))
				}
				return
			} else {
				http.Error(w, "Failed to create short URL", http.StatusInternalServerError)
				return
			}
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
		originalURL, err := h.Service.GetOriginalURL(r.Context(), shortID) // Используем метод интерфейса
		if err != nil {
			http.Error(w, "Invalid or non-existent short URL", http.StatusBadRequest)
			return
		}
		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}
