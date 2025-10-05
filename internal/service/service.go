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

// NewsMessage keeps the news and attached project
type NewsMessage struct {
	Project string
	Item    fetcher.NewsItem
}

func Run(cfg config.Config) {
	if err := storage.InitStorage(cfg); err != nil {
		log.Fatal("Error initialization of the storage:", err)
	}

	newsChan := make(chan NewsMessage, 100)

	go func() {
		for msg := range newsChan {
			printNews(msg)

			// We format for a file (clean text, without colors)
			formatted := fmt.Sprintf("[%s] %s (link: %s)", msg.Item.Title, msg.Item.Description, msg.Item.Link)
			if err := storage.SaveNews(msg.Project, []string{formatted}); err != nil {
				log.Println("News recording error:", err)
			}
		}
	}()

	go func() {
		// cache headers to eliminate duplicates
		seen := make(map[string]struct{})

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
					newsChan <- NewsMessage{Project: project, Item: n}
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
	fmt.Println("\n<==================================================================================>")
	fmt.Printf("                     RSS analysis for projects. Time: %s\n", time.Now().Format("15:04:05"))
	fmt.Println("<==================================================================================>")
}
