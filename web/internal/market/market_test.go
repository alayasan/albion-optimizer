package market

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Captured from GET /api/v2/stats/prices/T4_BAG?locations=Bridgewatch&qualities=1
const pricesFixture = `[{"item_id":"T4_BAG","city":"Bridgewatch","quality":1,` +
	`"sell_price_min":6639,"sell_price_min_date":"2026-11-03T02:10:00",` +
	`"sell_price_max":7758,"sell_price_max_date":"2026-11-03T02:10:00",` +
	`"buy_price_min":0,"buy_price_min_date":"0001-01-01T00:00:00",` +
	`"buy_price_max":0,"buy_price_max_date":"0001-01-01T00:00:00"}]`

// Captured from GET /api/v2/stats/history/T4_MAIN_SWORD?qualities=1&time-scale=24&date=2026-10-19&end_date=2026-10-20
const historyFixture = `[{"location":"Black Market","item_id":"T4_MAIN_SWORD","quality":1,"data":[` +
	`{"item_count":61,"avg_price":11876,"timestamp":"2026-10-19T00:00:00"},` +
	`{"item_count":204,"avg_price":11692,"timestamp":"2026-10-20T00:00:00"}]}]`

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &Client{BaseURL: server.URL, HTTP: server.Client()}, server
}

func cities(prices []Price) map[string]bool {
	got := make(map[string]bool, len(prices))
	for _, price := range prices {
		got[price.City] = true
	}
	return got
}

func TestPricesDecode(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(pricesFixture))
	})

	prices, err := client.Prices(context.Background(), []string{"T4_BAG"}, PricesQuery{
		Locations: []string{"Bridgewatch"},
		Qualities: []int{1},
	})
	if err != nil {
		t.Fatalf("Prices: %v", err)
	}
	if len(prices) != 1 {
		t.Fatalf("rows = %d, want 1", len(prices))
	}

	got := prices[0]
	if got.ItemID != "T4_BAG" {
		t.Errorf("ItemID = %q, want T4_BAG", got.ItemID)
	}
	if got.City != "Bridgewatch" {
		t.Errorf("City = %q, want Bridgewatch", got.City)
	}
	if got.Quality != 1 {
		t.Errorf("Quality = %d, want 1", got.Quality)
	}
	if got.SellPriceMin != 6639 || got.SellPriceMax != 7758 {
		t.Errorf("sell prices = %d/%d, want 6639/7758", got.SellPriceMin, got.SellPriceMax)
	}
	want := time.Date(2026, 11, 3, 2, 10, 0, 0, time.UTC)
	if !got.SellPriceMinDate.Equal(want) {
		t.Errorf("SellPriceMinDate = %s, want %s", got.SellPriceMinDate.Time, want)
	}
	if got.BuyPriceMin != 0 || !got.BuyPriceMinDate.IsZero() {
		t.Errorf("zero sentinel not preserved: price=%d date=%s",
			got.BuyPriceMin, got.BuyPriceMinDate.Time)
	}
}

func TestHistoryDecode(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(historyFixture))
	})

	history, err := client.History(context.Background(), []string{"T4_MAIN_SWORD"}, HistoryQuery{
		Qualities: []int{1},
		Start:     time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC),
		End:       time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		Scale:     24,
	})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("blocks = %d, want 1", len(history))
	}

	block := history[0]
	if block.ItemID != "T4_MAIN_SWORD" || block.Location != "Black Market" || block.Quality != 1 {
		t.Errorf("block = %+v", block)
	}
	if len(block.Data) != 2 {
		t.Fatalf("points = %d, want 2", len(block.Data))
	}
	if block.Data[0].ItemCount != 61 || block.Data[0].AvgPrice != 11876 {
		t.Errorf("first point = %+v", block.Data[0])
	}
	want := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	if !block.Data[1].Timestamp.Equal(want) {
		t.Errorf("last timestamp = %s, want %s", block.Data[1].Timestamp.Time, want)
	}
}

func TestPricesBuildsRequest(t *testing.T) {
	var gotPath, gotQuery, gotAccept string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotAccept = r.Header.Get("Accept")
		_, _ = w.Write([]byte("[]"))
	})

	if _, err := client.Prices(context.Background(), []string{"T4_BAG", "T4_MAIN_SWORD"}, PricesQuery{
		Locations: []string{"Bridgewatch", "Caerleon"},
		Qualities: []int{1, 2},
	}); err != nil {
		t.Fatalf("Prices: %v", err)
	}

	if want := "/api/v2/stats/prices/T4_BAG,T4_MAIN_SWORD"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	const wantQuery = "locations=Bridgewatch%2CCaerleon&qualities=1%2C2"
	if gotQuery != wantQuery {
		t.Errorf("query = %q, want %q", gotQuery, wantQuery)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}
}

func TestHistoryBuildsRequest(t *testing.T) {
	var gotQuery string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = w.Write([]byte("[]"))
	})

	if _, err := client.History(context.Background(), []string{"T4_MAIN_SWORD"}, HistoryQuery{
		Start: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		Scale: 24,
	}); err != nil {
		t.Fatalf("History: %v", err)
	}

	for _, want := range []string{
		"date=2026-10-20",
		"end_date=2026-10-21",
		"time-scale=24",
	} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}

func TestHistoryRejectsInvalidRange(t *testing.T) {
	client := &Client{BaseURL: "http://127.0.0.1:1"}

	if _, err := client.History(context.Background(), []string{"T4_BAG"}, HistoryQuery{Scale: 3}); !errors.Is(err, ErrScale) {
		t.Errorf("scale 3 error = %v, want ErrScale", err)
	}

	_, err := client.History(context.Background(), []string{"T4_BAG"}, HistoryQuery{
		Start: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		Scale: 24,
	})
	if err == nil || !strings.Contains(err.Error(), "before start date") {
		t.Errorf("reversed range error = %v, want before start date", err)
	}
}

func TestRequestsRejectEmptyIDs(t *testing.T) {
	client := &Client{BaseURL: "http://127.0.0.1:1"}

	if _, err := client.Prices(context.Background(), nil, PricesQuery{}); !errors.Is(err, ErrNoItems) {
		t.Errorf("Prices error = %v, want ErrNoItems", err)
	}
	if _, err := client.View(context.Background(), nil, PricesQuery{}); !errors.Is(err, ErrNoItems) {
		t.Errorf("View error = %v, want ErrNoItems", err)
	}
	if _, err := client.History(context.Background(), nil, HistoryQuery{Scale: 24}); !errors.Is(err, ErrNoItems) {
		t.Errorf("History error = %v, want ErrNoItems", err)
	}
}

func TestViewReturnsHTML(t *testing.T) {
	const table = `<head></head><body><table><tr><th>item_id</th></tr></table></body>`
	var gotAccept string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(table))
	})

	got, err := client.View(context.Background(), []string{"T4_BAG"}, PricesQuery{Qualities: []int{1}})
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	if got != table {
		t.Errorf("View body = %q, want %q", got, table)
	}
	if gotAccept != "text/html" {
		t.Errorf("Accept = %q, want text/html", gotAccept)
	}
}

func TestCommonLocations(t *testing.T) {
	want := []Location{
		{3003, "Black Market"},
		{2004, "Bridgewatch"},
		{5003, "Brecilien"},
		{3005, "Caerleon"},
		{4002, "Fort Sterling"},
		{1002, "Lymhurst"},
		{3008, "Martlock"},
		{7, "Thetford"},
	}
	if len(CommonLocations) != len(want) {
		t.Fatalf("CommonLocations = %d entries, want %d", len(CommonLocations), len(want))
	}

	seenID := map[int]bool{}
	seenName := map[string]bool{}
	for i, got := range CommonLocations {
		if got != want[i] {
			t.Errorf("CommonLocations[%d] = %+v, want %+v", i, got, want[i])
		}
		if got.ID == 0 || got.Name == "" {
			t.Errorf("CommonLocations[%d] is incomplete: %+v", i, got)
		}
		if got.String() != got.Name {
			t.Errorf("CommonLocations[%d].String() = %q, want %q", i, got.String(), got.Name)
		}
		if seenID[got.ID] {
			t.Errorf("duplicate location id %d", got.ID)
		}
		if seenName[got.Name] {
			t.Errorf("duplicate location name %q", got.Name)
		}
		seenID[got.ID] = true
		seenName[got.Name] = true
	}
}

func TestThrottleError(t *testing.T) {
	const throttle = "Throttled, be nice to the server, use compression (gzip, deflate, etc)!"
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(throttle))
	})

	_, err := client.Prices(context.Background(), []string{"T4_BAG"}, PricesQuery{})
	if err == nil {
		t.Fatal("expected throttle error")
	}
	if !strings.Contains(err.Error(), "429") || !strings.Contains(err.Error(), "Throttled") {
		t.Errorf("error = %q, want status 429 and body", err)
	}
}

// TestLive hits the real API and is skipped unless ALBION_API_LIVE is set.
func TestLive(t *testing.T) {
	if os.Getenv("ALBION_API_LIVE") == "" {
		t.Skip("set ALBION_API_LIVE=1 to run against the live API")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := New(East)

	prices, err := client.Prices(ctx, []string{"T4_BAG"}, PricesQuery{Qualities: []int{1, 2}})
	if err != nil {
		t.Fatalf("Prices: %v", err)
	}
	if len(prices) == 0 {
		t.Error("Prices returned no rows")
	}
	t.Logf("prices rows=%d first=%+v", len(prices), prices[0])

	// Location IDs and sloppy names must come back as canonical city names.
	byID, err := client.Prices(ctx, []string{"T4_BAG"}, PricesQuery{
		Locations: []string{"3003", "7"},
		Qualities: []int{1},
	})
	if err != nil {
		t.Fatalf("Prices by id: %v", err)
	}
	if got := cities(byID); len(got) != 2 || !got["Black Market"] || !got["Thetford"] {
		t.Errorf("prices by id cities = %v, want Black Market and Thetford", got)
	}

	byName, err := client.Prices(ctx, []string{"T4_BAG"}, PricesQuery{
		Locations: []string{"fortsterling"},
		Qualities: []int{1},
	})
	if err != nil {
		t.Fatalf("Prices by name: %v", err)
	}
	if got := cities(byName); len(got) != 1 || !got["Fort Sterling"] {
		t.Errorf("prices by name cities = %v, want Fort Sterling", got)
	}

	history, err := client.History(ctx, []string{"T4_MAIN_SWORD"}, HistoryQuery{
		Qualities: []int{1},
		Start:     time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC),
		End:       time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		Scale:     24,
	})
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(history) == 0 || len(history[0].Data) == 0 {
		t.Error("History returned no buckets")
	}

	view, err := client.View(ctx, []string{"T4_BAG"}, PricesQuery{Qualities: []int{1}})
	if err != nil {
		t.Fatalf("View: %v", err)
	}
	if !strings.Contains(view, "<table") {
		t.Errorf("View body is not a table: %q", view)
	}
}

func TestNewRegionHosts(t *testing.T) {
	cases := map[string]string{
		East:     "https://east.albion-online-data.com",
		Asia:     "https://east.albion-online-data.com",
		West:     "https://west.albion-online-data.com",
		Americas: "https://west.albion-online-data.com",
		Europe:   "https://europe.albion-online-data.com",
		"":       "https://east.albion-online-data.com",
		"bogus":  "https://east.albion-online-data.com",
	}
	for region, want := range cases {
		if got := New(region).BaseURL; got != want {
			t.Errorf("New(%q).BaseURL = %q, want %q", region, got, want)
		}
	}
}
