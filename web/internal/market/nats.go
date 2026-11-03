package market

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Topics published by the Albion Data Project. The deduped variants are the
// ones to consume: the API promises the same message at most once every 5
// minutes, while ingest topics carry every upload including duplicates.
const (
	TopicMarketOrders    = "marketorders.deduped"
	TopicMarketHistories = "markethistories.deduped"
	TopicGoldPrices      = "goldprices.deduped"
)

// Auction type values used by the order feed.
const (
	AuctionOffer   = "offer"   // sell order
	AuctionRequest = "request" // buy order
)

// Timescale values used by the history feed.
const (
	TimescaleHours = 0 // 24 hour view
	TimescaleDays  = 1 // 7 day view
	TimescaleWeeks = 2 // 4 week view
)

const (
	// DefaultStaleAfter is how long an order may go unseen before the feed
	// drops it. Orders are re-sent every 5 minutes while they still exist, so
	// an order that stays silent for hours has almost certainly been filled.
	DefaultStaleAfter = 24 * time.Hour

	pruneInterval = 5 * time.Minute

	// .NET ticks are 100ns intervals since 0001-01-01.
	ticksPerSecond = 10_000_000
	// ticksToUnix is the number of seconds between 0001-01-01 and 1970-01-01.
	ticksToUnix = 62_135_596_800

	expiresMillisLayout = "2006-01-02T15:04:05.999999"
	expiresLayout       = "2006-01-02T15:04:05"
)

// NATS broker ports per region. The HTTP API splits the world into east, west,
// europe and asia, but the feed only has three brokers.
var natsPorts = map[string]int{
	Americas: 4222,
	Asia:     24222,
	Europe:   34222,
}

// Americas is the NATS region serving both the east and west HTTP APIs.
const Americas = "americas"

// NATSURL returns the public feed URL for a region. Unknown regions, and the
// HTTP-only East and West regions, fall back to the Americas broker.
func NATSURL(region string) string {
	port, ok := natsPorts[region]
	if !ok {
		port = natsPorts[Americas]
	}
	return "nats://public:thenewalbiondata@nats.albion-online-data.com:" + strconv.Itoa(port)
}

// LocationID is a location from the feed. The deduped feed sends numeric ids
// (3003) while the ingest tooling sometimes sends names ("Caerleon"), so both
// are accepted and kept in their original string form.
type LocationID string

// UnmarshalJSON accepts a JSON number or string.
func (l *LocationID) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var name string
		if err := json.Unmarshal(data, &name); err != nil {
			return err
		}
		*l = LocationID(name)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return err
	}
	*l = LocationID(number.String())
	return nil
}

// Name resolves the location to its canonical city name when the id is known,
// otherwise it returns the raw value.
func (l LocationID) Name() string {
	id, err := strconv.Atoi(string(l))
	if err != nil {
		return string(l)
	}
	for _, location := range CommonLocations {
		if location.ID == id {
			return location.Name
		}
	}
	return string(l)
}

// Order is one live market order from TopicMarketOrders. Prices are in silver.
type Order struct {
	ID               int64      `json:"Id"`
	ItemID           string     `json:"ItemTypeId"`
	GroupTypeID      string     `json:"ItemGroupTypeId"`
	LocationID       LocationID `json:"LocationId"`
	QualityLevel     int        `json:"QualityLevel"`
	EnchantmentLevel int        `json:"EnchantmentLevel"`
	Price            int64      `json:"UnitPriceSilver"`
	Amount           int        `json:"Amount"`
	AuctionType      string     `json:"AuctionType"`
	Expires          string     `json:"Expires"`
}

// ExpiresAt parses the Expires field, which has no zone and an optional
// fractional second.
func (o Order) ExpiresAt() (time.Time, error) {
	for _, layout := range []string{expiresMillisLayout, expiresLayout} {
		if parsed, err := time.Parse(layout, o.Expires); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("market: parse order expiry %q", o.Expires)
}

// Ticks is a .NET timestamp (100ns intervals since 0001-01-01) as sent by the
// history feed.
type Ticks uint64

// Time converts the tick count to UTC.
func (t Ticks) Time() time.Time {
	seconds := int64(t/ticksPerSecond) - ticksToUnix
	nanos := int64(t%ticksPerSecond) * 100
	return time.Unix(seconds, nanos).UTC()
}

// HistoryEntry is one aggregated bucket from TopicMarketHistories.
type HistoryEntry struct {
	ItemAmount   int64  `json:"ItemAmount"`
	SilverAmount uint64 `json:"SilverAmount"`
	Timestamp    Ticks  `json:"Timestamp"`
}

// AveragePrice is the total silver divided by the amount sold.
func (e HistoryEntry) AveragePrice() int64 {
	if e.ItemAmount <= 0 {
		return 0
	}
	return int64(e.SilverAmount) / e.ItemAmount
}

// HistoryUpload is one message from TopicMarketHistories: the history of a
// single item at a single location, quality and timescale.
type HistoryUpload struct {
	AlbionID      int32          `json:"AlbionId"`
	ItemID        string         `json:"AlbionIdString"`
	LocationID    LocationID     `json:"LocationId"`
	QualityLevel  int            `json:"QualityLevel"`
	Timescale     int            `json:"Timescale"`
	MarketHistory []HistoryEntry `json:"MarketHistories"`
}

type orderEntry struct {
	order    Order
	lastSeen time.Time
}

type historyKey struct {
	itemID    string
	location  LocationID
	quality   int
	timescale int
}

// Feed is an in-memory view of the live feed, so the app can answer price
// questions without polling the HTTP API.
type Feed struct {
	staleAfter time.Duration
	clock      func() time.Time

	mu        sync.RWMutex
	orders    map[int64]orderEntry
	histories map[historyKey]HistoryUpload
}

func newFeed(staleAfter time.Duration) *Feed {
	return &Feed{
		staleAfter: staleAfter,
		clock:      time.Now,
		orders:     make(map[int64]orderEntry),
		histories:  make(map[historyKey]HistoryUpload),
	}
}

// Orders returns a copy of the live orders, sorted by item, location, auction
// type and price so the result is stable.
func (f *Feed) Orders() []Order {
	f.mu.RLock()
	defer f.mu.RUnlock()

	orders := make([]Order, 0, len(f.orders))
	for _, entry := range f.orders {
		orders = append(orders, entry.order)
	}
	sort.Slice(orders, func(i, j int) bool {
		a, b := orders[i], orders[j]
		switch {
		case a.ItemID != b.ItemID:
			return a.ItemID < b.ItemID
		case a.LocationID != b.LocationID:
			return a.LocationID < b.LocationID
		case a.AuctionType != b.AuctionType:
			return a.AuctionType < b.AuctionType
		}
		return a.Price < b.Price
	})
	return orders
}

// Histories returns a copy of the latest history upload per item, location,
// quality and timescale.
func (f *Feed) Histories() []HistoryUpload {
	f.mu.RLock()
	defer f.mu.RUnlock()

	histories := make([]HistoryUpload, 0, len(f.histories))
	for _, upload := range f.histories {
		histories = append(histories, upload)
	}
	sort.Slice(histories, func(i, j int) bool {
		a, b := histories[i], histories[j]
		switch {
		case a.ItemID != b.ItemID:
			return a.ItemID < b.ItemID
		case a.LocationID != b.LocationID:
			return a.LocationID < b.LocationID
		case a.QualityLevel != b.QualityLevel:
			return a.QualityLevel < b.QualityLevel
		}
		return a.Timescale < b.Timescale
	})
	return histories
}

// OrderCount reports how many live orders are held.
func (f *Feed) OrderCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.orders)
}

// HistoryCount reports how many history blocks are held.
func (f *Feed) HistoryCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.histories)
}

func (f *Feed) applyOrder(order Order) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.orders[order.ID] = orderEntry{order: order, lastSeen: f.clock()}
}

func (f *Feed) applyHistory(upload HistoryUpload) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.histories[historyKey{
		itemID:    upload.ItemID,
		location:  upload.LocationID,
		quality:   upload.QualityLevel,
		timescale: upload.Timescale,
	}] = upload
}

// prune drops orders that have not been seen within staleAfter.
func (f *Feed) prune() int {
	cutoff := f.clock().Add(-f.staleAfter)

	f.mu.Lock()
	defer f.mu.Unlock()

	dropped := 0
	for id, entry := range f.orders {
		if entry.lastSeen.Before(cutoff) {
			delete(f.orders, id)
			dropped++
		}
	}
	return dropped
}

// SubscribeOptions configures Subscribe.
type SubscribeOptions struct {
	// URL is a broker URL such as NATSURL(Asia).
	URL string
	// StaleAfter overrides DefaultStaleAfter.
	StaleAfter time.Duration
	// OnError receives decode and connection errors. Errors are dropped when
	// OnError is nil.
	OnError func(error)
}

// Subscribe connects to the feed and keeps a Feed up to date until ctx is
// cancelled. It returns as soon as the connection is established and the
// subscriptions are registered.
func Subscribe(ctx context.Context, opts SubscribeOptions) (*Feed, error) {
	if opts.URL == "" {
		return nil, fmt.Errorf("market: no NATS url")
	}
	if opts.StaleAfter <= 0 {
		opts.StaleAfter = DefaultStaleAfter
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	report := func(err error) {
		if opts.OnError != nil {
			opts.OnError(err)
		}
	}

	conn, err := nats.Connect(opts.URL,
		nats.Name("albion-optimizer"),
		nats.Timeout(10*time.Second),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			report(err)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("market: connect %s: %w", opts.URL, err)
	}

	feed := newFeed(opts.StaleAfter)

	if _, err := conn.Subscribe(TopicMarketOrders, func(msg *nats.Msg) {
		orders, err := decodeOrders(msg.Data)
		if err != nil {
			report(err)
			return
		}
		for _, order := range orders {
			feed.applyOrder(order)
		}
	}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("market: subscribe %s: %w", TopicMarketOrders, err)
	}

	if _, err := conn.Subscribe(TopicMarketHistories, func(msg *nats.Msg) {
		upload, err := decodeHistory(msg.Data)
		if err != nil {
			report(err)
			return
		}
		feed.applyHistory(upload)
	}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("market: subscribe %s: %w", TopicMarketHistories, err)
	}

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	go func() {
		ticker := time.NewTicker(pruneInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				feed.prune()
			}
		}
	}()

	return feed, nil
}

// decodeOrders accepts either a single order, which is what the deduped topic
// publishes, or a {"Orders": [...]} upload batch.
func decodeOrders(data []byte) ([]Order, error) {
	var batch struct {
		Orders []Order `json:"Orders"`
	}
	if err := json.Unmarshal(data, &batch); err == nil && len(batch.Orders) > 0 {
		return batch.Orders, nil
	}

	var order Order
	if err := json.Unmarshal(data, &order); err != nil {
		return nil, fmt.Errorf("market: decode order: %w", err)
	}
	if order.ItemID == "" || order.ID == 0 {
		return nil, fmt.Errorf("market: decode order: missing Id or ItemTypeId")
	}
	return []Order{order}, nil
}

func decodeHistory(data []byte) (HistoryUpload, error) {
	var upload HistoryUpload
	if err := json.Unmarshal(data, &upload); err != nil {
		return HistoryUpload{}, fmt.Errorf("market: decode history: %w", err)
	}
	if upload.ItemID == "" {
		return HistoryUpload{}, fmt.Errorf("market: decode history: missing AlbionIdString")
	}
	return upload, nil
}
