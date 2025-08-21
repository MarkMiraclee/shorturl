package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/logger"
	"shorturl/internal/middleware"
	"shorturl/internal/service"
	"shorturl/internal/storage"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"go.uber.org/zap"
)

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

type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

func (h *Handlers) HandleAPIShorten(cfg *config.Config) http.HandlerFunc {
	return h.handleAPIShorten(cfg)
}

func (h *Handlers) handleAPIShorten(cfg *config.Config) http.HandlerFunc {
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

		userID, ok := r.Context().Value(middleware.UserIDKey).(string)
		if !ok {
			logger.Logger.Error("userID not found in context")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		shortID, err := h.Service.CreateShortURL(r.Context(), userID, originalURL)
		var conflictErr *service.ErrConflict

		if err != nil {
			if errors.As(err, &conflictErr) {
				w.Header().Set("Content-Type", "application/json")
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
	return h.handleAPIShortenBatch(cfg)
}

func (h *Handlers) handleAPIShortenBatch(cfg *config.Config) http.HandlerFunc {
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
				return
			}

			userID, ok := r.Context().Value(middleware.UserIDKey).(string)
			if !ok {
				logger.Logger.Error("userID not found in context")
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			shortID, err := h.Service.CreateShortURL(r.Context(), userID, req.OriginalURL)
			if err != nil {
				logger.Logger.Error("Failed to create short URL for batch", zap.Error(err), zap.String("correlation_id", req.CorrelationID), zap.String("original_url", req.OriginalURL))
				http.Error(w, "Failed to create short URL for batch", http.StatusInternalServerError)
				return
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
	return h.handlePost(cfg)
}

func (h *Handlers) handlePost(cfg *config.Config) http.HandlerFunc {
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

		userID, ok := r.Context().Value(middleware.UserIDKey).(string)
		if !ok {
			logger.Logger.Error("userID not found in context")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		shortID, err := h.Service.CreateShortURL(r.Context(), userID, originalURL)
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
	return h.handleGet()
}

func (h *Handlers) handleGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortID := chi.URLParam(r, "shortID")
		if shortID == "" {
			http.Error(w, "Short ID is required", http.StatusBadRequest)
			return
		}
		defer func() {
			if errClose := r.Body.Close(); errClose != nil {
				logger.Logger.Error("Error closing request body", zap.Error(errClose))
			}
		}()
		originalURL, err := h.Service.GetOriginalURL(r.Context(), shortID)
		if err != nil {
			if errors.Is(err, storage.ErrURLDeleted) {
				w.WriteHeader(http.StatusGone)
				return
			}
			http.Error(w, "Invalid or non-existent short URL", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
	}
}

func (h *Handlers) HandleDeleteUserURLs() http.HandlerFunc {
	return h.handleDeleteUserURLs()
}

func (h *Handlers) handleDeleteUserURLs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(middleware.UserIDKey).(string)
		if !ok || userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var shortIDs []string
		if err := json.NewDecoder(r.Body).Decode(&shortIDs); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if len(shortIDs) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}

		if err := h.Service.DeleteURLs(r.Context(), userID, shortIDs); err != nil {
			http.Error(w, "Failed to schedule URL deletion", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func (h *Handlers) HandleGetUserURLs(cfg *config.Config) http.HandlerFunc {
	return h.handleGetUserURLs(cfg)
}

func (h *Handlers) handleGetUserURLs(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value(middleware.UserIDKey).(string)
		if !ok || userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		userURLs, err := h.Service.GetURLsByUserID(r.Context(), userID)
		if err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if len(userURLs) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		response := make([]UserURLResponse, len(userURLs))
		for i, urlPair := range userURLs {
			response[i] = UserURLResponse{
				ShortURL:    fmt.Sprintf("%s/%s", cfg.BaseURL, urlPair.ShortURL),
				OriginalURL: urlPair.OriginalURL,
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Logger.Error("Error writing JSON response for user URLs", zap.Error(err))
		}
	}
}

func (h *Handlers) HandlePing() http.HandlerFunc {
	return h.handlePing()
}

func (h *Handlers) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
		defer cancel()

		if err := h.Service.Ping(ctx); err != nil {
			logger.Logger.Error("Database ping failed", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}
