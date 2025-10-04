package service

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/vzx7/crypto-news-selector/internal/fetcher"
	"github.com/vzx7/crypto-news-selector/internal/storage"
)

type Config struct {
	Coins              []string
	RSSUrl             string
	Interval           time.Duration
	DailyCheckInterval time.Duration
}

// NewsMessage хранит новость и привязанную монету
type NewsMessage struct {
	Coin string
	Item fetcher.NewsItem
}

func Run(cfg Config) {
	// Инициализация хранилища
	if err := storage.InitStorage(cfg.Coins); err != nil {
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

	// Периодический сбор новостей
	go func() {
		// Мгновенный сбор
		items, err := fetcher.FetchNews(cfg.RSSUrl, cfg.Coins)
		if err == nil && len(items) > 0 {
			for _, n := range items {
				coin := findCoinInTitle(n.Title, cfg.Coins)
				if coin != "" {
					newsChan <- NewsMessage{Coin: coin, Item: n}
				}
			}
		}

		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		for range ticker.C {
			items, err := fetcher.FetchNews(cfg.RSSUrl, cfg.Coins)
			if err != nil {
				continue
			}
			for _, n := range items {
				coin := findCoinInTitle(n.Title, cfg.Coins)
				if coin != "" {
					newsChan <- NewsMessage{Coin: coin, Item: n}
				}
			}
		}
	}()

	// Ежедневная проверка хранилища
	dailyTicker := time.NewTicker(cfg.DailyCheckInterval)
	defer dailyTicker.Stop()

	for range dailyTicker.C {
		log.Println("Запуск ежедневной проверки хранения новостей...")
		go storage.CleanupAndArchive(cfg.Coins)
	}
}

// printNews выводит новость в терминал с цветами
func printNews(msg NewsMessage) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	// Красный жирный для монеты
	fmt.Printf("\n[%s] COIN: \033[1;31m%-10s\033[0m\n", timestamp, strings.ToUpper(msg.Coin))

	// Зеленый для заголовка
	fmt.Printf("TITLE: \033[32m%s\033[0m\n", msg.Item.Title)
	if msg.Item.Description != "" {
		fmt.Printf("DESC: %s\n\n", msg.Item.Description)
	}

	fmt.Printf("CONTENT: %s\n\n", msg.Item.Content)

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
