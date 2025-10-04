package storage

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vzx7/crypto-news-selector/pkg/utils"
)

var (
	NewsDir      string
	ArchiveDir   = "archive"
	LogRetention = 14 * 24 * time.Hour // 2 недели
	ArchiveLife  = 90 * 24 * time.Hour // 3 месяца
	MaxWorkers   = 5                   // максимум одновременно работающих горутин
	semaphore    = make(chan struct{}, MaxWorkers)
	wg           sync.WaitGroup
	coinLocks    sync.Map // mutex для каждого coin
)

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		log.Fatal("Не удалось определить домашний каталог пользователя")
	}
	NewsDir = filepath.Join(home, "news")
}

// InitStorage создаёт директории и мьютексы для монет
func InitStorage(coins []string) error {
	if err := os.MkdirAll(NewsDir, 0755); err != nil {
		return err
	}
	for _, coin := range coins {
		safeCoin := utils.NormalizeCoinName(coin)
		coinDir := filepath.Join(NewsDir, safeCoin)
		if err := os.MkdirAll(coinDir, 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(coinDir, ArchiveDir), 0755); err != nil {
			return err
		}
		coinLocks.Store(safeCoin, &sync.Mutex{})
	}

	// Асинхронная проверка при старте
	go CleanupAndArchive(coins)
	return nil
}

// SaveNews пишет новости в файл и запускает асинхронное архивирование
func SaveNews(coin string, news []string) error {
	safeCoin := utils.NormalizeCoinName(coin)
	today := time.Now().Format("2006-01-02")
	coinDir := filepath.Join(NewsDir, safeCoin)
	filename := filepath.Join(coinDir, fmt.Sprintf("%s.log", today))

	if err := os.MkdirAll(coinDir, 0755); err != nil {
		return err
	}

	// Читаем уже существующие записи, чтобы не писать дубли
	existing := make(map[string]struct{})
	if data, err := os.ReadFile(filename); err == nil {
		lines := strings.Split(string(data), "\n")
		// Ограничим память – учитываем только последние 200 записей
		for i := len(lines) - 200; i < len(lines); i++ {
			if i >= 0 && len(lines[i]) > 0 {
				// Убираем префикс с датой вида [2025-10-04T15:04:05Z]
				if idx := strings.Index(lines[i], "] "); idx != -1 {
					existing[lines[i][idx+2:]] = struct{}{}
				}
			}
		}
	}

	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, n := range news {
		if _, found := existing[n]; found {
			continue // пропускаем дубликат
		}
		line := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), n)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
		// добавляем в map, чтобы не писать дубли в рамках этого вызова
		existing[n] = struct{}{}
	}

	// Асинхронное архивирование старых файлов
	wg.Add(1)
	go func(c string) {
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() { <-semaphore }()
		archiveCoinFiles(c)
	}(safeCoin)

	return nil
}

// CleanupAndArchive проверяет все монеты и архивирует старые файлы
func CleanupAndArchive(coins []string) {
	for _, coin := range coins {
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			archiveCoinFiles(c)
			cleanupOldArchives(c)
		}(coin)
	}
	wg.Wait()
}

// getCoinMutex возвращает мьютекс для монеты, создавая его при необходимости
func getCoinMutex(coin string) *sync.Mutex {
	safeCoin := utils.NormalizeCoinName(coin)
	if m, ok := coinLocks.Load(safeCoin); ok {
		return m.(*sync.Mutex)
	}
	// ленивое создание
	mutex := &sync.Mutex{}
	actual, _ := coinLocks.LoadOrStore(safeCoin, mutex)
	return actual.(*sync.Mutex)
}

// archiveCoinFiles архивирует старые файлы для монеты
func archiveCoinFiles(coin string) {
	mutex := getCoinMutex(coin)
	mutex.Lock()
	defer mutex.Unlock()

	dir := filepath.Join(NewsDir, coin)
	files, _ := filepath.Glob(filepath.Join(dir, "*.log"))

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > LogRetention {
			if err := archiveFile(f, filepath.Join(dir, ArchiveDir)); err == nil {
				os.Remove(f)
			} else {
				log.Println("Ошибка архивирования:", err)
			}
		}
	}
}

// archiveFile создаёт zip-архив из файла
func archiveFile(filePath, archiveDir string) error {
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return err
	}

	zipName := filepath.Join(archiveDir, filepath.Base(filePath)+".zip")
	zipFile, err := os.Create(zipName)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, _ := f.Stat()
	header, _ := zip.FileInfoHeader(info)
	header.Name = filepath.Base(filePath)
	header.Method = zip.Deflate

	writer, _ := w.CreateHeader(header)
	_, err = io.Copy(writer, f)
	return err
}

// cleanupOldArchives удаляет архивы старше ArchiveLife
func cleanupOldArchives(coin string) {
	archivePath := filepath.Join(NewsDir, coin, ArchiveDir)
	files, _ := filepath.Glob(filepath.Join(archivePath, "*.zip"))

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > ArchiveLife {
			os.Remove(f)
		}
	}
}
