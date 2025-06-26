package service

import (
	"context"
	"errors"
	"fmt"
	"shorturl/internal/storage"
)

// ErrConflict is a service-level error for URL conflicts.
type ErrConflict struct {
	ExistingShortID string
}

// NewErrConflict creates a new service-level conflict error.
func NewErrConflict(existingShortID string) *ErrConflict {
	return &ErrConflict{ExistingShortID: existingShortID}
}

func (e *ErrConflict) Error() string {
	return fmt.Sprintf("original URL already exists, existing short ID: %s", e.ExistingShortID)
}

// ShortURLCreatorGetter определяет интерфейс для создания и получения коротких URL.
type ShortURLCreatorGetter interface {
	CreateShortURL(ctx context.Context, userID, originalURL string) (string, error)
	GetOriginalURL(ctx context.Context, shortID string) (string, error)
	GetURLsByUserID(ctx context.Context, userID string) ([]storage.URLPair, error)
}

// PersistentStorage определяет интерфейс для хранилищ с возможностью сохранения/загрузки в файл.
type PersistentStorage interface {
	LoadFromFile(filePath string) error
	SaveToFile(filePath string) error
}

// URLStorage - композиция обоих интерфейсов (может использоваться там, где требуется обе функциональности).
type URLStorage interface {
	ShortURLCreatorGetter
	PersistentStorage
}

// URLShortener определяет интерфейс сервиса для сокращения URL.
type URLShortener interface {
	CreateShortURL(ctx context.Context, userID, originalURL string) (string, error)
	GetOriginalURL(ctx context.Context, shortID string) (string, error)
	GetURLsByUserID(ctx context.Context, userID string) ([]storage.URLPair, error)
}

// URLService представляет собой реализацию сервиса сокращения URL.
type URLService struct {
	storage ShortURLCreatorGetter // Сервис зависит только от необходимого интерфейса
}

// NewURLService создает и возвращает новый экземпляр URLService.
func NewURLService(storage ShortURLCreatorGetter) *URLService {
	return &URLService{storage: storage}
}

func (s *URLService) CreateShortURL(ctx context.Context, userID, originalURL string) (string, error) {
	shortID, err := s.storage.CreateShortURL(ctx, userID, originalURL)
	if err != nil {
		var storageConflict *storage.ErrConflict
		if errors.As(err, &storageConflict) {
			return shortID, NewErrConflict(shortID)
		}
		return "", err
	}
	return shortID, nil
}

func (s *URLService) GetOriginalURL(ctx context.Context, shortID string) (string, error) {
	return s.storage.GetOriginalURL(ctx, shortID)
}

func (s *URLService) GetURLsByUserID(ctx context.Context, userID string) ([]storage.URLPair, error) {
	return s.storage.GetURLsByUserID(ctx, userID)
}
