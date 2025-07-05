package handlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"io"
	"net/http"
	"net/http/httptest"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/service"
	"shorturl/internal/storage"
	"strings"
	"testing"
)

// MockURLService заглушка для тестирования, реализует интерфейс service.URLShortener.
type MockURLService struct {
	URLs map[string]storage.URLPair
}

func (m *MockURLService) CreateShortURL(_ context.Context, userID, originalURL string) (string, error) {
	shortID := generateMockShortID()
	m.URLs[shortID] = storage.URLPair{UserID: userID, OriginalURL: originalURL, ShortURL: shortID}
	return shortID, nil
}

func (m *MockURLService) GetOriginalURL(_ context.Context, shortID string) (string, error) {
	pair, ok := m.URLs[shortID]
	if !ok {
		return "", fmt.Errorf("URL not found")
	}
	return pair.OriginalURL, nil
}

func (m *MockURLService) GetURLsByUserID(_ context.Context, userID string) ([]storage.URLPair, error) {
	var result []storage.URLPair
	for _, pair := range m.URLs {
		if pair.UserID == userID {
			result = append(result, pair)
		}
	}
	return result, nil
}

func NewMockURLService() *MockURLService {
	return &MockURLService{URLs: make(map[string]storage.URLPair)}
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

// TestHandleAPIShorten проверяет обработчик POST-запросов к /api/shorten.
func TestHandleAPIShorten(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	mockSvc := NewMockURLService()
	h := NewHandlers(mockSvc)

	router := chi.NewRouter()
	router.Post("/api/shorten", h.HandleAPIShorten(cfg))

	tests := []struct {
		name           string
		body           string
		expectedCode   int
		expectedResult string // Ожидаемый полный короткий URL
	}{
		{
			name:           "Valid URL",
			body:           `{"url": "http://example.com"}`,
			expectedCode:   http.StatusCreated,
			expectedResult: fmt.Sprintf("%s/%s", cfg.BaseURL, generateMockShortID()),
		},
		{
			name:           "Invalid URL format",
			body:           `{"url": "example.com"}`,
			expectedCode:   http.StatusBadRequest,
			expectedResult: "",
		},
		{
			name:           "Invalid JSON",
			body:           `{"url": "http://example.com"`,
			expectedCode:   http.StatusBadRequest,
			expectedResult: "",
		},
		{
			name:           "Empty JSON",
			body:           `{}`,
			expectedCode:   http.StatusBadRequest,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedCode {
				t.Errorf("Test: %s, Expected status %d, got %d", tt.name, tt.expectedCode, rr.Code)
			}

			if tt.expectedCode == http.StatusCreated {
				bodyBytes, _ := io.ReadAll(rr.Body)
				var resp handlers.ShortenResponse
				if err := json.Unmarshal(bodyBytes, &resp); err != nil {
					t.Fatalf("Test: %s, Failed to unmarshal JSON response: %v", tt.name, err)
				}
				if resp.Result != tt.expectedResult {
					t.Errorf("Test: %s, Expected result %s, got %s", tt.name, tt.expectedResult, resp.Result)
				}
			}
		})
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
	mockSvc.URLs = map[string]storage.URLPair{generateMockShortID(): {OriginalURL: "http://example.com"}}
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
