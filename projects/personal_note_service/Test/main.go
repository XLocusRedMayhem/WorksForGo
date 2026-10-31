package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"
)

// Данные для передачи в шаблоны
type PageData struct {
	Title   string
	Message string
	Year    int
}

var templates = template.Must(template.ParseGlob("templates/*.html"))

func renderTemplate(w http.ResponseWriter, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Ошибка шаблона %s: %v", name, err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
	}
}

// Главная страница
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		notFoundHandler(w, r)
		return
	}
	renderTemplate(w, "index.html", PageData{
		Title:   "Главная",
		Message: "Добро пожаловать на мой сайт на Go!",
		Year:    time.Now().Year(),
	})
}

// Страница "О нас"
func aboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "about.html", PageData{
		Title:   "О нас",
		Message: "Мы пишем на Go — быстро, просто и надёжно.",
		Year:    time.Now().Year(),
	})
}

// API-эндпоинт
func apiHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp := map[string]interface{}{
		"status":  "ok",
		"time":    time.Now().Format(time.RFC3339),
		"service": "go-site",
	}
	json.NewEncoder(w).Encode(resp)
}

// Обработка 404
func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	renderTemplate(w, "404.html", PageData{
		Title:   "404 — Страница не найдена",
		Message: "Упс! Такой страницы нет.",
		Year:    time.Now().Year(),
	})
}

// Middleware: логирование
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	mux := http.NewServeMux()

	// Статические файлы
	fs := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Маршруты
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/about", aboutHandler)
	mux.HandleFunc("/api/health", apiHandler)
	mux.HandleFunc("/404", notFoundHandler)

	// Обёртка middleware
	handler := loggingMiddleware(mux)

	addr := ":8080"
	log.Printf("Сервер запущен на http://localhost%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}
