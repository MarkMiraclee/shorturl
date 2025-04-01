package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
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

func main() {
	store := &URLStore{
		urls: make(map[string]string),
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
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

		case http.MethodGet:
			shortID := strings.TrimPrefix(r.URL.Path, "/")
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

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}
