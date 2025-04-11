package handlers_test

import (
	"context"
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

// MockURLService заглушка для тестирования, реализует интерфейс service.URLShortener.
type MockURLService struct {
	URLs map[string]string
}

func (m *MockURLService) CreateShortURL(originalURL string) (string, error) {
	shortID := generateMockShortID()
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

func NewMockURLService() *MockURLService {
	return &MockURLService{URLs: make(map[string]string)}
}

func NewHandlers(svc service.URLShortener) *handlers.Handlers {
	return &handlers.Handlers{
		Service: svc,
	}
}

// Для генерации предсказуемого короткого ID
func generateMockShortID() string {
	return "mockID01"
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
		t.Errorf("Expected %d, got %d", http.StatusCreated, rr.Code)
	}

	body, _ := io.ReadAll(rr.Body)
	expectedPrefix := fmt.Sprintf("%s/%s", cfg.BaseURL, generateMockShortID())
	if !strings.HasPrefix(string(body), expectedPrefix) {
		t.Errorf("Expected prefix %s, got %s", expectedPrefix, string(body))
	}
}

// TestHandleGet проверяет обработчик GET-запросов.
func TestHandleGet(t *testing.T) {
	mockSvc := NewMockURLService()
	mockSvc.URLs = map[string]string{generateMockShortID(): "http://example.com"}
	h := NewHandlers(mockSvc)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/"+generateMockShortID(), nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("shortID", generateMockShortID())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	req = req.WithContext(ctx)

	h.HandleGet().ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected %d, got %d", http.StatusTemporaryRedirect, rr.Code)
	}

	if rr.Header().Get("Location") != "http://example.com" {
		t.Errorf("Invalid Location header")
	}
}
