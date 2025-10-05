package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/vzx7/crypto-news-selector/pkg/utils"
)

const (
	pathConfig   = "./config/config.json"
	pathProjects = "./coins.txt"
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
	IntervalStr  string       `json:"interval"`
	RSS          []RSS        `json:"rss"`
	FileSettings FileSettings `json:"file_settings"`
	Projects     []string
	Interval     time.Duration
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

	if cfg.Projects, err = utils.LoadCoinsFromFile(pathProjects); err != nil {
		return nil, err
	}
	// парсим строки в time.Duration
	if cfg.Interval, err = time.ParseDuration(cfg.IntervalStr); err != nil {
		return nil, err
	}
	if cfg.FileSettings.DailyCheckInterval, err = time.ParseDuration(cfg.FileSettings.DailyCheckIntervalStr); err != nil {
		return nil, err
	}

	return &cfg, nil
}
