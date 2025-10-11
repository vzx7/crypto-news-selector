package coingecko

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GetPriceUSD возвращает цену в USD для монеты по символу (id CoinGecko)
func GetPriceUSD(symbol string) (float64, error) {
	if symbol == "" {
		return 0, nil
	}
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	if val, ok := result[symbol]["usd"]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("price not found")
}
