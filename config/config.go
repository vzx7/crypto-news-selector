package config

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"time"
)

const (
	pathConfig   = "./config/config.json"
	pathProjects = "./projects.txt"
)

type RSS struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type FileSettings struct {
	ArchiveDir            string `json:"archive_dir"`
	DailyCheckIntervalStr string `json:"daily_check_interval"`
	LogRetentionStr       string `json:"log_retention"`
	ArchiveLifeStr        string `json:"archive_life"`
	MaxWorkers            int    `json:"max_workers"`

	// вычисленные значения
	DailyCheckInterval time.Duration
	LogRetention       time.Duration
	ArchiveLife        time.Duration
}

type Config struct {
	IntervalStr    string       `json:"interval"`
	RSS            []RSS        `json:"rss"`
	FileSettings   FileSettings `json:"file_settings"`
	Projects       []string
	ProjectSymbols map[string]string // project -> symbol
	Interval       time.Duration
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(pathConfig)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.ProjectSymbols, err = loadProjectsWithSymbolsFromFile(pathProjects); err != nil {
		return nil, err
	}

	for p := range cfg.ProjectSymbols {
		cfg.Projects = append(cfg.Projects, p)
	}

	if cfg.Interval, err = time.ParseDuration(cfg.IntervalStr); err != nil {
		return nil, err
	}
	if cfg.FileSettings.DailyCheckInterval, err = time.ParseDuration(cfg.FileSettings.DailyCheckIntervalStr); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// loadProjectsWithSymbolsFromFile reads a file with formats "Project Symbol"
func loadProjectsWithSymbolsFromFile(fileName string) (map[string]string, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	projects := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, "|")
		var symbol string
		if len(parts) == 2 {
			symbol = strings.TrimSpace(parts[1])
		}

		projectName := strings.TrimSpace(parts[0])

		projects[projectName] = symbol
	}

	return projects, scanner.Err()
}
