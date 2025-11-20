package service

import (
	"fmt"
	"strings"
	"time"
)

// findProjectInTitle ищет проект в заголовке
func findProjectInTitle(title string, projects []string) string {
	lowerTitle := strings.ToLower(title)
	for _, c := range projects {
		if strings.Contains(lowerTitle, strings.ToLower(c)) {
			return c
		}
	}
	return ""
}

// logAnalysisTime — вывод времени анализа RSS
func logAnalysisTime() {
	fmt.Printf("\nRSS analysis for projects. Time: %s\n", time.Now().Format("15:04:05"))
	fmt.Println("<==================================================================================>")
}
