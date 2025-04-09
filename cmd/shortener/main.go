package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/service"
)

func main() {
	cfg := config.Load() // Загрузка конфигурации приложения.

	svc := service.NewURLService() // Создание экземпляра сервиса для работы с URL.
	h := handlers.NewHandlers(svc) // Создание экземпляра обработчиков HTTP-запросов, передавая ему сервис.

	r := chi.NewRouter()               // Создание нового роутера chi.
	r.Post("/", h.HandlePost(cfg))     // Определение маршрута для POST-запросов на корневой путь.
	r.Get("/{shortID}", h.HandleGet()) // Определение маршрута для GET-запросов с параметром shortID.

	fmt.Printf("Server address from config: %s\n", cfg.ServerAddress)
	fmt.Printf("Starting server on %s\n", cfg.BaseURL)
	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}
