package storage

import (
	"bufio"
	"encoding/json"
	"go.uber.org/zap"
	"math/rand"
	"os"
	"shorturl/internal/logger"
	"sync"
	"time"
)

// URLPair представляет собой пару короткого и оригинального URL.
type URLPair struct {
	ID          string `json:"id"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// InMemoryStorage представляет собой реализацию хранилища в памяти.
type InMemoryStorage struct {
	mu   sync.RWMutex
	urls map[string]string
}

// NewInMemoryStorage создает и возвращает новый экземпляр InMemoryStorage.
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		urls: make(map[string]string),
	}
}

func (s *InMemoryStorage) CreateShortURL(originalURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	shortID := generateShortID()
	s.urls[shortID] = originalURL
	return shortID, nil
}

func (s *InMemoryStorage) GetOriginalURL(shortID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.urls[shortID]
	if !ok {
		return "", nil // Или вернуть ошибку, в зависимости от логики
	}
	return url, nil
}

// FileStorage представляет собой реализацию хранилища в файле.
type FileStorage struct {
	mu       sync.RWMutex
	urls     map[string]string
	filePath string
}

// NewFileStorage создает и возвращает новый экземпляр FileStorage.
func NewFileStorage(filePath string) *FileStorage {
	s := &FileStorage{
		urls:     make(map[string]string),
		filePath: filePath,
	}
	err := s.LoadFromFile(filePath)
	if err != nil {
		logger.Logger.Error("Error loading data from file", // Используем zap.Error
			zap.String("path", filePath),
			zap.Error(err),
		)
	}
	return s
}

func (s *FileStorage) CreateShortURL(originalURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	shortID := generateShortID()
	err := s.appendToFile(s.filePath, shortID, originalURL)
	if err != nil {
		return "", err
	}
	s.urls[shortID] = originalURL
	return shortID, nil
}

func (s *FileStorage) GetOriginalURL(shortID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	url, ok := s.urls[shortID]
	if !ok {
		return "", nil
	}
	return url, nil
}

func (s *FileStorage) LoadFromFile(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Logger.Error("Error closing file", // Используем zap.Error
				zap.String("path", filePath),
				zap.Error(err),
			)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var pair URLPair
		if err := json.Unmarshal([]byte(line), &pair); err != nil {
			return err
		}
		s.urls[pair.ShortURL] = pair.OriginalURL
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (s *FileStorage) appendToFile(filePath string, shortURL string, originalURL string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Logger.Error("Error closing file", // Используем zap.Error
				zap.String("path", filePath),
				zap.Error(err),
			)
		}
	}()

	id := generateID()
	pair := URLPair{
		ID:          id,
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	jsonData, err := json.Marshal(pair)
	if err != nil {
		return err
	}
	_, err = file.WriteString(string(jsonData) + "\n")
	if err != nil {
		return err
	}
	return nil
}

func generateShortID() string {
	// Простая заглушка для генерации короткого ID
	return generateRandomString(8)
}

func generateID() string {
	// Простая заглушка для генерации ID
	return generateRandomString(16)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var randGen = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[randGen.Intn(len(letterBytes))]
	}
	return string(b)
}
