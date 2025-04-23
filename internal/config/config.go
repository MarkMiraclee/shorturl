package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	ServerAddress   string
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
}

// Load загружает конфигурацию из переменных окружения и флагов командной строки.
// Приоритет: переменные окружения > флаги > значения по умолчанию.
func Load() *Config {
	cfg := &Config{}

	envServerAddress := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")
	envFileStoragePath := os.Getenv("FILE_STORAGE_PATH") // Загрузка FileStoragePath

	var flagServerAddress string
	var flagBaseURL string

	flag.StringVar(&flagServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&flagBaseURL, "b", "", "Base URL for shortened links")

	flag.Parse()

	fmt.Printf("ENV SERVER_ADDRESS: '%s'\n", envServerAddress)
	fmt.Printf("ENV BASE_URL: '%s'\n", envBaseURL)
	fmt.Printf("ENV FILE_STORAGE_PATH: '%s'\n", envFileStoragePath) // Логирование FileStoragePath
	fmt.Printf("FLAG SERVER_ADDRESS: '%s'\n", flagServerAddress)
	fmt.Printf("FLAG BASE_URL: '%s'\n", flagBaseURL)

	if envServerAddress != "" {
		cfg.ServerAddress = envServerAddress
	} else {
		cfg.ServerAddress = flagServerAddress
	}

	if envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	} else {
		cfg.BaseURL = flagBaseURL
	}

	cfg.FileStoragePath = envFileStoragePath

	if cfg.BaseURL == "" {
		cfg.BaseURL = fmt.Sprintf("http://%s", cfg.ServerAddress)
	} else {
		cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")
	}

	fmt.Printf("CONFIG SERVER_ADDRESS after load: '%s'\n", cfg.ServerAddress)
	fmt.Printf("CONFIG BASE_URL after load: '%s'\n", cfg.BaseURL)
	fmt.Printf("CONFIG FILE_STORAGE_PATH after load: '%s'\n", cfg.FileStoragePath) // Логирование FileStoragePath после загрузки

	return cfg
}