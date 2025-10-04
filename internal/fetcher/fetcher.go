package fetcher

import (
	"log"
	"strings"

	"github.com/mmcdole/gofeed"
	"github.com/vzx7/crypto-news-selector/pkg/utils"
)

// NewsItem хранит заголовок, ссылку новости и описание
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
