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

	"github.com/vzx7/crypto-news-selector/config"
	"github.com/vzx7/crypto-news-selector/pkg/utils"
)

var (
	newsDir      string
	archiveDir   = "archive"
	logRetention = 14 * 24 * time.Hour // 2 недели
	archiveLife  = 90 * 24 * time.Hour // 3 месяца
	maxWorkers   = 5                   // максимум одновременно работающих горутин
	wg           sync.WaitGroup
	projectLocks sync.Map // mutex для каждого project
	semaphore    chan struct{}
)

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		log.Fatal("Не удалось определить домашний каталог пользователя")
	}
	newsDir = filepath.Join(home, "news")
}

func setParams(cfg config.Config) {
	// Если в конфиге есть указание, применить из конфига
	if cfg.FileSettings.MaxWorkers != 0 {
		maxWorkers = cfg.FileSettings.MaxWorkers
	}
	if cfg.FileSettings.ArchiveDir != "" {
		archiveDir = cfg.FileSettings.ArchiveDir
	}
	if cfg.FileSettings.LogRetention != 0 {
		logRetention = cfg.FileSettings.LogRetention
	}
	if cfg.FileSettings.ArchiveLife != 0 {
		archiveLife = cfg.FileSettings.ArchiveLife
	}
}

// InitStorage создаёт директории и мьютексы для монет
func InitStorage(cfg config.Config) error {
	if err := os.MkdirAll(newsDir, 0755); err != nil {
		return err
	}
	setParams(cfg)
	// создаём семафор с нужным количеством воркеров
	semaphore = make(chan struct{}, maxWorkers)

	for _, project := range cfg.Projects {
		safeProject := utils.NormalizeProjectName(project)
		projectDir := filepath.Join(newsDir, safeProject)
		if err := os.MkdirAll(projectDir, 0755); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Join(projectDir, cfg.FileSettings.ArchiveDir), 0755); err != nil {
			return err
		}
		projectLocks.Store(safeProject, &sync.Mutex{})
	}

	// Асинхронная проверка при старте
	go CleanupAndArchive(cfg.Projects)
	return nil
}

// SaveNews пишет новости в файл и запускает асинхронное архивирование
func SaveNews(project string, news []string) error {
	safeProject := utils.NormalizeProjectName(project)
	today := time.Now().Format("2006-01-02")
	projectDir := filepath.Join(newsDir, safeProject)
	filename := filepath.Join(projectDir, fmt.Sprintf("%s.log", today))

	if err := os.MkdirAll(projectDir, 0755); err != nil {
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

	wg.Add(1)
	go func(c string) {
		defer wg.Done()
		semaphore <- struct{}{}
		defer func() { <-semaphore }()
		archiveProjectFiles(c)
	}(safeProject)

	return nil
}

// CleanupAndArchive проверяет все монеты и архивирует старые файлы
func CleanupAndArchive(projects []string) {
	for _, project := range projects {
		wg.Add(1)
		go func(c string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			archiveProjectFiles(c)
			cleanupOldArchives(c)
		}(project)
	}
	wg.Wait()
}

// getProjectMutex возвращает мьютекс для монеты, создавая его при необходимости
func getProjectMutex(project string) *sync.Mutex {
	safeProject := utils.NormalizeProjectName(project)
	if m, ok := projectLocks.Load(safeProject); ok {
		return m.(*sync.Mutex)
	}
	// ленивое создание
	mutex := &sync.Mutex{}
	actual, _ := projectLocks.LoadOrStore(safeProject, mutex)
	return actual.(*sync.Mutex)
}

// archiveProjectFiles архивирует старые файлы для монеты
func archiveProjectFiles(project string) {
	mutex := getProjectMutex(project)
	mutex.Lock()
	defer mutex.Unlock()

	dir := filepath.Join(newsDir, project)
	files, _ := filepath.Glob(filepath.Join(dir, "*.log"))

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > logRetention {
			if err := archiveFile(f, filepath.Join(dir, archiveDir)); err == nil {
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
func cleanupOldArchives(project string) {
	archivePath := filepath.Join(newsDir, project, archiveDir)
	files, _ := filepath.Glob(filepath.Join(archivePath, "*.zip"))

	for _, f := range files {
		info, err := os.Stat(f)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > archiveLife {
			os.Remove(f)
		}
	}
}
