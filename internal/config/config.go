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
	LogFormat       string `env:"LOG_FORMAT" envDefault:"json"`
}

// String реализует интерфейс fmt.Stringer для структуры Config.
func (c *Config) String() string {
	return fmt.Sprintf(
		"Config: "+
			"ServerAddress='%s', "+
			"BaseURL='%s', "+
			"FileStoragePath='%s', "+
			"LogLevel='%s', "+
			"LogFormat='%s'",
		c.ServerAddress,
		c.BaseURL,
		c.FileStoragePath,
		c.LogLevel,
		c.LogFormat,
	)
}

// Load загружает конфигурацию из переменных окружения и флагов командной строки.
// Приоритет: переменные окружения > флаги > значения по умолчанию.
func Load() *Config {
	cfg := &Config{}

	envServerAddress := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")
	envFileStoragePath := os.Getenv("FILE_STORAGE_PATH")
	envLogLevel := os.Getenv("LOG_LEVEL")
	envLogFormat := os.Getenv("LOG_FORMAT")

	var flagServerAddress string
	var flagBaseURL string
	var flagLogLevel string
	var flagLogFormat string

	flag.StringVar(&flagServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&flagBaseURL, "b", "", "Base URL for shortened links")
	flag.StringVar(&flagLogLevel, "l", "info", "Log level (debug, info, warn, error, fatal)")
	flag.StringVar(&flagLogFormat, "f", "json", "Log format (text, json)")

	flag.Parse()

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

	return cfg
}
