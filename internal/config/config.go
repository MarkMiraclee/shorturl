package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	ServerAddress string
	BaseURL       string
}

func Load() *Config {
	cfg := &Config{}

	envServerAddress := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")

	var flagServerAddress string
	var flagBaseURL string

	flag.StringVar(&flagServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&flagBaseURL, "b", "", "Base URL for shortened links")

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

	if cfg.BaseURL == "" {
		cfg.BaseURL = fmt.Sprintf("http://%s", cfg.ServerAddress)
	} else {
		cfg.BaseURL = strings.TrimSuffix(cfg.BaseURL, "/")
	}

	return cfg
}
