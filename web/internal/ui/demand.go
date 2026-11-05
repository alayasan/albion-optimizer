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

// demandCache keeps daily trade history per item so the demand columns can be
// summed for any window without asking the market API again. A failed fetch
// backs off for ttl, like the price cache.
type demandCache struct {
	client  *market.Client
	city    string
	quality int
	ttl     time.Duration
	now     func() time.Time

	mu      sync.RWMutex
	entries map[string]cachedDemand
	retryAt time.Time
}

func newDemandCache(client *market.Client, city string, quality int, ttl time.Duration) *demandCache {
	return &demandCache{
		client:  client,
		city:    city,
		quality: quality,
		ttl:     ttl,
		now:     time.Now,
		entries: make(map[string]cachedDemand),
	}
}

// Lookup returns how many units of each item were traded in the trailing
// window of days days, counting today. Items the API has no history for are
// absent from the result, which the template renders as a blank cell.
func (c *demandCache) Lookup(ctx context.Context, ids []string, days int) map[string]int {
	if len(ids) == 0 {
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
		c.fetch(ctx, stale, now)
	}

	cutoff := startOfDay(now).AddDate(0, 0, -(days - 1))

	result := make(map[string]int, len(ids))
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, id := range ids {
		entry, ok := c.entries[id]
		if !ok {
			continue
		}
		units := 0
		for _, point := range entry.points {
			if !point.Timestamp.Before(cutoff) {
				units += point.ItemCount
			}
		}
		result[id] = units
	}
	return result
}

// fetch refreshes the daily history of the given items in one request.
func (c *demandCache) fetch(ctx context.Context, ids []string, now time.Time) {
	ctx, cancel := context.WithTimeout(ctx, priceTimeout)
	defer cancel()

	blocks, err := c.client.History(ctx, ids, market.HistoryQuery{
		Locations: []string{c.city},
		Qualities: []int{c.quality},
		Start:     startOfDay(now).AddDate(0, 0, -(demandHistoryDays - 1)),
		End:       startOfDay(now),
		Scale:     24,
	})

	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		c.retryAt = now.Add(priceRetry)
		return
	}
	// The API omits blocks for items it has no history for, so every requested
	// id is stamped as fetched; otherwise those ids would be retried on every
	// render.
	byID := make(map[string][]market.HistoryPoint, len(blocks))
	for _, block := range blocks {
		byID[block.ItemID] = block.Data
	}
	for _, id := range ids {
		c.entries[id] = cachedDemand{points: byID[id], fetched: now}
	}
}

func startOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
