package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"math/rand"
	"net/http"
	"shorturl/internal/config"
	"strings"
	"time"
)

type URLStore struct {
	urls map[string]string
}

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func NewRandomString(size int) string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, size)
	for i := range result {
		result[i] = chars[r.Intn(len(chars))]
	}
	return string(result)
}

func handlePost(store *URLStore, cfg *config.Config) http.HandlerFunc {
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

		shortID := NewRandomString(8)
		store.urls[shortID] = originalURL

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, "%s/%s", cfg.BaseURL, shortID)
	}
}

func handleGet(store *URLStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		shortID := chi.URLParam(r, "shortID")
		if len(shortID) != 8 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		originalURL, ok := store.urls[shortID]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func main() {
	cfg := config.Load()
	store := &URLStore{urls: make(map[string]string)}

	router := chi.NewRouter()
	router.Post("/", handlePost(store, cfg))
	router.Get("/{shortID}", handleGet(store))

	http.ListenAndServe(cfg.ServerAddress, router)
}
