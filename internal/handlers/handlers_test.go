package handlers_test

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"net/http/httptest"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/service"
	"strings"
	"testing"
)

// MockURLService заглушка для тестирования.
type MockURLService struct {
	URLs map[string]string
}

func (m *MockURLService) CreateShortURL(originalURL string) (string, error) {
	shortID := "abcdefgh"
	m.URLs[shortID] = originalURL
	return shortID, nil
}

func (m *MockURLService) GetOriginalURL(shortID string) (string, error) {
	originalURL, ok := m.URLs[shortID]
	if !ok {
		return "", fmt.Errorf("URL not found")
	}
	return originalURL, nil
}

// Проверка, что MockURLService реализует интерфейс service.URLShortener.
var _ service.URLShortener = (*MockURLService)(nil)

func NewMockURLService() *MockURLService {
	return &MockURLService{URLs: make(map[string]string)}
}

func NewHandlers(svc service.URLShortener) *handlers.Handlers {
	return &handlers.Handlers{
		Service: svc,
	}
}

// TestHandlePost проверяет обработчик POST-запросов.
func TestHandlePost(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	mockSvc := NewMockURLService()
	h := NewHandlers(mockSvc)

	router := chi.NewRouter()
	router.Post("/", h.HandlePost(cfg))

	req, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader("http://example.com"))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", rr.Code)
	}

	body, _ := io.ReadAll(rr.Body)
	if !strings.HasPrefix(string(body), cfg.BaseURL) {
		t.Errorf("Expected prefix %s, got %s", cfg.BaseURL, string(body))
	}
}

// TestHandleGet проверяет обработчик GET-запросов.
func TestHandleGet(t *testing.T) {
	mockSvc := NewMockURLService()
	mockSvc.URLs = map[string]string{"abcdefgh": "http://example.com"}
	h := NewHandlers(mockSvc) // Используем MockURLService напрямую

	router := chi.NewRouter()
	router.Get("/{shortID}", h.HandleGet())

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
