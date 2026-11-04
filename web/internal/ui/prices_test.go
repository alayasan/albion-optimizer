package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alayasan/albion-optimizer/web/internal/crafting"
	"github.com/alayasan/albion-optimizer/web/internal/market"
)

// fakeMarket serves /api/v2/stats/prices from a fixed price table and records
// how many requests it received.
type fakeMarket struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests int
	paths    []string
	failing  bool
}

func newFakeMarket(t *testing.T, prices map[string]int) *fakeMarket {
	t.Helper()

	fake := &fakeMarket{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		fake.requests++
		fake.paths = append(fake.paths, r.URL.RequestURI())
		failing := fake.failing
		fake.mu.Unlock()

		if failing {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}

		const prefix = "/api/v2/stats/prices/"
		ids := strings.Split(strings.TrimPrefix(r.URL.Path, prefix), ",")
		rows := make([]string, 0, len(ids))
		for _, id := range ids {
			price, ok := prices[id]
			if !ok {
				continue
			}
			rows = append(rows, fmt.Sprintf(
				`{"item_id":%q,"city":"Caerleon","quality":1,"sell_price_min":%d,`+
					`"sell_price_min_date":"2026-11-03T02:10:00",`+
					`"sell_price_max":0,"sell_price_max_date":"0001-01-01T00:00:00",`+
					`"buy_price_min":0,"buy_price_min_date":"0001-01-01T00:00:00",`+
					`"buy_price_max":0,"buy_price_max_date":"0001-01-01T00:00:00"}`,
				id, price))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, "[%s]", strings.Join(rows, ","))
	}))
	t.Cleanup(fake.server.Close)

	return fake
}

func (f *fakeMarket) client() *market.Client {
	return &market.Client{BaseURL: f.server.URL, HTTP: f.server.Client()}
}

func (f *fakeMarket) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

func (f *fakeMarket) setFailing(failing bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failing = failing
}

func TestPriceCacheServesFromMemory(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{"T4_METALBAR": 1200})
	cache := newPriceCache(fake.client(), "Caerleon", time.Minute)

	for i := 0; i < 3; i++ {
		rows := cache.Lookup(context.Background(), []string{"T4_METALBAR"})
		if got := rows["T4_METALBAR"].SellPriceMin; got != 1200 {
			t.Fatalf("lookup %d: sell price = %d, want 1200", i, got)
		}
	}

	if got := fake.count(); got != 1 {
		t.Errorf("requests = %d, want 1", got)
	}
}

func TestPriceCacheRefetchesAfterTTL(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{"T4_METALBAR": 1200})
	cache := newPriceCache(fake.client(), "Caerleon", time.Minute)

	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }

	cache.Lookup(context.Background(), []string{"T4_METALBAR"})
	now = now.Add(30 * time.Second)
	cache.Lookup(context.Background(), []string{"T4_METALBAR"})
	if got := fake.count(); got != 1 {
		t.Fatalf("requests within ttl = %d, want 1", got)
	}

	now = now.Add(2 * time.Minute)
	cache.Lookup(context.Background(), []string{"T4_METALBAR"})
	if got := fake.count(); got != 2 {
		t.Errorf("requests after ttl = %d, want 2", got)
	}
}

func TestPriceCacheBacksOffAfterFailure(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{"T4_METALBAR": 1200})
	fake.setFailing(true)
	cache := newPriceCache(fake.client(), "Caerleon", time.Minute)

	if rows := cache.Lookup(context.Background(), []string{"T4_METALBAR"}); len(rows) != 0 {
		t.Fatalf("rows = %v, want none on failure", rows)
	}

	// Within the retry window the cache must not hammer the API again.
	cache.Lookup(context.Background(), []string{"T4_METALBAR"})
	if got := fake.count(); got != 1 {
		t.Errorf("requests after failure = %d, want 1", got)
	}
}

func TestPriceCacheUnknownItemIsBlank(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{})
	cache := newPriceCache(fake.client(), "Caerleon", time.Minute)

	if rows := cache.Lookup(context.Background(), []string{"T4_METALBAR"}); len(rows) != 0 {
		t.Errorf("rows = %v, want none", rows)
	}
}

func TestBuildPlanRowsAndPrices(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{
		"T4_METALBAR": 1000,
		"T4_LEATHER":  500,
	})
	cache := newPriceCache(fake.client(), "Caerleon", time.Minute)

	selected := &crafting.Item{
		UniqueName:     "T8_MAIN_SWORD",
		BaseUniqueName: "MAIN_SWORD",
		Label:          "Elder's Broadsword",
	}
	state := crafting.DefaultState()
	state.Item = selected.UniqueName

	table := buildPlan(context.Background(), cache, state, selected)
	rows := table.Rows
	if len(rows) != len(crafting.PlanTiers)*len(crafting.PlanEnchants) {
		t.Fatalf("rows = %d, want %d", len(rows), len(crafting.PlanTiers)*len(crafting.PlanEnchants))
	}
	if want := []string{"Metal Bar", "Leather", "", ""}; !slices.Equal(table.Headers, want) {
		t.Errorf("headers = %v, want %v", table.Headers, want)
	}

	first := rows[0]
	if first.Tier != 4 || first.Enchant != 0 || !first.NewTier {
		t.Errorf("first row = %+v, want T4 enchant 0 with a tier break", first)
	}
	if len(first.Cells) != planIngredientColumns {
		t.Fatalf("cells = %d, want %d", len(first.Cells), planIngredientColumns)
	}
	if first.Cells[0].ItemID != "T4_METALBAR" || first.Cells[0].Count != 16 {
		t.Errorf("cell 0 = %+v, want T4_METALBAR x16", first.Cells[0])
	}
	if first.Cells[0].Label != "Steel Bar" {
		t.Errorf("cell 0 label = %q, want Steel Bar", first.Cells[0].Label)
	}
	if !first.Cells[0].HasPrice || first.Cells[0].Price != 1000 {
		t.Errorf("cell 0 price = %+v, want 1000", first.Cells[0])
	}
	if first.Cells[0].Age != "2026-11-03 02:10" {
		t.Errorf("cell 0 age = %q, want 2026-11-03 02:10", first.Cells[0].Age)
	}
	if first.Cells[2].ItemID != "" || first.Cells[2].HasPrice {
		t.Errorf("unused cell = %+v, want empty", first.Cells[2])
	}

	// Enchanted tiers use level materials, which the fake API does not price.
	if got := rows[1].Cells[0].ItemID; got != "T4_METALBAR_LEVEL1" {
		t.Errorf("row 1 cell 0 = %q, want T4_METALBAR_LEVEL1", got)
	}
	if rows[1].Cells[0].HasPrice {
		t.Errorf("row 1 cell 0 has price %v, want blank", rows[1].Cells[0].Price)
	}

	for i, row := range rows {
		if i == 0 {
			continue
		}
		if wantNewTier := row.Tier != rows[i-1].Tier; row.NewTier != wantNewTier {
			t.Errorf("row %d NewTier = %v, want %v", i, row.NewTier, wantNewTier)
		}
	}
}

func TestBuildPlanUsesOverrides(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{
		"T4_METALBAR": 1000,
		"T4_LEATHER":  500,
	})
	cache := newPriceCache(fake.client(), "Caerleon", time.Minute)

	selected := &crafting.Item{UniqueName: "T8_MAIN_SWORD", BaseUniqueName: "MAIN_SWORD"}
	state := crafting.DefaultState()
	state.Overrides = map[string]float64{"T4_METALBAR": 77}

	cell := buildPlan(context.Background(), cache, state, selected).Rows[0].Cells[0]
	if !cell.HasOverride || cell.Override != 77 || cell.Price != 77 {
		t.Errorf("cell = %+v, want override 77", cell)
	}
	if cell.Age != "" {
		t.Errorf("age = %q, want empty for a manual price", cell.Age)
	}
	if got := fake.count(); got != 1 {
		t.Errorf("requests = %d, want 1 (price fetch still happens for the other material)", got)
	}
}

func TestBuildPlanWithoutSelection(t *testing.T) {
	cache := newPriceCache(newFakeMarket(t, nil).client(), "Caerleon", time.Minute)
	table := buildPlan(context.Background(), cache, crafting.DefaultState(), nil)
	if len(table.Rows) != 0 {
		t.Errorf("rows = %v, want none", table.Rows)
	}
	if len(table.Headers) != planIngredientColumns {
		t.Errorf("headers = %v, want %d blanks", table.Headers, planIngredientColumns)
	}
}
