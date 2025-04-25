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
	LogLevel        string `env:"LOG_LEVEL" envDefault:"info"`
	LogFormat       string `env:"LOG_FORMAT" envDefault:"json"` // Добавлено поле LogFormat
}

// Load загружает конфигурацию из переменных окружения и флагов командной строки.
// Приоритет: переменные окружения > флаги > значения по умолчанию.
func Load() *Config {
	cfg := &Config{}

	envServerAddress := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")
	envFileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	envLogLevel := os.Getenv("LOG_LEVEL")
	envLogFormat := os.Getenv("LOG_FORMAT") // Загрузка LogFormat

	var flagServerAddress string
	var flagBaseURL string
	var flagLogLevel string
	var flagLogFormat string // Флаг для LogFormat

	flag.StringVar(&flagServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&flagBaseURL, "b", "", "Base URL for shortened links")
	flag.StringVar(&flagLogLevel, "l", "info", "Log level (debug, info, warn, error, fatal)")
	flag.StringVar(&flagLogFormat, "f", "json", "Log format (text, json)") // Добавлен флаг

	flag.Parse()

	fmt.Printf("ENV SERVER_ADDRESS: '%s'\n", envServerAddress)
	fmt.Printf("ENV BASE_URL: '%s'\n", envBaseURL)
	fmt.Printf("ENV FILE_STORAGE_PATH: '%s'\n", envFileStoragePath)
	fmt.Printf("ENV LOG_LEVEL: '%s'\n", envLogLevel)
	fmt.Printf("ENV LOG_FORMAT: '%s'\n", envLogFormat) // Логирование LogFormat
	fmt.Printf("FLAG SERVER_ADDRESS: '%s'\n", flagServerAddress)
	fmt.Printf("FLAG BASE_URL: '%s'\n", flagBaseURL)
	fmt.Printf("FLAG LOG_LEVEL: '%s'\n", flagLogLevel)
	fmt.Printf("FLAG LOG_FORMAT: '%s'\n", flagLogFormat) // Логирование флага LogFormat

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

	if envLogLevel != "" {
		cfg.LogLevel = envLogLevel
	} else {
		cfg.LogLevel = flagLogLevel
	}

	if envLogFormat != "" {
		cfg.LogFormat = envLogFormat
	} else {
		cfg.LogFormat = flagLogFormat
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = fmt.Sprintf("http://%s", cfg.ServerAddress)
	} else {
		cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")
	}

	fmt.Printf("CONFIG SERVER_ADDRESS after load: '%s'\n", cfg.ServerAddress)
	fmt.Printf("CONFIG BASE_URL after load: '%s'\n", cfg.BaseURL)
	fmt.Printf("CONFIG FILE_STORAGE_PATH after load: '%s'\n", cfg.FileStoragePath)
	fmt.Printf("CONFIG LOG_LEVEL after load: '%s'\n", cfg.LogLevel)
	fmt.Printf("CONFIG LOG_FORMAT after load: '%s'\n", cfg.LogFormat) // Логирование итогового LogFormat

	return cfg
}
