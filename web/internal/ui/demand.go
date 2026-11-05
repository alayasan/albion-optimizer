package ui

import (
	"context"
	"sync"
	"time"

	"github.com/alayasan/albion-optimizer/web/internal/market"
)

// demandHistoryDays is how many days of daily trade history the cache keeps per
// item. It covers the longest selectable window, so changing the window never
// costs another request.
const demandHistoryDays = 30

type cachedDemand struct {
	points  []market.HistoryPoint
	fetched time.Time
}

// demandCache keeps daily trade history per item and hub so the demand columns
// can be summed for any window without asking the market API again. A failed
// fetch backs off for priceRetry, like the price cache.
type demandCache struct {
	client  *market.Client
	quality int
	ttl     time.Duration
	now     func() time.Time

	mu      sync.RWMutex
	entries map[priceKey]cachedDemand
	retryAt time.Time
}

func newDemandCache(client *market.Client, quality int, ttl time.Duration) *demandCache {
	return &demandCache{
		client:  client,
		quality: quality,
		ttl:     ttl,
		now:     time.Now,
		entries: make(map[priceKey]cachedDemand),
	}
}

// Lookup returns how many units of each item and hub were traded in the
// trailing window of days days, counting today. Keys the API has no history for
// are absent from the result, which the template renders as a blank cell.
func (c *demandCache) Lookup(ctx context.Context, keys []priceKey, days int) map[priceKey]int {
	if len(keys) == 0 {
		return nil
	}
	if days < 1 {
		days = 1
	}
	if days > demandHistoryDays {
		days = demandHistoryDays
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

	cutoff := startOfDay(now).AddDate(0, 0, -(days - 1))

	result := make(map[priceKey]int, len(keys))
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, key := range keys {
		entry, ok := c.entries[key]
		if !ok {
			continue
		}
		units := 0
		for _, point := range entry.points {
			if !point.Timestamp.Before(cutoff) {
				units += point.ItemCount
			}
		}
		result[key] = units
	}
	return result
}

// fetch refreshes the daily history of the stale hub and item pairs, one
// request per hub.
func (c *demandCache) fetch(ctx context.Context, stale map[string][]string, now time.Time) {
	ctx, cancel := context.WithTimeout(ctx, priceTimeout)
	defer cancel()

	start := startOfDay(now).AddDate(0, 0, -(demandHistoryDays - 1))
	end := startOfDay(now)

	history := make(map[priceKey][]market.HistoryPoint)
	failed := false
	for city, ids := range stale {
		blocks, err := c.client.History(ctx, ids, market.HistoryQuery{
			Locations: []string{city},
			Qualities: []int{c.quality},
			Start:     start,
			End:       end,
			Scale:     24,
		})
		if err != nil {
			failed = true
			continue
		}
		for _, block := range blocks {
			history[priceKey{id: block.ItemID, city: city}] = block.Data
		}
		// The API omits blocks for items it has no history for, so every
		// requested id is stamped below; otherwise those ids would be retried
		// on every render.
		for _, id := range ids {
			if _, ok := history[priceKey{id: id, city: city}]; !ok {
				history[priceKey{id: id, city: city}] = nil
			}
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if failed {
		c.retryAt = now.Add(priceRetry)
	}
	for key, points := range history {
		c.entries[key] = cachedDemand{points: points, fetched: now}
	}
}

func startOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
