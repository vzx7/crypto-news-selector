package service

import (
	"fmt"
	"log"
	"time"

	"github.com/vzx7/crypto-news-selector/config"
	"github.com/vzx7/crypto-news-selector/internal/fetcher"
	"github.com/vzx7/crypto-news-selector/internal/storage"
	"github.com/vzx7/crypto-news-selector/internal/web"
	"github.com/vzx7/crypto-news-selector/pkg/coingecko"
)

// NewsMessage — структура для новостных сообщений
type NewsMessage struct {
	Project  string
	PriceUSD float64
	Item     fetcher.NewsItem
}

// StartNewsPipeline запускает основной цикл обработки новостей
func StartNewsPipeline(cfg config.Config) {
	newsChan := make(chan NewsMessage, 100)

	go handleIncomingNews(newsChan)
	go pollRSS(cfg, newsChan)
}

// handleIncomingNews — обработчик входящих новостей
func handleIncomingNews(newsChan <-chan NewsMessage) {
	for msg := range newsChan {
		printNews(msg)

		web.AddNews(web.NewsMessage{
			Project:   msg.Project,
			Timestamp: time.Now(),
			PriceUSD:  msg.PriceUSD,
			Item:      msg.Item,
		})

		formatted := formatNewsForStorage(msg)
		if err := storage.SaveNews(msg.Project, []string{formatted}); err != nil {
			log.Println("News recording error:", err)
		}
	}
}

// pollRSS — цикл опроса RSS-источников
func pollRSS(cfg config.Config, newsChan chan<- NewsMessage) {
	seen := make(map[string]struct{})
	priceCache := newPriceCache()

	processRSS := func(rssURL string) {
		items, err := fetcher.FetchNews(rssURL, cfg.Projects)
		if err != nil {
			log.Printf("Error when collecting news from %s: %v", rssURL, err)
			return
		}

		for _, n := range items {
			if _, exists := seen[n.Title]; exists {
				continue
			}

			project := findProjectInTitle(n.Title, cfg.Projects)
			if project == "" {
				continue
			}

			symbol := cfg.ProjectSymbols[project]
			price, ok := priceCache.Get(symbol)
			if !ok {
				time.Sleep(300 * time.Millisecond) // чтобы не заддосить API
				p, err := coingecko.GetPriceUSD(symbol)
				if err != nil {
					log.Printf("Failed to get price for %s: %v", project, err)
					p = 0
				}
				price = p
				priceCache.Set(symbol, price)
			}

			newsChan <- NewsMessage{
				Project:  project,
				Item:     n,
				PriceUSD: price,
			}
			seen[n.Title] = struct{}{}
		}

		priceCache.Clean()
	}

	// мгновенный запуск
	for _, rss := range cfg.RSS {
		logAnalysisTime()
		processRSS(rss.Url)
	}

	// периодический опрос
	ticker := time.NewTicker(cfg.Interval)
	for range ticker.C {
		logAnalysisTime()
		for _, rss := range cfg.RSS {
			processRSS(rss.Url)
		}
	}
}

// formatNewsForStorage — форматирует новость для записи в файл
func formatNewsForStorage(msg NewsMessage) string {
	priceStr := "Price: N/A"
	if msg.PriceUSD > 0 {
		priceStr = fmt.Sprintf("Price: $%.2f", msg.PriceUSD)
	}

	return fmt.Sprintf("[%s] %s (link: %s) %s",
		time.Now().Format("2006-01-02 15:04:05"), msg.Item.Title, msg.Item.Link, priceStr)
}
