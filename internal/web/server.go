package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/vzx7/crypto-news-selector/internal/fetcher"
)

type NewsMessage struct {
	Project   string
	Timestamp time.Time
	Item      fetcher.NewsItem
	PriceUSD  float64
}

var (
	newsList []NewsMessage
	mu       sync.Mutex
	clients  = make(map[chan NewsMessage]bool)
)

// Start web-service :8080
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

// AddNews adds news and notifies all clients
func AddNews(msg NewsMessage) {
	mu.Lock()
	defer mu.Unlock()

	// delete news older than 3 days and add a new one
	cutoff := time.Now().Add(-72 * time.Hour)
	var filtered []NewsMessage
	for _, n := range newsList {
		if n.Timestamp.After(cutoff) {
			filtered = append(filtered, n)
		}
	}
	filtered = append(filtered, msg)
	newsList = filtered

	// send the news to all connected clients
	for ch := range clients {
		select {
		case ch <- msg:
		default:
			delete(clients, ch)
			close(ch)
		}
	}
}

// servePage — HTML-page
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

// serveEvents — stream of events SSE
func serveEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan NewsMessage, 100)

	mu.Lock()
	clients[ch] = true
	// We will send the already accumulated news
	for _, n := range newsList {
		data, _ := json.Marshal(n)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}
	mu.Unlock()

	// Listening to new messages in the same thread
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-r.Context().Done():
			mu.Lock()
			delete(clients, ch)
			close(ch)
			mu.Unlock()
			return
		}
	}
}
