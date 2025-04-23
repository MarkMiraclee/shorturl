package storage

import (
	"bufio"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

// URLPair представляет собой пару короткого и оригинального URL.
type URLPair struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// URLStorage определяет интерфейс для хранилища URL.
type URLStorage interface {
	CreateShortURL(originalURL string) (string, error)
	GetOriginalURL(shortID string) (string, error)
	LoadFromFile(filePath string) error
	SaveToFile(filePath string) error
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

func (s *InMemoryStorage) LoadFromFile(_ string) error {
	return nil
}

func (s *InMemoryStorage) SaveToFile(_ string) error {
	return nil
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
		log.Printf("Error loading data %s: %v", filePath, err)
	}
	return s
}

func (s *FileStorage) CreateShortURL(originalURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	shortID := generateShortID()
	s.urls[shortID] = originalURL
	// return shortID, s.SaveToFile(s.filePath)
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
			log.Printf("Error closing file %s: %v", filePath, err)
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

func (s *FileStorage) SaveToFile(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Error closing file %s: %v", filePath, err)
		}
	}()

	s.mu.RLock()
	defer s.mu.RUnlock()
	for shortURL, originalURL := range s.urls {
		uuid := generateUUID()
		pair := URLPair{
			UUID:        uuid,
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
	}
	return nil
}

func generateShortID() string {
	// Простая заглушка для генерации короткого ID
	return generateRandomString(8)
}

func generateUUID() string {
	// Простая заглушка для генерации UUID
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
