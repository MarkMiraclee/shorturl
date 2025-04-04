package main

import (
	"net/http"
	"shorturl/internal/config"
	"shorturl/internal/handlers"
	"shorturl/internal/router"
)

func main() {
	cfg := config.Load()
	store := &handlers.URLStore{URLs: make(map[string]string)}
	r := router.NewRouter(store, cfg)
	http.ListenAndServe(cfg.ServerAddress, r)
}
