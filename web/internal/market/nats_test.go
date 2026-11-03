package market

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

// Captured from marketorders.deduped on the Asia broker.
const orderFixture = `{"Amount":41,"AuctionType":"request","EnchantmentLevel":0,` +
	`"Expires":"3026-03-06T06:28:46.935876","Id":9649105993,` +
	`"ItemGroupTypeId":"T6_CAPE","ItemTypeId":"T6_CAPE","LocationId":3003,` +
	`"QualityLevel":2,"UnitPriceSilver":23068}`

// Same topic, with a named location and a sell order.
const offerFixture = `{"Amount":10,"AuctionType":"offer","EnchantmentLevel":1,` +
	`"Expires":"3026-03-06T06:28:46","Id":9649105994,` +
	`"ItemGroupTypeId":"T5_BAG","ItemTypeId":"T5_BAG","LocationId":"Caerleon",` +
	`"QualityLevel":1,"UnitPriceSilver":1234}`

// Captured from markethistories.deduped on the Asia broker.
const historyNATSFixture = `{"AlbionId":2216,"AlbionIdString":"T6_OFF_BOOK","LocationId":3003,` +
	`"QualityLevel":2,"Timescale":0,"MarketHistories":[` +
	`{"ItemAmount":28,"SilverAmount":787657,"Timestamp":639292680000000000},` +
	`{"ItemAmount":4,"SilverAmount":99996,"Timestamp":639292716000000000}]}`

func TestDecodeOrder(t *testing.T) {
	orders, err := decodeOrders([]byte(orderFixture))
	if err != nil {
		t.Fatalf("decodeOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("orders = %d, want 1", len(orders))
	}

	order := orders[0]
	if order.ID != 9649105993 {
		t.Errorf("ID = %d, want 9649105993", order.ID)
	}
	if order.ItemID != "T6_CAPE" || order.GroupTypeID != "T6_CAPE" {
		t.Errorf("item = %q/%q, want T6_CAPE", order.ItemID, order.GroupTypeID)
	}
	if order.LocationID != "3003" {
		t.Errorf("LocationID = %q, want 3003", order.LocationID)
	}
	if got := order.LocationID.Name(); got != "Black Market" {
		t.Errorf("LocationID.Name() = %q, want Black Market", got)
	}
	if order.QualityLevel != 2 || order.EnchantmentLevel != 0 {
		t.Errorf("quality/enchant = %d/%d, want 2/0", order.QualityLevel, order.EnchantmentLevel)
	}
	if order.Price != 23068 || order.Amount != 41 {
		t.Errorf("price/amount = %d/%d, want 23068/41", order.Price, order.Amount)
	}
	if order.AuctionType != AuctionRequest {
		t.Errorf("AuctionType = %q, want request", order.AuctionType)
	}

	expires, err := order.ExpiresAt()
	if err != nil {
		t.Fatalf("ExpiresAt: %v", err)
	}
	want := time.Date(3026, 3, 6, 6, 28, 46, 935876000, time.UTC)
	if !expires.Equal(want) {
		t.Errorf("ExpiresAt = %s, want %s", expires, want)
	}
}

func TestDecodeOrderNamedLocation(t *testing.T) {
	orders, err := decodeOrders([]byte(offerFixture))
	if err != nil {
		t.Fatalf("decodeOrders: %v", err)
	}
	if orders[0].LocationID != "Caerleon" {
		t.Errorf("LocationID = %q, want Caerleon", orders[0].LocationID)
	}
	if got := orders[0].LocationID.Name(); got != "Caerleon" {
		t.Errorf("LocationID.Name() = %q, want Caerleon", got)
	}
	if orders[0].AuctionType != AuctionOffer {
		t.Errorf("AuctionType = %q, want offer", orders[0].AuctionType)
	}
}

func TestDecodeOrderBatch(t *testing.T) {
	batch := `{"Orders":[` + orderFixture + `,` + offerFixture + `]}`
	orders, err := decodeOrders([]byte(batch))
	if err != nil {
		t.Fatalf("decodeOrders: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("orders = %d, want 2", len(orders))
	}
}

func TestDecodeOrderRejectsGarbage(t *testing.T) {
	if _, err := decodeOrders([]byte(`{"Amount":1}`)); err == nil {
		t.Error("expected error for order without Id and ItemTypeId")
	}
	if _, err := decodeOrders([]byte(`not json`)); err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestDecodeHistoryFeed(t *testing.T) {
	upload, err := decodeHistory([]byte(historyNATSFixture))
	if err != nil {
		t.Fatalf("decodeHistory: %v", err)
	}
	if upload.ItemID != "T6_OFF_BOOK" || upload.AlbionID != 2216 {
		t.Errorf("item = %q/%d, want T6_OFF_BOOK/2216", upload.ItemID, upload.AlbionID)
	}
	if upload.QualityLevel != 2 || upload.Timescale != TimescaleHours {
		t.Errorf("quality/timescale = %d/%d, want 2/0", upload.QualityLevel, upload.Timescale)
	}
	if len(upload.MarketHistory) != 2 {
		t.Fatalf("entries = %d, want 2", len(upload.MarketHistory))
	}

	entry := upload.MarketHistory[0]
	want := time.Date(2026, 11, 3, 2, 0, 0, 0, time.UTC)
	if got := entry.Timestamp.Time(); !got.Equal(want) {
		t.Errorf("timestamp = %s, want %s", got, want)
	}
	if got := entry.AveragePrice(); got != 28130 {
		t.Errorf("AveragePrice = %d, want 28130", got)
	}
	if got := upload.MarketHistory[1].AveragePrice(); got != 24999 {
		t.Errorf("AveragePrice = %d, want 24999", got)
	}
}

func TestLocationIDName(t *testing.T) {
	cases := map[LocationID]string{
		"3003":      "Black Market",
		"7":         "Thetford",
		"4002":      "Fort Sterling",
		"Caerleon":  "Caerleon",
		"unknown-1": "unknown-1",
	}
	for in, want := range cases {
		if got := in.Name(); got != want {
			t.Errorf("LocationID(%q).Name() = %q, want %q", in, got, want)
		}
	}
}

func TestNATSURL(t *testing.T) {
	cases := map[string]string{
		Americas: "nats://public:thenewalbiondata@nats.albion-online-data.com:4222",
		Asia:     "nats://public:thenewalbiondata@nats.albion-online-data.com:24222",
		Europe:   "nats://public:thenewalbiondata@nats.albion-online-data.com:34222",
		East:     "nats://public:thenewalbiondata@nats.albion-online-data.com:4222",
		West:     "nats://public:thenewalbiondata@nats.albion-online-data.com:4222",
		"":       "nats://public:thenewalbiondata@nats.albion-online-data.com:4222",
	}
	for region, want := range cases {
		if got := NATSURL(region); got != want {
			t.Errorf("NATSURL(%q) = %q, want %q", region, got, want)
		}
	}
}

func TestFeedUpsertsAndPrunesOrders(t *testing.T) {
	now := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	feed := newFeed(24 * time.Hour)
	feed.clock = func() time.Time { return now }

	orders, err := decodeOrders([]byte(orderFixture))
	if err != nil {
		t.Fatalf("decodeOrders: %v", err)
	}
	feed.applyOrder(orders[0])

	// The same order id arriving again must replace the row, not duplicate it.
	refreshed := orders[0]
	refreshed.Price = 21000
	feed.applyOrder(refreshed)

	if got := feed.OrderCount(); got != 1 {
		t.Fatalf("OrderCount = %d, want 1", got)
	}
	if got := feed.Orders()[0].Price; got != 21000 {
		t.Errorf("price = %d, want 21000", got)
	}

	// Still fresh: nothing to prune.
	if dropped := feed.prune(); dropped != 0 {
		t.Errorf("prune dropped %d fresh orders, want 0", dropped)
	}

	now = now.Add(25 * time.Hour)
	if dropped := feed.prune(); dropped != 1 {
		t.Errorf("prune dropped %d stale orders, want 1", dropped)
	}
	if got := feed.OrderCount(); got != 0 {
		t.Errorf("OrderCount = %d after prune, want 0", got)
	}
}

func TestFeedReplacesHistories(t *testing.T) {
	feed := newFeed(DefaultStaleAfter)

	upload, err := decodeHistory([]byte(historyNATSFixture))
	if err != nil {
		t.Fatalf("decodeHistory: %v", err)
	}
	feed.applyHistory(upload)
	feed.applyHistory(upload)

	other := upload
	other.QualityLevel = 3
	feed.applyHistory(other)

	if got := feed.HistoryCount(); got != 2 {
		t.Errorf("HistoryCount = %d, want 2 (one per quality)", got)
	}

	histories := feed.Histories()
	if histories[0].QualityLevel != 2 || histories[1].QualityLevel != 3 {
		t.Errorf("histories not sorted by quality: %+v", histories)
	}
}

func TestSubscribeRequiresURL(t *testing.T) {
	if _, err := Subscribe(context.Background(), SubscribeOptions{}); err == nil {
		t.Error("expected error without a URL")
	}
}

// TestSubscribeLive connects to the real broker and is skipped unless
// ALBION_NATS_LIVE is set.
func TestSubscribeLive(t *testing.T) {
	if os.Getenv("ALBION_NATS_LIVE") == "" {
		t.Skip("set ALBION_NATS_LIVE=1 to run against the live feed")
	}

	ctx, cancel := context.WithCancel(context.Background())

	var mu sync.Mutex
	var feedErrors []error
	feed, err := Subscribe(ctx, SubscribeOptions{
		URL: NATSURL(Asia),
		OnError: func(err error) {
			mu.Lock()
			feedErrors = append(feedErrors, err)
			mu.Unlock()
		},
	})
	if err != nil {
		cancel()
		t.Fatalf("Subscribe: %v", err)
	}
	defer func() {
		cancel()
		mu.Lock()
		defer mu.Unlock()
		for _, err := range feedErrors {
			t.Logf("feed error: %v", err)
		}
	}()

	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) && feed.OrderCount()+feed.HistoryCount() == 0 {
		time.Sleep(200 * time.Millisecond)
	}

	orders := feed.Orders()
	histories := feed.Histories()
	if len(orders) == 0 && len(histories) == 0 {
		t.Skip("feed was quiet for 60s; rerun when market traffic is flowing")
	}

	t.Logf("orders=%d histories=%d", len(orders), len(histories))
	if len(orders) > 0 {
		t.Logf("first order: %+v", orders[0])
		if orders[0].ItemID == "" || orders[0].Price <= 0 {
			t.Errorf("decoded order looks wrong: %+v", orders[0])
		}
	}
	if len(histories) > 0 {
		t.Logf("first history: %s %s q%d %d entries",
			histories[0].ItemID, histories[0].LocationID.Name(),
			histories[0].QualityLevel, len(histories[0].MarketHistory))
	}
}
