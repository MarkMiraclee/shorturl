package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"
	"github.com/go-chi/chi/v5"
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

func handlePost(store *URLStore) http.HandlerFunc {
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
		fmt.Fprintf(w, "http://localhost:8080/%s", shortID)
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
			w.WriteHeader(http.StatusBadRequest) // Оставляем как было
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}
}

func main() {
	store := &URLStore{
		urls: make(map[string]string),
	}

	router := chi.NewRouter()

	// Роутинг с Chi
	router.Post("/", handlePost(store))
	router.Get("/{shortID}", handleGet(store))

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", router)
}
