package fetcher

import (
	"log"
	"regexp"
	"strings"

	"github.com/mmcdole/gofeed"
	"github.com/vzx7/crypto-news-selector/pkg/utils"
)

// NewsItem stores a title, news link and description
type NewsItem struct {
	Title       string
	Link        string
	Description string
	Content     string
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
			pattern := `\b` + regexp.QuoteMeta(project) + `\b`
			matched, _ := regexp.MatchString(pattern, strings.ToLower(item.Title))
			if matched {
				items = append(items, NewsItem{
					Title:       item.Title,
					Link:        item.Link,
					Description: utils.StripHTML(item.Description),
					Content:     utils.StripHTML(item.Content),
				})
			}
		}
	}
	return items, nil
}
