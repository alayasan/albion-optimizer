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
	priceCity    = "Caerleon"
	priceQuality = 1

	priceTTL     = 10 * time.Minute
	priceRetry   = time.Minute
	priceTimeout = 5 * time.Second
)

type cachedPrice struct {
	row     market.Price
	fetched time.Time
}

// priceCache keeps the material prices used by the crafting table so rendering
// a page does not hit the market API every time. Entries are refreshed only
// once they are older than ttl, and a failed fetch backs off for priceRetry.
type priceCache struct {
	client *market.Client
	city   string
	ttl    time.Duration
	now    func() time.Time

	mu      sync.RWMutex
	entries map[string]cachedPrice
	retryAt time.Time
}

func newPriceCache(client *market.Client, city string, ttl time.Duration) *priceCache {
	return &priceCache{
		client:  client,
		city:    city,
		ttl:     ttl,
		now:     time.Now,
		entries: make(map[string]cachedPrice),
	}
}

// Lookup returns the current price row per item id. Missing or stale ids are
// fetched in one request; ids the API has no row for are absent from the
// result, which the templates render as a blank price.
func (c *priceCache) Lookup(ctx context.Context, ids []string) map[string]market.Price {
	if len(ids) == 0 {
		return nil
	}

	c.mu.Lock()
	now := c.now()
	stale := make([]string, 0, len(ids))
	for _, id := range ids {
		entry, ok := c.entries[id]
		if !ok || now.Sub(entry.fetched) > c.ttl {
			stale = append(stale, id)
		}
	}
	canFetch := now.After(c.retryAt)
	c.mu.Unlock()

	if len(stale) > 0 && canFetch {
		ctx, cancel := context.WithTimeout(ctx, priceTimeout)
		defer cancel()

		rows, err := c.client.Prices(ctx, stale, market.PricesQuery{
			Locations: []string{c.city},
			Qualities: []int{priceQuality},
		})

		c.mu.Lock()
		if err != nil {
			c.retryAt = now.Add(priceRetry)
		} else {
			for _, row := range rows {
				c.entries[row.ItemID] = cachedPrice{row: row, fetched: now}
			}
		}
		c.mu.Unlock()
	}

	result := make(map[string]market.Price, len(ids))
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, id := range ids {
		if entry, ok := c.entries[id]; ok {
			result[id] = entry.row
		}
	}
	return result
}
