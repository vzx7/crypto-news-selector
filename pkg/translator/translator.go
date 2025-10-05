package translator

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type libreTranslateRequest struct {
	Q      string `json:"q"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type libreTranslateResponse struct {
	TranslatedText string `json:"translatedText"`
}

// TranslateDescription translates the text description from ln into ln
func Translate(text string, from string, into string) (string, error) {
	if text == "" {
		return "", nil
	}

	payload := libreTranslateRequest{
		Q:      text,
		Source: from,
		Target: into,
	}
	data, _ := json.Marshal(payload)

	resp, err := http.Post("https://libretranslate.com/translate", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result libreTranslateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.TranslatedText, nil
}
