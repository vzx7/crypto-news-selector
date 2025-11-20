package service

import (
	"fmt"
	"strings"
	"time"
)

// printNews красиво выводит новость в консоль
func printNews(msg NewsMessage) {
	timestamp := time.Now()

	fmt.Printf("\n[%s] PROJECT: \033[1;31m%-10s\033[0m\n\n", timestamp.Format("2006-01-02 15:04:05"), strings.ToUpper(msg.Project))
	fmt.Printf("TITLE: \033[32m%s\033[0m\n", msg.Item.Title)

	if msg.Item.Description != "" {
		fmt.Printf("DESC: %s\n\n", msg.Item.Description)
	}
	if msg.Item.Content != "" {
		fmt.Printf("CONTENT: %s\n\n", msg.Item.Content)
	}

	fmt.Printf("LINK: \033[34m%s\033[0m\n\n", msg.Item.Link)
	fmt.Println(">>>---------------------------------------------------------------------------->>>")
}
