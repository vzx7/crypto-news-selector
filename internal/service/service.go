package service

import (
	"log"
	"time"

	"github.com/vzx7/crypto-news-selector/config"
	"github.com/vzx7/crypto-news-selector/internal/storage"
	"github.com/vzx7/crypto-news-selector/internal/web"
)

// Run — точка входа модуля сервисов
func Run(cfg config.Config) {
	if err := storage.InitStorage(cfg); err != nil {
		log.Fatal("Error initialization of the storage:", err)
	}

	// запуск веб-сервера
	go web.Start()

	// запуск основного пайплайна обработки новостей
	go StartNewsPipeline(cfg)

	// периодическая проверка хранилища
	startStorageMaintenance(cfg)

	select {} // блокируем main
}

// startStorageMaintenance — ежедневная проверка и очистка хранилища
func startStorageMaintenance(cfg config.Config) {
	ticker := time.NewTicker(cfg.FileSettings.DailyCheckInterval)
	go func() {
		for range ticker.C {
			log.Println("Launching daily news storage checks...")
			go storage.CleanupAndArchive(cfg.Projects)
		}
	}()
}
