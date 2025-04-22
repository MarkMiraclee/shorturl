package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	FileStoragePath string
}

// Load загружает конфигурацию из переменных окружения и флагов командной строки.
// Приоритет: переменные окружения > флаги > значения по умолчанию.
func Load() *Config {
	cfg := &Config{
		ServerAddress:   ":8080",
		BaseURL:         "http://localhost:8080",
		FileStoragePath: "shortener.json",
	}

	flag.StringVar(&cfg.ServerAddress, "a", os.Getenv("SERVER_ADDRESS"), "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", os.Getenv("BASE_URL"), "Base URL for short links")
	flag.StringVar(&cfg.FileStoragePath, "f", os.Getenv("FILE_STORAGE_PATH"), "File storage path for short URLs")

	flag.Parse()

	// Если переменная окружения установлена, она имеет приоритет
	if envAddr := os.Getenv("SERVER_ADDRESS"); envAddr != "" {
		cfg.ServerAddress = envAddr
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}
	if envFilePath := os.Getenv("FILE_STORAGE_PATH"); envFilePath != "" {
		cfg.FileStoragePath = envFilePath
	}

	return cfg
}
