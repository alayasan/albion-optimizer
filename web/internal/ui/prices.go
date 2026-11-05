package ui

import (
	"context"
	"sync"
	"time"

	"github.com/alayasan/albion-optimizer/web/internal/market"
)

const (
	// Materials are priced on the same region as the live feed.
	priceRegion  = market.Asia
	priceQuality = 1

	priceTTL     = 10 * time.Minute
	priceRetry   = time.Minute
	priceTimeout = 5 * time.Second
)

// priceKey identifies one item at one hub.
type priceKey struct {
	id   string
	city string
}

type cachedPrice struct {
	row     market.Price
	fetched time.Time
}

// priceCache keeps the market prices used by the crafting table so rendering a
// page does not hit the market API every time. Entries are refreshed only once
// they are older than ttl, and a failed fetch backs off for priceRetry.
type priceCache struct {
	client *market.Client
	ttl    time.Duration
	now    func() time.Time

	mu      sync.RWMutex
	entries map[priceKey]cachedPrice
	retryAt time.Time
}

func newPriceCache(client *market.Client, ttl time.Duration) *priceCache {
	return &priceCache{
		client:  client,
		ttl:     ttl,
		now:     time.Now,
		entries: make(map[priceKey]cachedPrice),
	}
}

// Lookup returns the current price row per item and hub. Missing or stale keys
// are fetched one request per hub; keys the API has no row for are absent from
// the result, which the templates render as a blank price.
func (c *priceCache) Lookup(ctx context.Context, keys []priceKey) map[priceKey]market.Price {
	if len(keys) == 0 {
		return nil
	}

	c.mu.Lock()
	now := c.now()
	stale := make(map[string][]string)
	for _, key := range keys {
		entry, ok := c.entries[key]
		if !ok || now.Sub(entry.fetched) > c.ttl {
			stale[key.city] = append(stale[key.city], key.id)
		}
	}
	canFetch := now.After(c.retryAt)
	c.mu.Unlock()

	if len(stale) > 0 && canFetch {
		c.fetch(ctx, stale, now)
	}

	result := make(map[priceKey]market.Price, len(keys))
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, key := range keys {
		if entry, ok := c.entries[key]; ok {
			result[key] = entry.row
		}
	}
	return result
}

// fetch refreshes every stale hub, one request per hub.
func (c *priceCache) fetch(ctx context.Context, stale map[string][]string, now time.Time) {
	ctx, cancel := context.WithTimeout(ctx, priceTimeout)
	defer cancel()

	quotes := make(map[priceKey]market.Price)
	failed := false
	for city, ids := range stale {
		rows, err := c.client.Prices(ctx, ids, market.PricesQuery{
			Locations: []string{city},
			Qualities: []int{priceQuality},
		})
		if err != nil {
			failed = true
			continue
		}
		for _, row := range rows {
			quotes[priceKey{id: row.ItemID, city: city}] = row
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if failed {
		c.retryAt = now.Add(priceRetry)
	}
	for key, row := range quotes {
		c.entries[key] = cachedPrice{row: row, fetched: now}
	}
}
