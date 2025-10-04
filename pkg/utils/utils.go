package utils

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
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

func StripHTML(htmlStr string) string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr
	}
	var sb strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return sb.String()
}
