package service

import (
	"shorturl/internal/storage"
)

// URLShortener определяет интерфейс для сервиса сокращения URL.
type URLShortener interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(shortID string) (string, error)
}

// URLService реализует интерфейс URLShortener и содержит бизнес-логику.
type URLService struct {
	storage storage.URLStorage
}

// NewURLService создает новый экземпляр URLService с заданным хранилищем.
func NewURLService(storage storage.URLStorage) *URLService {
	return &URLService{
		storage: storage,
	}
}

// CreateShortURL создает новую короткую ссылку и сохраняет ее в хранилище.
func (s *URLService) CreateShortURL(originalURL string) (string, error) {
	return s.storage.CreateShortURL(originalURL)
}

// GetOriginalURL получает оригинальный URL из хранилища по короткому идентификатору.
func (s *URLService) GetOriginalURL(shortID string) (string, error) {
	return s.storage.Get(shortID)
}
