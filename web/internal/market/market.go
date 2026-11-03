// Package market reads market data from the Albion Online Data Project API
// (https://east.albion-online-data.com/api-docs).
//
// Wrapped endpoints:
//
//	GET /api/v2/stats/prices/{ids}   current prices, JSON
//	GET /api/v2/stats/view/{ids}     same rows as an HTML table fragment
//	GET /api/v2/stats/history/{ids}  aggregated history, JSON
//
// Observed API behaviour worth knowing:
//
//   - Timestamps carry no zone offset ("2026-11-03T02:10:00"). The zero value
//     "0001-01-01T00:00:00" means "no data", not a real trade.
//   - /view ignores format=json and a .json extension; it always answers with
//     text/html, and leaves cells empty where /prices would report 0.
//   - /history silently returns [] when time-scale is missing or not 1, 6 or
//     24, so Scale is validated before the request is made.
//   - time-scale=24 buckets are labelled one day before the requested start
//     date, and the newest daily bucket lags roughly a week behind real time.
//   - The locations parameter takes a numeric location ID or a name; spaces are
//     optional and case does not matter. Responses always use the canonical
//     name, so "3003", "blackmarket" and "Black Market" all come back as
//     "Black Market".
package market

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Regions supported by the API.
const (
	East   = "east"
	West   = "west"
	Europe = "europe"
	Asia   = "asia"
)

const (
	timestampLayout = "2006-01-02T15:04:05"
	dateLayout      = "2006-01-02"

	// maxBody caps a response so a misbehaving server cannot exhaust memory.
	maxBody = 8 << 20

	userAgent = "albion-optimizer/1.0 (+https://github.com/alayasan/albion-optimizer)"
)

// Location is a market hub. ID and Name are interchangeable in the locations
// parameter; use the ID when you want a value that never needs escaping.
type Location struct {
	ID   int
	Name string
}

// String returns the canonical name, which is what responses report back.
func (l Location) String() string {
	return l.Name
}

// CommonLocations are the eight hubs the API returns when Locations is empty.
var CommonLocations = []Location{
	{ID: 3003, Name: "Black Market"},
	{ID: 2004, Name: "Bridgewatch"},
	{ID: 5003, Name: "Brecilien"},
	{ID: 3005, Name: "Caerleon"},
	{ID: 4002, Name: "Fort Sterling"},
	{ID: 1002, Name: "Lymhurst"},
	{ID: 3008, Name: "Martlock"},
	{ID: 7, Name: "Thetford"},
}

var (
	// ErrNoItems is returned when a request carries no item ids.
	ErrNoItems = errors.New("market: no item ids")

	// ErrScale is returned by History when Scale is not 1, 6 or 24 hours,
	// because the API answers such requests with an empty list.
	ErrScale = errors.New("market: history time scale must be 1, 6 or 24 hours")
)

// Timestamp is an API timestamp. The wire format has no zone offset, so values
// are parsed as UTC. A zero Timestamp is the API's "never traded" sentinel.
type Timestamp struct {
	time.Time
}

// UnmarshalJSON parses the API timestamp layout.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if raw == "" {
		t.Time = time.Time{}
		return nil
	}
	parsed, err := time.Parse(timestampLayout, raw)
	if err != nil {
		return fmt.Errorf("market: parse timestamp %q: %w", raw, err)
	}
	t.Time = parsed
	return nil
}

// Price is one row of GET /api/v2/stats/prices: an item at one city and
// quality. Prices are in silver.
type Price struct {
	ItemID           string    `json:"item_id"`
	City             string    `json:"city"`
	Quality          int       `json:"quality"`
	SellPriceMin     int64     `json:"sell_price_min"`
	SellPriceMinDate Timestamp `json:"sell_price_min_date"`
	SellPriceMax     int64     `json:"sell_price_max"`
	SellPriceMaxDate Timestamp `json:"sell_price_max_date"`
	BuyPriceMin      int64     `json:"buy_price_min"`
	BuyPriceMinDate  Timestamp `json:"buy_price_min_date"`
	BuyPriceMax      int64     `json:"buy_price_max"`
	BuyPriceMaxDate  Timestamp `json:"buy_price_max_date"`
}

// History is one item/location block of GET /api/v2/stats/history.
type History struct {
	ItemID   string         `json:"item_id"`
	Location string         `json:"location"`
	Quality  int            `json:"quality"`
	Data     []HistoryPoint `json:"data"`
}

// HistoryPoint is one aggregated bucket inside a History block.
type HistoryPoint struct {
	ItemCount int       `json:"item_count"`
	AvgPrice  int64     `json:"avg_price"`
	Timestamp Timestamp `json:"timestamp"`
}

// PricesQuery filters /prices and /view requests. An empty Locations slice
// means CommonLocations, and an empty Qualities slice means qualities 1-5.
type PricesQuery struct {
	Locations []string
	Qualities []int
}

// HistoryQuery filters /history requests.
//
// Scale is the bucket width in hours and is required by the API.
// Start and End are inclusive days; zero values fall back to the API defaults
// (30 days back, then 30 days after the start).
type HistoryQuery struct {
	Locations []string
	Qualities []int
	Start     time.Time
	End       time.Time
	Scale     int
}

// Client queries one API region. The zero value is not usable; call New.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// httpHosts maps a region to the host that serves its HTTP API. Asia is
// published under the east host and the Americas under west; "asia" and
// "americas" only exist as NATS broker names.
var httpHosts = map[string]string{
	East:     "east",
	Asia:     "east",
	West:     "west",
	Americas: "west",
	Europe:   "europe",
}

// New returns a client for a region such as East. An empty or unknown region
// falls back to East.
func New(region string) *Client {
	host, ok := httpHosts[region]
	if !ok {
		host = httpHosts[East]
	}
	return &Client{
		BaseURL: "https://" + host + ".albion-online-data.com",
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// Prices returns current prices for the given item ids.
func (c *Client) Prices(ctx context.Context, ids []string, q PricesQuery) ([]Price, error) {
	body, err := c.get(ctx, "/api/v2/stats/prices/"+idPath(ids), queryValues(q.Locations, q.Qualities), "application/json")
	if err != nil {
		return nil, err
	}
	var prices []Price
	if err := json.Unmarshal(body, &prices); err != nil {
		return nil, fmt.Errorf("market: decode prices: %w", err)
	}
	return prices, nil
}

// View returns the /view response: an HTML table fragment with the same
// columns as Prices. The API only supports HTML here, so the value is meant
// for direct embedding, not parsing.
func (c *Client) View(ctx context.Context, ids []string, q PricesQuery) (string, error) {
	body, err := c.get(ctx, "/api/v2/stats/view/"+idPath(ids), queryValues(q.Locations, q.Qualities), "text/html")
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// History returns aggregated price history for the given item ids.
func (c *Client) History(ctx context.Context, ids []string, q HistoryQuery) ([]History, error) {
	switch q.Scale {
	case 1, 6, 24:
	default:
		return nil, ErrScale
	}
	if !q.Start.IsZero() && !q.End.IsZero() && q.End.Before(q.Start) {
		return nil, fmt.Errorf("market: end date %s is before start date %s",
			q.End.Format(dateLayout), q.Start.Format(dateLayout))
	}

	values := queryValues(q.Locations, q.Qualities)
	values.Set("time-scale", strconv.Itoa(q.Scale))
	if !q.Start.IsZero() {
		values.Set("date", q.Start.Format(dateLayout))
	}
	if !q.End.IsZero() {
		values.Set("end_date", q.End.Format(dateLayout))
	}

	body, err := c.get(ctx, "/api/v2/stats/history/"+idPath(ids), values, "application/json")
	if err != nil {
		return nil, err
	}
	var history []History
	if err := json.Unmarshal(body, &history); err != nil {
		return nil, fmt.Errorf("market: decode history: %w", err)
	}
	return history, nil
}

func idPath(ids []string) string {
	escaped := make([]string, len(ids))
	for i, id := range ids {
		escaped[i] = url.PathEscape(id)
	}
	return strings.Join(escaped, ",")
}

func queryValues(locations []string, qualities []int) url.Values {
	values := url.Values{}
	if len(locations) > 0 {
		values.Set("locations", strings.Join(locations, ","))
	}
	if len(qualities) > 0 {
		numbers := make([]string, len(qualities))
		for i, quality := range qualities {
			numbers[i] = strconv.Itoa(quality)
		}
		values.Set("qualities", strings.Join(numbers, ","))
	}
	return values
}

func (c *Client) get(ctx context.Context, path string, query url.Values, accept string) ([]byte, error) {
	if path == "" || strings.HasSuffix(path, "/") {
		return nil, ErrNoItems
	}

	endpoint := strings.TrimRight(c.BaseURL, "/") + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("market: build request: %w", err)
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("User-Agent", userAgent)

	client := c.HTTP
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("market: get %s: %w", endpoint, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, fmt.Errorf("market: read %s: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("market: %s: %s: %s",
			endpoint, resp.Status, strings.TrimSpace(string(body)))
	}
	return body, nil
}
