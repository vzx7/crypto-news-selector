package main

import (
	"log"

	"github.com/vzx7/crypto-news-selector/config"
	"github.com/vzx7/crypto-news-selector/internal/service"
	"github.com/vzx7/crypto-news-selector/pkg/utils"
)

func main() {
	cfg, err := config.LoadConfig("./config/config.json")
	if err != nil {
		log.Fatal("Ошибка загрузки конфига:", err)
	}

	coins, err := utils.LoadCoinsFromFile("coins.txt")
	if err != nil {
		log.Fatalf("Ошибка при загрузке списка монет: %v", err)
	}

	service.Run(service.Config{
		Interval:           cfg.Interval,
		DailyCheckInterval: cfg.DailyCheckInterval,
		Coins:              coins,
		RSSUrl:             cfg.RSS[0].Url, // пока первый RSS
	})
}
