package service

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/vzx7/crypto-news-selector/config"
	"github.com/vzx7/crypto-news-selector/internal/fetcher"
	"github.com/vzx7/crypto-news-selector/internal/storage"
	"github.com/vzx7/crypto-news-selector/internal/web"
)

// NewsMessage keeps the news and attached project
// NewsMessage keeps the news and attached project
type NewsMessage struct {
	Project  string
	PriceUSD float64
	Item     fetcher.NewsItem
}

type PriceCache struct {
	mu    sync.Mutex
	cache map[string]float64
}

func NewPriceCache() *PriceCache {
	return &PriceCache{cache: make(map[string]float64)}
}

func (p *PriceCache) Get(symbol string) (float64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	val, ok := p.cache[symbol]
	return val, ok
}

func (p *PriceCache) Set(symbol string, price float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache[symbol] = price
}

func Run(cfg config.Config) {
	if err := storage.InitStorage(cfg); err != nil {
		log.Fatal("Error initialization of the storage:", err)
	}

	go web.Start() // запускаем веб-сервер

	newsChan := make(chan NewsMessage, 100)
	//priceCache := NewPriceCache()

	// Обработка новостей из канала
	go func() {
		for msg := range newsChan {
			printNews(msg) // вывод в терминал
			web.AddNews(web.NewsMessage{Project: msg.Project, Item: msg.Item, PriceUSD: msg.PriceUSD})

			// Форматируем для хранения в файл
			formatted := fmt.Sprintf("[%s] %s (link: %s) %s",
				time.Now().Format("2006-01-02 15:04:05"), msg.Item.Title, msg.Item.Link,
				func() string {
					if msg.PriceUSD > 0 {
						return fmt.Sprintf("Price: $%.2f", msg.PriceUSD)
					}
					return "Price: N/A"
				}(),
			)
			if err := storage.SaveNews(msg.Project, []string{formatted}); err != nil {
				log.Println("News recording error:", err)
			}
		}
	}()

	// Горутин для опроса RSS
	go func() {
		seen := make(map[string]struct{}) // кэш заголовков для избежания дублей

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
				if project != "" {
					//symbol := cfg.ProjectSymbols[project]

					// Проверяем кэш
					/* 					price, ok := priceCache.Get(symbol)
					   					if !ok {
					   						// Задержка перед новым запросом
					   						time.Sleep(300 * time.Millisecond)

					   						p, err := coingecko.GetPriceUSD(symbol)
					   						if err != nil {
					   							log.Printf("Failed to get price for %s: %v", project, err)
					   							p = 0
					   						}
					   						price = p
					   						priceCache.Set(symbol, price)
					   					} */

					newsChan <- NewsMessage{
						Project:  project,
						Item:     n,
						PriceUSD: 0.0,
					}
					seen[n.Title] = struct{}{}
				}
			}
		}

		// Мгновенный сбор при старте
		for _, rss := range cfg.RSS {
			logAnalysisTime()
			processRSS(rss.Url)
		}

		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		for range ticker.C {
			logAnalysisTime()
			for _, rss := range cfg.RSS {
				processRSS(rss.Url)
			}
		}
	}()

	// Daily storage check
	dailyTicker := time.NewTicker(cfg.FileSettings.DailyCheckInterval)
	defer dailyTicker.Stop()

	for range dailyTicker.C {
		log.Println("Launching daily news storage checks...")
		go storage.CleanupAndArchive(cfg.Projects)
	}
}

// printNews leads the news to the terminal with flowers
func printNews(msg NewsMessage) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Red fat for the name of the project
	fmt.Printf("\n[%s] PROJECT: \033[1;31m%-10s\033[0m\n\n", timestamp, strings.ToUpper(msg.Project))

	// Green for title
	fmt.Printf("TITLE: \033[32m%s\033[0m\n", msg.Item.Title)

	if msg.Item.Description != "" {
		fmt.Printf("DESC: %s\n\n", msg.Item.Description)
	}

	if msg.Item.Content != "" {
		fmt.Printf("CONTENT: %s\n\n", msg.Item.Content)
	}

	// Blue for link
	fmt.Printf("LINK: \033[34m%s\033[0m\n\n", msg.Item.Link)

	fmt.Println(">>>---------------------------------------------------------------------------->>>")

	// --- Добавляем на веб ---
	web.AddNews(web.NewsMessage{
		Project:   msg.Project,
		Timestamp: timestamp, // добавляем отдельное поле
		Item: fetcher.NewsItem{
			Title:       msg.Item.Title,
			Link:        msg.Item.Link,
			Description: msg.Item.Description,
			Content:     msg.Item.Content,
			Published:   msg.Item.Published,
		},
	})
}

// findProjectInTitle looking for a project in the heading of news
func findProjectInTitle(title string, projects []string) string {
	lowerTitle := strings.ToLower(title)
	for _, c := range projects {
		if strings.Contains(lowerTitle, strings.ToLower(c)) {
			return c
		}
	}
	return ""
}

func logAnalysisTime() {
	fmt.Printf("                 	RSS analysis for projects. Time: %s\n", time.Now().Format("15:04:05"))
	fmt.Println("<==================================================================================>")
}
