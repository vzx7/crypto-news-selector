package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/vzx7/crypto-news-selector/internal/fetcher"
)

type NewsMessage struct {
	Project   string
	Timestamp string
	Item      fetcher.NewsItem
	PriceUSD  float64
}

var (
	newsList []NewsMessage
	mu       sync.Mutex
	clients  = make(map[chan NewsMessage]bool)
)

// Start запускает веб-сервер на :8080
func Start() {
	http.HandleFunc("/", servePage)
	http.HandleFunc("/events", serveEvents)

	log.Println("🌐 Web UI started at http://localhost:8080")
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal(err)
		}
	}()
}

// AddNews добавляет новость и уведомляет всех клиентов
func AddNews(msg NewsMessage) {
	mu.Lock()
	defer mu.Unlock()

	newsList = append(newsList, msg)
	for ch := range clients {
		select {
		case ch <- msg:
		default:
			// если клиент не читает — удаляем
			delete(clients, ch)
			close(ch)
		}
	}
}

// servePage — HTML-страница
func servePage(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("internal/web/templates/index.html")
	if err != nil {
		http.Error(w, "Template parsing error", http.StatusInternalServerError)
		log.Println("Template parsing error:", err)
		return
	}
	if err := t.Execute(w, nil); err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		log.Println("Template execution error:", err)
	}
}

// serveEvents — поток событий SSE
func serveEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan NewsMessage, 10)
	mu.Lock()
	clients[ch] = true
	// Отправим уже накопленные новости
	for _, n := range newsList {
		data, _ := json.Marshal(n)
		fmt.Fprintf(w, "data: %s\n\n", data)
	}
	mu.Unlock()

	// отдельная горутина — слушаем новые
	go func() {
		for msg := range ch {
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}()

	<-r.Context().Done()
	mu.Lock()
	delete(clients, ch)
	close(ch)
	mu.Unlock()
}
