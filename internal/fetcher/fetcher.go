package fetcher

import (
	"log"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
	"github.com/vzx7/crypto-news-selector/pkg/utils"
)

// NewsItem хранит заголовок и ссылку новости
type NewsItem struct {
	Title       string
	Link        string
	Description string
	Content     string
}

// FetchNews опрашивает RSS и возвращает новости по монетам
func FetchNews(rssUrl string, coins []string) ([]NewsItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(rssUrl)
	if err != nil {
		log.Printf("Ошибка при парсинге RSS: %v", err)
		return nil, err
	}

	var items []NewsItem

	for _, item := range feed.Items {
		for _, coin := range coins {
			if strings.Contains(strings.ToLower(item.Title), strings.ToLower(coin)) {
				items = append(items, NewsItem{
					Title:       item.Title,
					Link:        item.Link,
					Description: item.Description,
					Content:     utils.StripHTML(item.Content),
				})
			}
		}
	}
	return items, nil
}

// InitFetcher запускает периодический сбор новостей
func InitFetcher(rssUrl string, coins []string, interval time.Duration, outChan chan<- NewsItem) {
	// Мгновенный запуск
	items, err := FetchNews(rssUrl, coins)
	if err == nil && len(items) > 0 {
		for _, n := range items {
			outChan <- n
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		items, err := FetchNews(rssUrl, coins)
		if err != nil {
			continue
		}
		for _, n := range items {
			outChan <- n
		}
	}
}
