package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/vzx7/crypto-news-selector/config"
	"github.com/vzx7/crypto-news-selector/internal/fetcher"
	"github.com/vzx7/crypto-news-selector/internal/storage"
)

// NewsMessage хранит новость и привязанную монету
type NewsMessage struct {
	Coin string
	Item fetcher.NewsItem
}

func Run(cfg config.Config) {
	// Инициализация хранилища
	if err := storage.InitStorage(cfg); err != nil {
		log.Fatal("Ошибка инициализации хранилища:", err)
	}

	newsChan := make(chan NewsMessage, 100)

	// Асинхронная обработка новостей
	go func() {
		for msg := range newsChan {
			printNews(msg) // цветной вывод в терминал

			// Форматируем для файла (чистый текст, без цветов)
			formatted := fmt.Sprintf("[%s] %s (link: %s)", msg.Item.Title, msg.Item.Description, msg.Item.Link)
			if err := storage.SaveNews(msg.Coin, []string{formatted}); err != nil {
				log.Println("Ошибка записи новостей:", err)
			}
		}
	}()

	// Периодический сбор новостей с нескольких RSS
	go func() {
		seen := make(map[string]struct{}) // кеш заголовков для устранения дубликатов

		processRSS := func(rssURL string) {
			items, err := fetcher.FetchNews(rssURL, cfg.Projects)
			if err != nil {
				log.Printf("Ошибка при сборе новостей с %s: %v", rssURL, err)
				return
			}

			for _, n := range items {
				if _, exists := seen[n.Title]; exists {
					continue // пропускаем дубликат
				}

				coin := findCoinInTitle(n.Title, cfg.Projects)
				if coin != "" {
					newsChan <- NewsMessage{Coin: coin, Item: n}
					seen[n.Title] = struct{}{}
				}
			}
		}

		// Мгновенный сбор при старте
		for _, rss := range cfg.RSS {
			processRSS(rss.Url)
		}

		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		for range ticker.C {
			for _, rss := range cfg.RSS {
				processRSS(rss.Url)
			}
		}
	}()

	// Ежедневная проверка хранилища
	dailyTicker := time.NewTicker(cfg.FileSettings.DailyCheckInterval)
	defer dailyTicker.Stop()

	for range dailyTicker.C {
		log.Println("Запуск ежедневной проверки хранения новостей...")
		go storage.CleanupAndArchive(cfg.Projects)
	}
}

// printNews выводит новость в терминал с цветами
func printNews(msg NewsMessage) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Красный жирный для монеты
	fmt.Printf("\n[%s] COIN: \033[1;31m%-10s\033[0m\n\n", timestamp, strings.ToUpper(msg.Coin))

	// Зеленый для заголовка
	fmt.Printf("TITLE: \033[32m%s\033[0m\n\n", msg.Item.Title)
	if msg.Item.Description != "" {
		fmt.Printf("DESC: %s\n\n", msg.Item.Description)
	}

	if msg.Item.Content != "" {
		fmt.Printf("CONTENT: %s\n\n", msg.Item.Content)
	}

	// Синий для ссылки
	fmt.Printf("LINK: \033[34m%s\033[0m\n\n", msg.Item.Link)

	fmt.Println(">>>---------------------------------------------------------------------------->>>")
	fmt.Println(">>>---------------------------------------------------------------------------->>>")
}

// findCoinInTitle ищет монету в заголовке новости
func findCoinInTitle(title string, coins []string) string {
	lowerTitle := strings.ToLower(title)
	for _, c := range coins {
		if strings.Contains(lowerTitle, strings.ToLower(c)) {
			return c
		}
	}
	return ""
}
