package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
    // Создаем новый роутер
    r := chi.NewRouter()
    
    // Подключаем встроенное middleware для логирования запросов
    r.Use(middleware.Logger)
    
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Привет, мир!"))
    })
    fmt.Print("Server Start")

    http.ListenAndServe(":8080", r)
}