package storage

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

// URLStorage определяет интерфейс для взаимодействия с хранилищем URL.
type URLStorage interface {
	CreateShortURL(originalURL string) (string, error)
	Get(shortID string) (string, error)
}

// InMemoryStorage реализует интерфейс URLStorage, используя map в памяти.
type InMemoryStorage struct {
	store map[string]string
	mu    sync.RWMutex
	r     *rand.Rand // Генератор случайных строк
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		store: make(map[string]string),
		mu:    sync.RWMutex{},
		r:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// CreateShortURL создает новую короткую ссылку и сохраняет ее в хранилище.
func (s *InMemoryStorage) CreateShortURL(originalURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	shortID := make([]byte, 8)
	for i := range shortID {
		shortID[i] = chars[s.r.Intn(len(chars))]
	}
	shortIDStr := string(shortID)
	s.store[shortIDStr] = originalURL
	return shortIDStr, nil
}

// Get возвращает оригинальный URL по короткому идентификатору из хранилища.
func (s *InMemoryStorage) Get(shortID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	originalURL, ok := s.store[shortID]
	if !ok {
		return "", errors.New("URL not found")
	}
	return originalURL, nil
}
