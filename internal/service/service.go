package service

import (
	"errors"
	"shorturl/internal/generator"
	"strings"
)

// URLShortener определяет интерфейс для сервиса сокращения URL.
type URLShortener interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(shortID string) (string, error)
}

type URLService struct {
	store map[string]string
	gen   *generator.RandomGenerator
}

// NewURLService создает и возвращает новый экземпляр URLService.
func NewURLService() *URLService {
	return &URLService{
		store: make(map[string]string),
		gen:   generator.NewRandomGenerator(),
	}
}

// CreateShortURL создает новую короткую ссылку для заданного оригинального URL.
// Возвращает короткий идентификатор или ошибку, если URL некорректный.
func (s *URLService) CreateShortURL(originalURL string) (string, error) {
	if !strings.HasPrefix(originalURL, "http") {
		return "", errors.New("invalid URL format")
	}

	shortID := s.gen.NewRandomString(8)
	s.store[shortID] = originalURL
	return shortID, nil
}

// GetOriginalURL возвращает оригинальный URL, связанный с заданным коротким идентификатором.
// Возвращает оригинальный URL или ошибку, если идентификатор не найден или некорректен.
func (s *URLService) GetOriginalURL(shortID string) (string, error) {
	if len(shortID) != 8 {
		return "", errors.New("invalid short ID")
	}

	originalURL, ok := s.store[shortID]
	if !ok {
		return "", errors.New("URL not found")
	}
	return originalURL, nil
}
