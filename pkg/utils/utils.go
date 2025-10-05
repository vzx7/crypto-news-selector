package utils

import (
	"bufio"
	"os"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// LoadProjectsFromFile Upload projects from a file
func LoadProjectsFromFile(fileName string) ([]string, error) {
	file, error := os.Open(fileName)
	if error != nil {
		return nil, error
	}

	defer file.Close()

	var projects []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			projects = append(projects, line)
		}
	}
	return projects, scanner.Err()
}

// NormalizeProjectName We replace everything that is not letters/numbers/hyphen/emphasizing _
func NormalizeProjectName(name string) string {
	re := regexp.MustCompile(`[^\w\-]+`)
	return re.ReplaceAllString(name, "_")
}

// StripHTML Clean text from html tags
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

// MatchesProjectName Find all the entry of the project in the header
func MatchesProjectName(title, project string) bool {
	title = strings.ToLower(title)
	project = strings.ToLower(project)

	parts := strings.Fields(project)
	if len(parts) > 1 {
		// Between the words we admit spaces, hyphen or absence of a separator
		pattern := `\b` + strings.Join(parts, `[-_ ]*`) + `\b`
		matched, _ := regexp.MatchString(pattern, title)
		return matched
	}

	// иначе — обычное слово
	pattern := `\b` + regexp.QuoteMeta(project) + `\b`
	matched, _ := regexp.MatchString(pattern, title)
	return matched
}
