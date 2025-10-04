package utils

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

func LoadCoinsFromFile(fileName string) ([]string, error) {
	file, error := os.Open(fileName)
	if error != nil {
		return nil, error
	}

	defer file.Close()

	var coins []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			coins = append(coins, line)
		}
	}
	return coins, scanner.Err()
}

func NormalizeCoinName(name string) string {
	// Заменяем всё, что не буквы/цифры/дефис/подчеркивание на _
	re := regexp.MustCompile(`[^\w\-]+`)
	return re.ReplaceAllString(name, "_")
}
