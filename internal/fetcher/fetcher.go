package fetcher

import (
	"log"

	"github.com/mmcdole/gofeed"
	"github.com/vzx7/crypto-news-selector/pkg/utils"
)

// NewsItem stores a title, news link and description
type NewsItem struct {
	Title       string
	Link        string
	Description string
	Content     string
	Published   string
}

// FetchNews interviews RSS and returns news on projects
func FetchNews(rssUrl string, projects []string) ([]NewsItem, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(rssUrl)
	if err != nil {
		log.Printf("RSS parsing error: %v", err)
		return nil, err
	}

	var items []NewsItem

	for _, item := range feed.Items {
		for _, project := range projects {
			if utils.MatchesProjectName(item.Title, project) {
				pubTime := ""
				if item.PublishedParsed != nil {
					pubTime = item.PublishedParsed.Format("2006-01-02 15:04:05")
				}
				items = append(items, NewsItem{
					Title:       item.Title,
					Link:        item.Link,
					Description: utils.StripHTML(item.Description),
					Content:     utils.StripHTML(item.Content),
					Published:   pubTime,
				})
			}
		}
	}
	return items, nil
}
