package config

import (
	"encoding/json"
	"os"
	"time"
)

type RSS struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

type Config struct {
	IntervalStr           string `json:"interval"`             // строка из JSON
	DailyCheckIntervalStr string `json:"daily_check_interval"` // строка из JSON
	RSS                   []RSS  `json:"rss"`

	Interval           time.Duration // внутреннее поле
	DailyCheckInterval time.Duration // внутреннее поле
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// парсим строки в time.Duration
	if cfg.Interval, err = time.ParseDuration(cfg.IntervalStr); err != nil {
		return nil, err
	}
	if cfg.DailyCheckInterval, err = time.ParseDuration(cfg.DailyCheckIntervalStr); err != nil {
		return nil, err
	}

	return &cfg, nil
}
