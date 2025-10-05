package main

import (
	"log"

	"github.com/vzx7/crypto-news-selector/config"
	"github.com/vzx7/crypto-news-selector/internal/service"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Ошибка загрузки конфига:", err)
	}

	service.Run(*cfg)
}
