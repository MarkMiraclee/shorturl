package service

import (
	"context"
	"errors"
	"fmt"
	"shorturl/internal/storage"
)

// ErrConflict — ошибка уровня сервиса для конфликтов URL.
type ErrConflict struct {
	ExistingShortID string
}

// NewErrConflict создаёт ошибку конфликта на уровне сервиса.
func NewErrConflict(existingShortID string) *ErrConflict {
	return &ErrConflict{ExistingShortID: existingShortID}
}

func (e *ErrConflict) Error() string {
	return fmt.Sprintf("original URL already exists, existing short ID: %s", e.ExistingShortID)
}

// deleteTask — задача на удаление URL.
type deleteTask struct {
	userID   string
	shortIDs []string
}

// ShortURLCreatorGetter определяет интерфейс для создания и получения коротких URL.
type ShortURLCreatorGetter interface {
	CreateShortURL(ctx context.Context, userID, originalURL string) (string, error)
	GetOriginalURL(ctx context.Context, shortID string) (string, error)
	GetURLsByUserID(ctx context.Context, userID string) ([]storage.URLPair, error)
	DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error
}

// PersistentStorage определяет интерфейс для хранилищ с возможностью сохранения/загрузки в файл.
type PersistentStorage interface {
	LoadFromFile(filePath string) error
	SaveToFile(filePath string) error
}

// URLStorage - композиция обоих интерфейсов (может использоваться там, где требуется обе функциональность).
type URLStorage interface {
	ShortURLCreatorGetter
	PersistentStorage
}

// URLShortener определяет интерфейс сервиса для сокращения URL.
type URLShortener interface {
	CreateShortURL(ctx context.Context, userID, originalURL string) (string, error)
	GetOriginalURL(ctx context.Context, shortID string) (string, error)
	GetURLsByUserID(ctx context.Context, userID string) ([]storage.URLPair, error)
	Ping(ctx context.Context) error
	DeleteURLs(ctx context.Context, userID string, shortIDs []string) error
}

// Pinger определяет интерфейс для проверки доступности хранилища.
type Pinger interface {
	PingContext(ctx context.Context) error
}

// URLService представляет собой реализацию сервиса сокращения URL.
type URLService struct {
	storage  ShortURLCreatorGetter
	pinger   Pinger
	deleteCh chan deleteTask
	closeCh  chan struct{}
}

// NewURLService создаёт и возвращает новый экземпляр URLService.
func NewURLService(storage ShortURLCreatorGetter, pinger Pinger) *URLService {
	s := &URLService{
		storage:  storage,
		pinger:   pinger,
		deleteCh: make(chan deleteTask, 100), // буферизованный канал
		closeCh:  make(chan struct{}),
	}
	go s.deleteWorker()
	return s
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

func (s *URLService) DeleteURLs(ctx context.Context, userID string, shortIDs []string) error {
	select {
	case s.deleteCh <- deleteTask{userID: userID, shortIDs: shortIDs}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *URLService) deleteWorker() {
	for {
		select {
		case task := <-s.deleteCh:
			// Можно добавить отдельный контекст с таймаутом
			_ = s.storage.DeleteUserURLs(context.Background(), task.userID, task.shortIDs)
		case <-s.closeCh:
			return
		}
	}
}

func (s *URLService) Ping(ctx context.Context) error {
	if s.pinger != nil {
		return s.pinger.PingContext(ctx)
	}
	return nil
}

// Close реализует io.Closer для URLService
func (s *URLService) Close() error {
	close(s.closeCh)
	return nil
}
