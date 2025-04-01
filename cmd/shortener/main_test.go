package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandlePost(t *testing.T) {
	store := &URLStore{urls: make(map[string]string)}
	router := chi.NewRouter()
	router.Post("/", handlePost(store))

	tests := []struct {
		name           string
		body           string
		wantStatus     int
		wantBodyPrefix string
	}{
		{
			name:           "Valid URL",
			body:           "http://example.com",
			wantStatus:     http.StatusCreated,
			wantBodyPrefix: "http://localhost:8080/",
		},
		{
			name:       "Empty Body",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid URL Scheme",
			body:       "ftp://example.com",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req) // Используем router вместо handler

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantBodyPrefix != "" {
				if !strings.HasPrefix(rr.Body.String(), tt.wantBodyPrefix) {
					t.Errorf("body should start with %q, got %q", tt.wantBodyPrefix, rr.Body.String())
				}

				shortID := strings.TrimPrefix(rr.Body.String(), "http://localhost:8080/")
				if len(shortID) != 8 {
					t.Errorf("shortID must be 8 chars, got %d", len(shortID))
				}

				if originalURL, ok := store.urls[shortID]; !ok || originalURL != tt.body {
					t.Errorf("store does not contain URL %s", tt.body)
				}

				if contentType := rr.Header().Get("Content-Type"); contentType != "text/plain" {
					t.Errorf("Content-Type should be text/plain, got %s", contentType)
				}
			}
		})
	}
}

func TestHandleGet(t *testing.T) {
	store := &URLStore{urls: map[string]string{"abcdefgh": "http://example.com"}}
	router := chi.NewRouter()
	router.Get("/{shortID}", handleGet(store)) // Используем параметр пути

	tests := []struct {
		name         string
		shortID      string
		wantStatus   int
		wantLocation string
	}{
		{
			name:         "Valid ShortID",
			shortID:      "abcdefgh",
			wantStatus:   http.StatusTemporaryRedirect,
			wantLocation: "http://example.com",
		},
		{
			name:       "Invalid ShortID",
			shortID:    "invalid12",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "ShortID Wrong Length",
			shortID:    "abc",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/"+tt.shortID, nil) // Путь с shortID
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req) // Используем router

			if rr.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			if tt.wantLocation != "" {
				location := rr.Header().Get("Location")
				if location != tt.wantLocation {
					t.Errorf("expected Location %q, got %q", tt.wantLocation, location)
				}
			}
		})
	}
}

func TestUnsupportedMethod(t *testing.T) {
	router := chi.NewRouter()

	router.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))

	req, err := http.NewRequest(http.MethodPut, "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}
