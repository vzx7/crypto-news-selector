package service

import "sync"

type PriceCache struct {
	mu    sync.Mutex
	cache map[string]float64
}

func newPriceCache() *PriceCache {
	return &PriceCache{cache: make(map[string]float64)}
}

func (p *PriceCache) Get(symbol string) (float64, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	val, ok := p.cache[symbol]
	return val, ok
}

func (p *PriceCache) Set(symbol string, price float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache[symbol] = price
}

func (p *PriceCache) Clean() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cache = make(map[string]float64)
}
