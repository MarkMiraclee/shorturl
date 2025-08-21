package storage

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"shorturl/internal/logger"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/lib/pq"
)

// ErrConflict указывает на нарушение уникальности для оригинального URL.
// Включает существующий короткий ID.
type ErrConflict struct {
	ExistingShortID string
}

func NewErrConflict(existingShortID string) *ErrConflict {
	return &ErrConflict{ExistingShortID: existingShortID}
}

func (e *ErrConflict) Error() string {
	return fmt.Sprintf("original URL already exists, existing short ID: %s", e.ExistingShortID)
}

// ErrURLDeleted индикатор удаления URL
var ErrURLDeleted = errors.New("url is deleted")

type DatabaseStorage struct {
	db *sql.DB
}

// NewDatabaseStorage создает и возвращает новый экземпляр DatabaseStorage.
func NewDatabaseStorage(dsn string) (*DatabaseStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверяем соединение с базой данных
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Проверяем и создаем таблицу urls, если она не существует
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS urls (
			short_url    TEXT PRIMARY KEY,
			original_url TEXT NOT NULL UNIQUE,
			user_id      TEXT,
			is_deleted   BOOLEAN NOT NULL DEFAULT FALSE
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create or check table: %w", err)
	}

	_, err = db.ExecContext(context.Background(), `CREATE INDEX IF NOT EXISTS user_id_idx ON urls (user_id);`)
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	logger.Logger.Info("Successfully connected to PostgreSQL and ensured table 'urls' exists")
	return &DatabaseStorage{db: db}, nil
}

func (s *DatabaseStorage) CreateShortURL(ctx context.Context, userID, originalURL string) (string, error) {
	candidateShortID := generateShortID()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			logger.Logger.Error("failed to rollback transaction", zap.Error(err))
		}
	}()

	result, err := tx.ExecContext(ctx,
		`INSERT INTO urls (short_url, original_url, user_id) VALUES ($1, $2, $3)
		 ON CONFLICT (original_url) DO NOTHING`,
		candidateShortID, originalURL, userID)

	if err != nil {
		return "", fmt.Errorf("failed to execute insert on conflict: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		if err := tx.Commit(); err != nil {
			return "", fmt.Errorf("failed to commit transaction for new insert: %w", err)
		}
		return candidateShortID, nil
	}

	var existingShortID string
	err = tx.QueryRowContext(ctx,
		"SELECT short_url FROM urls WHERE original_url = $1",
		originalURL).Scan(&existingShortID)

	if err != nil {
		return "", fmt.Errorf("conflict occurred but failed to retrieve existing short_id: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction for conflict case: %w", err)
	}
	return existingShortID, NewErrConflict(existingShortID)
}

func (s *DatabaseStorage) GetOriginalURL(ctx context.Context, shortID string) (string, error) {
	var originalURL string
	var isDeleted bool
	err := s.db.QueryRowContext(ctx,
		"SELECT original_url, is_deleted FROM urls WHERE short_url = $1",
		shortID).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get original URL: %w", err)
	}
	if isDeleted {
		return "", ErrURLDeleted
	}
	return originalURL, nil
}

func (s *DatabaseStorage) GetURLsByUserID(ctx context.Context, userID string) ([]URLPair, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT short_url, original_url FROM urls WHERE user_id = $1 AND is_deleted = FALSE", userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query urls by user id: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Logger.Error("failed to close rows", zap.Error(err))
		}
	}()

	var urls []URLPair
	for rows.Next() {
		var pair URLPair
		pair.UserID = userID
		if err := rows.Scan(&pair.ShortURL, &pair.OriginalURL); err != nil {
			return nil, fmt.Errorf("failed to scan url pair: %w", err)
		}
		urls = append(urls, pair)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return urls, nil
}

func (s *DatabaseStorage) DeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	if len(shortIDs) == 0 {
		return nil
	}
	query := `UPDATE urls SET is_deleted = TRUE WHERE user_id = $1 AND short_url = ANY($2)`
	_, err := s.db.ExecContext(ctx, query, userID, pq.Array(shortIDs))
	if err != nil {
		return fmt.Errorf("failed to batch delete urls: %w", err)
	}
	return nil
}

func (s *DatabaseStorage) PingContext(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *DatabaseStorage) Close() error {
	return s.db.Close()
}

// InMemoryStorage представляет собой реализацию хранилища в памяти.
type InMemoryStorage struct {
	mu   sync.RWMutex
	urls map[string]URLPair
}

// NewInMemoryStorage создает и возвращает новый экземпляр InMemoryStorage.
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		urls: make(map[string]URLPair),
	}
}

func (s *InMemoryStorage) CreateShortURL(_ context.Context, userID, originalURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	shortID := generateShortID()
	s.urls[shortID] = URLPair{
		ShortURL:    shortID,
		OriginalURL: originalURL,
		UserID:      userID,
		DeletedFlag: false,
	}
	return shortID, nil
}

func (s *InMemoryStorage) DeleteUserURLs(_ context.Context, userID string, shortIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, shortID := range shortIDs {
		if pair, ok := s.urls[shortID]; ok {
			if pair.UserID == userID {
				pair.DeletedFlag = true
				s.urls[shortID] = pair
			}
		}
	}
	return nil
}

func (s *InMemoryStorage) GetOriginalURL(_ context.Context, shortID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pair, ok := s.urls[shortID]
	if !ok {
		return "", nil
	}
	if pair.DeletedFlag {
		return "", ErrURLDeleted
	}
	return pair.OriginalURL, nil
}

func (s *InMemoryStorage) GetURLsByUserID(_ context.Context, userID string) ([]URLPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var userURLs []URLPair
	for _, pair := range s.urls {
		if pair.UserID == userID && !pair.DeletedFlag {
			userURLs = append(userURLs, pair)
		}
	}
	return userURLs, nil
}

// URLPair представляет собой пару короткого и оригинального URL.
type URLPair struct {
	UUID        string `json:"id"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id,omitempty"`
	DeletedFlag bool   `json:"-"`
}

// FileStorage представляет собой реализацию хранилища в файле.
type FileStorage struct {
	mu       sync.RWMutex
	urls     map[string]URLPair
	filePath string
}

// NewFileStorage создает и возвращает новый экземпляр FileStorage.
func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		urls:     make(map[string]URLPair),
		filePath: filePath,
	}
	if err := fs.loadFromFile(); err != nil {
		return nil, err
	}
	return fs, nil
}

func (s *FileStorage) CreateShortURL(_ context.Context, userID, originalURL string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	shortID := generateShortID()
	pair := URLPair{UserID: userID, ShortURL: shortID, OriginalURL: originalURL, DeletedFlag: false}
	if err := s.appendToFile(&pair); err != nil {
		return "", err
	}
	s.urls[shortID] = pair
	return shortID, nil
}

func (s *FileStorage) DeleteUserURLs(_ context.Context, userID string, shortIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, shortID := range shortIDs {
		if pair, ok := s.urls[shortID]; ok {
			if pair.UserID == userID {
				pair.DeletedFlag = true
				s.urls[shortID] = pair
			}
		}
	}
	return nil
}

func (s *FileStorage) GetOriginalURL(_ context.Context, shortID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pair, ok := s.urls[shortID]
	if !ok {
		return "", nil
	}
	if pair.DeletedFlag {
		return "", ErrURLDeleted
	}
	return pair.OriginalURL, nil
}

func (s *FileStorage) GetURLsByUserID(_ context.Context, userID string) ([]URLPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var userURLs []URLPair
	for _, pair := range s.urls {
		if pair.UserID == userID && !pair.DeletedFlag {
			userURLs = append(userURLs, pair)
		}
	}
	return userURLs, nil
}

func (s *FileStorage) loadFromFile() error {
	file, err := os.OpenFile(s.filePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Logger.Error("failed to close file in loadFromFile", zap.Error(err))
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		var pair URLPair
		if err := json.Unmarshal([]byte(line), &pair); err != nil {
			logger.Logger.Warn("Error unmarshalling line from file storage", zap.Error(err), zap.String("line", line))
			continue
		}
		s.urls[pair.ShortURL] = pair
	}
	return scanner.Err()
}

func (s *FileStorage) appendToFile(pair *URLPair) error {
	file, err := os.OpenFile(s.filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Logger.Error("failed to close file in appendToFile", zap.Error(err))
		}
	}()

	pair.UUID = uuid.NewString()
	encoder := json.NewEncoder(file)

	return encoder.Encode(pair)
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
