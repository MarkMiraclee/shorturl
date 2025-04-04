package handlers

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/generator"
	"strings"
)

type URLStore struct {
	URLs map[string]string
}

func HandlePost(store *URLStore, cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		originalURL := string(body)
		if !strings.HasPrefix(originalURL, "http") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		shortID := generator.NewRandomString(8)
		store.URLs[shortID] = originalURL

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "%s/%s", cfg.BaseURL, shortID)
	}
}

func HandleGet(store *URLStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortID := chi.URLParam(r, "shortID")
		if len(shortID) != 8 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		originalURL, ok := store.URLs[shortID]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}
