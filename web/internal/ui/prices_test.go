package ui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alayasan/albion-optimizer/web/internal/crafting"
	"github.com/alayasan/albion-optimizer/web/internal/market"
)

// fakeMarket serves /api/v2/stats/prices and /api/v2/stats/history from fixed
// tables and records how many requests it received.
type fakeMarket struct {
	server   *httptest.Server
	mu       sync.Mutex
	requests int
	paths    []string
	failing  bool
	history  map[string]int
}

func newFakeMarket(t *testing.T, prices map[string]int) *fakeMarket {
	t.Helper()

	fake := &fakeMarket{}
	fake.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		fake.requests++
		fake.paths = append(fake.paths, r.URL.RequestURI())
		failing := fake.failing
		history := fake.history
		fake.mu.Unlock()

		if failing {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/api/v2/stats/history/") {
			writeHistory(t, w, r, history)
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

// writeHistory answers one daily bucket per day of the cached window, so the
// demand sums stay the same whatever day the test runs on.
func writeHistory(t *testing.T, w http.ResponseWriter, r *http.Request, history map[string]int) {
	t.Helper()

	ids := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v2/stats/history/"), ",")
	today := startOfDay(time.Now())
	blocks := make([]string, 0, len(ids))
	for _, id := range ids {
		units, ok := history[id]
		if !ok {
			continue
		}
		points := make([]string, 0, demandHistoryDays)
		for day := demandHistoryDays - 1; day >= 0; day-- {
			stamp := today.AddDate(0, 0, -day).Format("2006-01-02T15:04:05")
			points = append(points, fmt.Sprintf(
				`{"item_count":%d,"avg_price":1000,"timestamp":%q}`, units, stamp))
		}
		blocks = append(blocks, fmt.Sprintf(
			`{"location":"Caerleon","item_id":%q,"quality":1,"data":[%s]}`,
			id, strings.Join(points, ",")))
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = fmt.Fprintf(w, "[%s]", strings.Join(blocks, ","))
}

func (f *fakeMarket) client() *market.Client {
	return &market.Client{BaseURL: f.server.URL, HTTP: f.server.Client()}
}

func (f *fakeMarket) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests
}

// countPrices counts only /prices requests, which the history fetch would
// otherwise inflate.
func (f *fakeMarket) countPrices() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, path := range f.paths {
		if strings.HasPrefix(path, "/api/v2/stats/prices/") {
			count++
		}
	}
	return count
}

func (f *fakeMarket) setFailing(failing bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failing = failing
}

// newTestCaches points both market caches at one fake server.
func newTestCaches(fake *fakeMarket) (*priceCache, *demandCache) {
	client := fake.client()
	return newPriceCache(client, "Caerleon", time.Minute),
		newDemandCache(client, "Caerleon", 1, time.Minute)
}

// setHistory sets how many units of each item trade per day.
func (f *fakeMarket) setHistory(history map[string]int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.history = history
}

func TestPriceCacheServesFromMemory(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{"T4_METALBAR": 1200})
	cache, _ := newTestCaches(fake)

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
	cache, _ := newTestCaches(fake)

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
	cache, _ := newTestCaches(fake)

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
	cache, _ := newTestCaches(fake)

	if rows := cache.Lookup(context.Background(), []string{"T4_METALBAR"}); len(rows) != 0 {
		t.Errorf("rows = %v, want none", rows)
	}
}

func TestBuildPlanRowsAndPrices(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{
		"T4_METALBAR": 1000,
		"T4_LEATHER":  500,
	})
	prices, demands := newTestCaches(fake)

	selected := &crafting.Item{
		UniqueName:     "T8_MAIN_SWORD",
		BaseUniqueName: "MAIN_SWORD",
		Label:          "Elder's Broadsword",
	}
	state := crafting.DefaultState()
	state.Item = selected.UniqueName

	// The fake market quotes T4 materials at 2026-11-03T02:10Z.
	now := time.Date(2026, 11, 3, 5, 10, 0, 0, time.UTC)
	table := buildPlan(context.Background(), prices, demands, state, selected, now)
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
	if first.Cells[0].AgeHours != "3h" {
		t.Errorf("cell 0 age = %q, want 3h", first.Cells[0].AgeHours)
	}
	if first.Cells[0].AgeClass != "age-0" {
		t.Errorf("cell 0 age class = %q, want age-0", first.Cells[0].AgeClass)
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
	prices, demands := newTestCaches(fake)

	selected := &crafting.Item{UniqueName: "T8_MAIN_SWORD", BaseUniqueName: "MAIN_SWORD"}
	state := crafting.DefaultState()
	state.Overrides = map[string]float64{"T4_METALBAR": 77}

	cell := buildPlan(context.Background(), prices, demands, state, selected, time.Now()).Rows[0].Cells[0]
	if !cell.HasOverride || cell.Override != 77 || cell.Price != 77 {
		t.Errorf("cell = %+v, want override 77", cell)
	}
	if cell.AgeHours != "" || cell.AgeClass != "" {
		t.Errorf("age = %q/%q, want none for a manual price", cell.AgeHours, cell.AgeClass)
	}
	if got := fake.countPrices(); got != 1 {
		t.Errorf("price requests = %d, want 1 (the other material still gets fetched)", got)
	}
}

func TestBuildPlanWithoutSelection(t *testing.T) {
	prices, demands := newTestCaches(newFakeMarket(t, nil))
	table := buildPlan(context.Background(), prices, demands, crafting.DefaultState(), nil, time.Now())
	if len(table.Rows) != 0 {
		t.Errorf("rows = %v, want none", table.Rows)
	}
	if len(table.Headers) != planIngredientColumns {
		t.Errorf("headers = %v, want %d blanks", table.Headers, planIngredientColumns)
	}
}

func TestPriceAge(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		seen  time.Time
		label string
		class string
	}{
		{now, "0h", "age-0"},
		{now.Add(-30 * time.Minute), "0h", "age-0"},
		{now.Add(-5 * time.Hour), "5h", "age-0"},
		{now.Add(-9 * time.Hour), "9h", "age-1"},
		{now.Add(-20 * time.Hour), "20h", "age-2"},
		{now.Add(-30 * time.Hour), "30h", "age-3"},
		{now.Add(-41 * time.Hour), "41h", "age-5"},
		{now.Add(-48 * time.Hour), "48h+", "age-5"},
		{now.Add(-30 * 24 * time.Hour), "48h+", "age-5"},
		{now.Add(2 * time.Hour), "0h", "age-0"}, // clock skew
		{time.Time{}, "", ""},                   // never traded
	}
	for _, tc := range cases {
		label, class := priceAge(now, tc.seen)
		if label != tc.label || class != tc.class {
			t.Errorf("priceAge(%s) = %q/%q, want %q/%q", now.Sub(tc.seen), label, class, tc.label, tc.class)
		}
	}
}

func TestBuildPlanMergesUnenchantedIngredients(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{})
	prices, demands := newTestCaches(fake)

	// 2H_ARCANESTAFF_CRYSTAL uses planks and bars per enchant, plus one
	// artefact that is the same item for enchant 0 through 4.
	selected := &crafting.Item{
		UniqueName:     "T8_2H_ARCANESTAFF_CRYSTAL",
		BaseUniqueName: "2H_ARCANESTAFF_CRYSTAL",
	}
	table := buildPlan(context.Background(), prices, demands, crafting.DefaultState(), selected, time.Now())

	tier := make([]PlanRow, 0, len(crafting.PlanEnchants))
	for _, row := range table.Rows {
		if row.Tier != crafting.PlanTiers[0] {
			continue
		}
		tier = append(tier, row)
	}
	if len(tier) != len(crafting.PlanEnchants) {
		t.Fatalf("tier rows = %d, want %d", len(tier), len(crafting.PlanEnchants))
	}

	artefact := tier[0].Cells[sharedIngredientColumn]
	if artefact.ItemID != "T4_ARTEFACT_2H_ARCANESTAFF_CRYSTAL" {
		t.Errorf("artefact cell = %q", artefact.ItemID)
	}
	if artefact.Rowspan != len(tier) {
		t.Errorf("artefact rowspan = %d, want %d", artefact.Rowspan, len(tier))
	}

	for i, row := range tier[1:] {
		if len(row.Cells) != planIngredientColumns-1 {
			t.Errorf("row %d cells = %d, want %d", i+1, len(row.Cells), planIngredientColumns-1)
		}
	}

	// The fourth column holds no material, so it is never merged.
	if got := tier[0].Cells[planIngredientColumns-1].Rowspan; got != 0 {
		t.Errorf("empty fourth column rowspan = %d, want 0", got)
	}

	// The first two columns hold enchanted materials and stay per row.
	for i, row := range tier[1:] {
		if row.Cells[0].ItemID != "T4_PLANKS_LEVEL"+strconv.Itoa(i+1) {
			t.Errorf("row %d first material = %q", i+1, row.Cells[0].ItemID)
		}
	}
}

func TestBuildPlanKeepsRowsWithoutEnchantedMaterialsMergedPerTier(t *testing.T) {
	fake := newFakeMarket(t, map[string]int{})
	prices, demands := newTestCaches(fake)

	// A two-ingredient recipe has nothing to merge.
	selected := &crafting.Item{UniqueName: "T8_MAIN_SWORD", BaseUniqueName: "MAIN_SWORD"}
	table := buildPlan(context.Background(), prices, demands, crafting.DefaultState(), selected, time.Now())

	for _, row := range table.Rows {
		for column, cell := range row.Cells {
			if cell.Rowspan != 0 {
				t.Errorf("tier %d column %d rowspan = %d, want 0", row.Tier, column, cell.Rowspan)
			}
		}
		if len(row.Cells) != planIngredientColumns {
			t.Errorf("tier %d cells = %d, want %d", row.Tier, len(row.Cells), planIngredientColumns)
		}
	}
}

// fakeWithHistory builds a fake market that only carries trade history.
func fakeWithHistory(t *testing.T, history map[string]int) *fakeMarket {
	t.Helper()
	fake := newFakeMarket(t, nil)
	fake.setHistory(history)
	return fake
}

// buildSwordPlan renders the T8 broadsword plan against prices for its T4
// materials and its T4 output.
func buildSwordPlan(t *testing.T, extra map[string]int, fee float64) PlanTable {
	t.Helper()

	prices := map[string]int{
		"T4_METALBAR":   1000,
		"T4_LEATHER":    500,
		"T4_MAIN_SWORD": 30000,
	}
	for id, price := range extra {
		prices[id] = price
	}

	fake := newFakeMarket(t, prices)
	fake.setHistory(map[string]int{"T4_MAIN_SWORD": 12})
	state := crafting.DefaultState()
	state.StationFee = fee

	selected := &crafting.Item{
		UniqueName:     "T8_MAIN_SWORD",
		BaseUniqueName: "MAIN_SWORD",
		Label:          "Elder's Broadsword",
	}
	now := time.Date(2026, 11, 3, 5, 10, 0, 0, time.UTC)
	client := fake.client()
	table := buildPlan(context.Background(), newPriceCache(client, "Caerleon", time.Minute),
		newDemandCache(client, "Caerleon", 1, time.Minute), state, selected, now)
	if len(table.Rows) == 0 {
		t.Fatal("no plan rows")
	}
	return table
}

func TestBuildPlanOutputPriceAndMargin(t *testing.T) {
	row := buildSwordPlan(t, nil, 0).Rows[0]

	if row.Output.ItemID != "T4_MAIN_SWORD" {
		t.Errorf("output = %q, want T4_MAIN_SWORD", row.Output.ItemID)
	}
	if !row.Output.HasPrice || row.Output.Price != 30000 {
		t.Errorf("output price = %+v, want 30000", row.Output)
	}
	if row.Output.AgeHours != "3h" {
		t.Errorf("output age = %q, want 3h", row.Output.AgeHours)
	}

	econ := row.Economics
	if !econ.HasCost || econ.Cost != 20000 {
		t.Errorf("cost = %v (has %v), want 20000", econ.Cost, econ.HasCost)
	}
	if !econ.HasRevenue || econ.Revenue != 30000 {
		t.Errorf("revenue = %v (has %v), want 30000", econ.Revenue, econ.HasRevenue)
	}
	if !econ.HasProfit() || econ.Profit != 10000 {
		t.Errorf("profit = %v, want 10000", econ.Profit)
	}
	if !econ.HasGain() || econ.Gain != 50 {
		t.Errorf("gain = %v, want 50", econ.Gain)
	}
}

func TestBuildPlanStationFeeIsPartOfCost(t *testing.T) {
	row := buildSwordPlan(t, nil, 500).Rows[0]

	if row.Economics.Cost != 20500 {
		t.Errorf("cost = %v, want 20500", row.Economics.Cost)
	}
	if row.Economics.Profit != 9500 {
		t.Errorf("profit = %v, want 9500", row.Economics.Profit)
	}
}

func TestBuildPlanMarginIsBlankWithMissingPrices(t *testing.T) {
	// The enchanted rows need level materials the fake market does not price.
	row := buildSwordPlan(t, nil, 0).Rows[1]

	if row.Economics.HasCost {
		t.Error("cost reported as complete without every ingredient priced")
	}
	if row.Economics.HasProfit() || row.Economics.HasGain() {
		t.Error("margin reported without a complete cost")
	}
}

func TestBuildPlanOutputOverrideWinsOverQuote(t *testing.T) {
	state := crafting.DefaultState()
	state.Overrides = map[string]float64{"T4_MAIN_SWORD": 40000}

	selected := &crafting.Item{
		UniqueName:     "T8_MAIN_SWORD",
		BaseUniqueName: "MAIN_SWORD",
		Label:          "Elder's Broadsword",
	}
	fake := newFakeMarket(t, map[string]int{"T4_METALBAR": 1000, "T4_LEATHER": 500, "T4_MAIN_SWORD": 30000})
	prices, demands := newTestCaches(fake)
	row := buildPlan(context.Background(), prices, demands, state, selected, time.Now()).Rows[0]

	if !row.Output.HasOverride || row.Output.Override != 40000 {
		t.Errorf("output override = %+v, want 40000", row.Output)
	}
	if row.Output.AgeHours != "" {
		t.Errorf("output age = %q, want none for a manual price", row.Output.AgeHours)
	}
	if row.Economics.Revenue != 40000 {
		t.Errorf("revenue = %v, want the override 40000", row.Economics.Revenue)
	}
}

func TestBuildPlanDemandAndSilverPerDay(t *testing.T) {
	row := buildSwordPlan(t, nil, 0).Rows[0]

	if !row.HasDemand || row.DemandUnits != 84 {
		t.Errorf("demand = %d (has %v), want 84 units over 7 days", row.DemandUnits, row.HasDemand)
	}
	if !row.HasBought || row.BoughtPerDay != 360000 {
		t.Errorf("bought per day = %v (has %v), want 360000", row.BoughtPerDay, row.HasBought)
	}
}

func TestDemandCacheReusesHistoryAcrossWindows(t *testing.T) {
	fake := fakeWithHistory(t, map[string]int{"T4_MAIN_SWORD": 12})
	cache := newDemandCache(fake.client(), "Caerleon", 1, time.Minute)

	week := cache.Lookup(context.Background(), []string{"T4_MAIN_SWORD"}, 7)
	month := cache.Lookup(context.Background(), []string{"T4_MAIN_SWORD"}, 30)

	if got := week["T4_MAIN_SWORD"]; got != 84 {
		t.Errorf("week units = %d, want 84", got)
	}
	if got := month["T4_MAIN_SWORD"]; got != 360 {
		t.Errorf("month units = %d, want 360", got)
	}
	if got := fake.count(); got != 1 {
		t.Errorf("requests = %d, want 1 (one history window serves every window)", got)
	}
}

func TestDemandCacheDoesNotRefetchItemsWithoutHistory(t *testing.T) {
	fake := fakeWithHistory(t, map[string]int{"T4_MAIN_SWORD": 3})
	cache := newDemandCache(fake.client(), "Caerleon", 1, time.Minute)

	for i := 0; i < 3; i++ {
		cache.Lookup(context.Background(), []string{"T4_MAIN_SWORD", "T8_MAIN_SWORD"}, 7)
	}

	if got := fake.count(); got != 1 {
		t.Errorf("requests = %d, want 1 (an item with no history is cached too)", got)
	}
	units := cache.Lookup(context.Background(), []string{"T8_MAIN_SWORD"}, 7)
	if got := units["T8_MAIN_SWORD"]; got != 0 {
		t.Errorf("units = %d, want 0", got)
	}
}
