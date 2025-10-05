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
	logRetention = 14 * 24 * time.Hour // 2 weeks
	archiveLife  = 90 * 24 * time.Hour // 3 months
	maxWorkers   = 5                   // maximum at the same time working Gorutin
	wg           sync.WaitGroup
	projectLocks sync.Map // mutex for each project
	semaphore    chan struct{}
)

func init() {
	home := os.Getenv("HOME")
	if home == "" {
		log.Fatal("It was not possible to determine the home user's home catalog!")
	}
	newsDir = filepath.Join(home, "news")
}

// setParams configure the configuration, take the fields from the config if it is there
func setParams(cfg config.Config) {
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

// InitStorage
func InitStorage(cfg config.Config) error {
	if err := os.MkdirAll(newsDir, 0755); err != nil {
		return err
	}

	setParams(cfg)

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

	go CleanupAndArchive(cfg.Projects)
	return nil
}

// SaveNews writes news to the file and launches asynchronous archiving
func SaveNews(project string, news []string) error {
	safeProject := utils.NormalizeProjectName(project)
	today := time.Now().Format("2006-01-02")
	projectDir := filepath.Join(newsDir, safeProject)
	filename := filepath.Join(projectDir, fmt.Sprintf("%s.log", today))

	if err := os.MkdirAll(projectDir, 0755); err != nil {
		return err
	}

	// We read existing notes so as not to write duplicate
	existing := make(map[string]struct{})
	if data, err := os.ReadFile(filename); err == nil {
		lines := strings.Split(string(data), "\n")
		// Limit memory - take into account only the last 200 records
		for i := len(lines) - 200; i < len(lines); i++ {
			if i >= 0 && len(lines[i]) > 0 {
				// We remove the prefix with the date of the species [2025-10-04t15: 04: 05z]
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
			continue
		}
		line := fmt.Sprintf("[%s] %s\n", time.Now().Format(time.RFC3339), n)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
		// We both in MAP so as not to write duplicate as part of this call
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

// CleanupAndArchive checks all projects and archives old files
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

// getProjectMutex Returns Mutex for the project, creating it if necessary
func getProjectMutex(project string) *sync.Mutex {
	safeProject := utils.NormalizeProjectName(project)
	if m, ok := projectLocks.Load(safeProject); ok {
		return m.(*sync.Mutex)
	}

	mutex := &sync.Mutex{}
	actual, _ := projectLocks.LoadOrStore(safeProject, mutex)
	return actual.(*sync.Mutex)
}

// archiveProjectFiles Archives old files for projects
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
				log.Println("Superior archiving:", err)
			}
		}
	}
}

// archiveFile Creates a ZIP archive from a file
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

// cleanupOldArchives Removes archives older than ArchiveLife
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
