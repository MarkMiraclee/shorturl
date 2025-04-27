package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
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
		return "", nil
	}
	return url, nil
}

// Merge принимает данные из другой мапы и объединяет их с текущей
func (s *InMemoryStorage) Merge(data map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range data {
		s.urls[k] = v
	}
}

// GetData возвращает копию текущих данных
func (s *InMemoryStorage) GetData() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dataCopy := make(map[string]string)
	for k, v := range s.urls {
		dataCopy[k] = v
	}
	return dataCopy
}

// FileStorage представляет собой реализацию хранилища в файле.
type FileStorage struct {
	mu       sync.RWMutex
	urls     map[string]string // Временно храним для SaveAllFromMemory
	filePath string
}

// NewFileStorage создает и возвращает новый экземпляр FileStorage.
func NewFileStorage(filePath string) *FileStorage {
	s := &FileStorage{
		urls:     make(map[string]string),
		filePath: filePath,
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

// LoadAllToMemory загружает все данные из файла в InMemoryStorage и возвращает информацию об ошибках
func (s *FileStorage) LoadAllToMemory(memStorage *InMemoryStorage) (int, int, error) {
	file, err := os.OpenFile(s.filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return 0, 0, err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Logger.Error("Error closing file in LoadAllToMemory", zap.Error(err), zap.String("path", s.filePath))
		}
	}()

	data := make(map[string]string)
	scanner := bufio.NewScanner(file)
	successfulLoads := 0
	failedLoads := 0
	var firstError error

	for scanner.Scan() {
		line := scanner.Text()
		var pair URLPair
		if err := json.Unmarshal([]byte(line), &pair); err != nil {
			logger.Logger.Warn("Error unmarshalling line in LoadAllToMemory", zap.Error(err), zap.String("line", line))
			failedLoads++
			if firstError == nil {
				firstError = err // Запоминаем первую ошибку
			}
			continue
		}
		data[pair.ShortURL] = pair.OriginalURL
		successfulLoads++
	}
	if err := scanner.Err(); err != nil {
		return successfulLoads, failedLoads, err
	}

	memStorage.Merge(data)

	var loadError error
	if failedLoads > 0 {
		loadError = fmt.Errorf("loaded %d records with %d errors", successfulLoads, failedLoads)
	}

	return successfulLoads, failedLoads, loadError
}

// SaveAllFromMemory сохраняет все данные из InMemoryStorage в файл
func (s *FileStorage) SaveAllFromMemory(memStorage *InMemoryStorage) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := memStorage.GetData()
	file, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // Перезаписываем файл
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Logger.Error("Error closing file in SaveAllFromMemory", zap.Error(err), zap.String("path", s.filePath))
		}
	}()

	encoder := json.NewEncoder(file)
	for shortURL, originalURL := range data {
		pair := URLPair{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		}
		if err := encoder.Encode(pair); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStorage) appendToFile(filePath string, shortURL string, originalURL string) error {
	logger.Logger.Info("Attempting to write to file", zap.String("path", filePath), zap.String("shortURL", shortURL), zap.String("originalURL", originalURL))
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		logger.Logger.Error("Error opening file for writing", zap.Error(err), zap.String("path", filePath))
		return err
	}
	defer func() {
		errClose := file.Close()
		if errClose != nil {
			logger.Logger.Error("Error closing file in appendToFile", zap.Error(errClose), zap.String("path", filePath))
		} else {
			logger.Logger.Info("Successfully closed file after writing", zap.String("path", filePath))
		}
	}()

	pair := URLPair{
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	jsonData, err := json.Marshal(pair)
	if err != nil {
		logger.Logger.Error("Error marshalling JSON", zap.Error(err), zap.String("shortURL", shortURL), zap.String("originalURL", originalURL))
		return err
	}
	_, err = file.WriteString(string(jsonData) + "\n")
	if err != nil {
		logger.Logger.Error("Error writing to file", zap.Error(err), zap.String("path", filePath), zap.String("data", string(jsonData)))
		return err
	}
	errSync := file.Sync()
	if errSync != nil {
		logger.Logger.Error("Error syncing file", zap.Error(errSync), zap.String("path", filePath))
		return errSync
	}
	logger.Logger.Info("Successfully wrote data to file", zap.String("path", filePath), zap.String("shortURL", shortURL), zap.String("originalURL", originalURL))
	return nil
}

func generateShortID() string {
	return generateRandomString(8)
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
