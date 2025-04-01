package main

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"shorturl/internal/config"
	"strings"
	"testing"
)

func TestHandlePost(t *testing.T) {
	store := &URLStore{urls: make(map[string]string)}
	cfg := &config.Config{BaseURL: "http://test"}

	router := chi.NewRouter()
	router.Post("/", handlePost(store, cfg))

	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader("http://example.com"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", rr.Code)
	}

	if !strings.HasPrefix(rr.Body.String(), cfg.BaseURL) {
		t.Errorf("Expected prefix %s", cfg.BaseURL)
	}
}

func TestHandleGet(t *testing.T) {
	store := &URLStore{urls: map[string]string{"abcdefgh": "http://example.com"}}

	router := chi.NewRouter()
	router.Get("/{shortID}", handleGet(store))

	req, _ := http.NewRequest(http.MethodGet, "/abcdefgh", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected 307, got %d", rr.Code)
	}

	if rr.Header().Get("Location") != "http://example.com" {
		t.Errorf("Invalid Location header")
	}
}
