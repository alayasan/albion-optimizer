package crafting

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/alayasan/albion-optimizer/web/internal/market"
)

//go:embed data/items.json
var itemsJSON []byte

// Item is a craftable gear item, reduced to the highest available tier.
type Item struct {
	UniqueName     string `json:"uniqueName"`
	BaseUniqueName string `json:"baseUniqueName"`
	Label          string `json:"label"`
}

// Option is a value/label pair rendered by the UI controls.
type Option struct {
	Value string
	Label string
}

// overridePrefix is the form field prefix for manual unit prices.
const overridePrefix = "price["

// Form field prefixes for the per-material trade selections.
const (
	cityPrefix = "buyCity["
	modePrefix = "buyMode["
)

const (
	// DefaultCity is the hub prices are read in until a trade picks another.
	DefaultCity = "Bridgewatch"

	// ModeOrder keeps a trade in the order book: a sell order for the product,
	// a buy order for a material. ModeInstant crosses the spread and trades
	// against the offers already on the market.
	ModeOrder   = "order"
	ModeInstant = "instant"
)

var (
	items []Item

	Qualities = []Option{
		{Value: "1", Label: "Normal"},
		{Value: "2", Label: "Good"},
		{Value: "3", Label: "Outstanding"},
		{Value: "4", Label: "Excellent"},
		{Value: "5", Label: "Masterpiece"},
	}

	ZoneQualities = []Option{
		{Value: "1", Label: "1"},
		{Value: "2", Label: "2"},
		{Value: "3", Label: "3"},
		{Value: "4", Label: "4"},
		{Value: "5", Label: "5"},
		{Value: "6", Label: "6"},
	}

	HideoutPowers = []Option{
		{Value: "1", Label: "1"},
		{Value: "2", Label: "2"},
		{Value: "3", Label: "3"},
		{Value: "4", Label: "4"},
		{Value: "5", Label: "5"},
		{Value: "6", Label: "6"},
		{Value: "7", Label: "7"},
		{Value: "8", Label: "8"},
		{Value: "9", Label: "9"},
	}

	DailyBonuses = []Option{
		{Value: "none", Label: "None"},
		{Value: "10", Label: "10%"},
		{Value: "20", Label: "20%"},
	}

	Locations = []Option{
		{Value: "bonus-city", Label: "Bonus City"},
		{Value: "hideout", Label: "Hideout"},
		{Value: "island", Label: "Island"},
	}

	DemandWindows = []Option{
		{Value: "1", Label: "1 day"},
		{Value: "7", Label: "7 days"},
		{Value: "14", Label: "14 days"},
		{Value: "30", Label: "30 days"},
	}

	// Cities are the hubs an item can be priced in, in the order the market
	// package lists them.
	Cities = cityOptions()

	SellModes = []Option{
		{Value: ModeOrder, Label: "Sell order"},
		{Value: ModeInstant, Label: "Instant sell"},
	}

	BuyModes = []Option{
		{Value: ModeInstant, Label: "Instant buy"},
		{Value: ModeOrder, Label: "Buy order"},
	}

	tierPattern    = regexp.MustCompile(`^T(\d+)_`)
	levelPattern   = regexp.MustCompile(`_LEVEL\d+$`)
	enchantPattern = regexp.MustCompile(`@\d+$`)
)

func init() {
	if err := json.Unmarshal(itemsJSON, &items); err != nil {
		panic(fmt.Sprintf("crafting: parse embedded items: %v", err))
	}
	sortItems(items)
}

func sortItems(list []Item) {
	sort.Slice(list, func(i, j int) bool {
		return list[i].Label < list[j].Label
	})
}

// cityOptions mirrors the market hubs as UI options, so the selectable cities
// and the API locations cannot drift apart.
func cityOptions() []Option {
	options := make([]Option, len(market.CommonLocations))
	for i, location := range market.CommonLocations {
		options[i] = Option{Value: location.Name, Label: location.Name}
	}
	return options
}

// Items returns the embedded, sorted list of craftable items.
func Items() []Item {
	return items
}

// Lookup returns the item with the given unique name.
func Lookup(uniqueName string) (Item, bool) {
	for _, item := range items {
		if item.UniqueName == uniqueName {
			return item, true
		}
	}
	return Item{}, false
}

// Tier extracts the tier number from a unique name such as T8_MAIN_SWORD.
func Tier(uniqueName string) int {
	match := tierPattern.FindStringSubmatch(uniqueName)
	if len(match) != 2 {
		return 0
	}
	tier, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return tier
}

// State is the complete crafting configuration. It is persisted in a cookie.
type State struct {
	Item         string  `json:"item"`
	Quality      int     `json:"quality"`
	Location     string  `json:"location"`
	Quantity     int     `json:"quantity"`
	StationFee   float64 `json:"stationFee"`
	RefreshData  bool    `json:"refreshData"`
	UseFocus     bool    `json:"useFocus"`
	ZoneQuality  int     `json:"zoneQuality"`
	HideoutPower int     `json:"hideoutPower"`
	DailyBonus   string  `json:"dailyBonus"`
	// DemandDays is how many days of market history the demand columns cover.
	DemandDays int `json:"demandDays"`
	// SellCity and SellMode decide where and how the crafted item is priced.
	SellCity string `json:"sellCity,omitempty"`
	SellMode string `json:"sellMode,omitempty"`
	// BuyCities and BuyModes override the city and trade per material, keyed
	// by the material's item id. Entries equal to the defaults are dropped.
	BuyCities map[string]string `json:"buyCities,omitempty"`
	BuyModes  map[string]string `json:"buyModes,omitempty"`
	// Overrides holds manually entered unit prices, keyed by item id.
	Overrides map[string]float64 `json:"prices,omitempty"`
}

// DefaultState mirrors the defaults used by the original React store.
func DefaultState() State {
	item := ""
	if len(items) > 0 {
		item = items[0].UniqueName
	}
	return State{
		Item:         item,
		Quality:      2,
		Location:     "bonus-city",
		Quantity:     1,
		StationFee:   0,
		ZoneQuality:  2,
		HideoutPower: 1,
		DailyBonus:   "none",
		DemandDays:   7,
		SellCity:     DefaultCity,
		SellMode:     ModeOrder,
	}
}

// Normalize clamps every field to a valid option.
func (s State) Normalize() State {
	if _, ok := Lookup(s.Item); !ok {
		if len(items) > 0 {
			s.Item = items[0].UniqueName
		} else {
			s.Item = ""
		}
	}
	s.Quality = clamp(s.Quality, 1, 5)
	s.ZoneQuality = clamp(s.ZoneQuality, 1, 6)
	s.HideoutPower = clamp(s.HideoutPower, 1, 9)
	if s.Quantity < 1 {
		s.Quantity = 1
	}
	if s.StationFee < 0 {
		s.StationFee = 0
	}
	if !hasOption(Locations, s.Location) {
		s.Location = "bonus-city"
	}
	if !hasOption(DailyBonuses, s.DailyBonus) {
		s.DailyBonus = "none"
	}
	if !hasOption(DemandWindows, strconv.Itoa(s.DemandDays)) {
		s.DemandDays = 7
	}
	if !hasOption(Cities, s.SellCity) {
		s.SellCity = DefaultCity
	}
	if !hasOption(SellModes, s.SellMode) {
		s.SellMode = ModeOrder
	}
	s.BuyCities = pruneSelection(s.BuyCities, Cities, DefaultCity)
	s.BuyModes = pruneSelection(s.BuyModes, BuyModes, ModeInstant)
	for itemID, price := range s.Overrides {
		if itemID == "" || price <= 0 {
			delete(s.Overrides, itemID)
		}
	}
	if len(s.Overrides) == 0 {
		s.Overrides = nil
	}
	return s
}

// FromValues builds a normalized state from submitted form values.
func FromValues(values url.Values) State {
	s := State{
		Item:         values.Get("item"),
		Quality:      atoi(values.Get("quality"), 2),
		Location:     values.Get("location"),
		Quantity:     atoi(values.Get("quantity"), 1),
		StationFee:   atof(values.Get("stationFee"), 0),
		RefreshData:  truthy(values.Get("refreshData")),
		UseFocus:     truthy(values.Get("useFocus")),
		ZoneQuality:  atoi(values.Get("zoneQuality"), 2),
		HideoutPower: atoi(values.Get("hideoutPower"), 1),
		DailyBonus:   values.Get("dailyBonus"),
		DemandDays:   atoi(values.Get("demandDays"), 7),
		SellCity:     values.Get("sellCity"),
		SellMode:     values.Get("sellMode"),
		BuyCities:    parseSelections(values, cityPrefix),
		BuyModes:     parseSelections(values, modePrefix),
		Overrides:    parseOverrides(values),
	}
	return s.Normalize()
}

// parseOverrides reads price[ITEM_ID] form fields into a price map.
func parseOverrides(values url.Values) map[string]float64 {
	var overrides map[string]float64
	for key, entries := range values {
		itemID, ok := fieldItemID(key, overridePrefix)
		if !ok || len(entries) == 0 {
			continue
		}
		price, err := strconv.ParseFloat(strings.TrimSpace(entries[0]), 64)
		if err != nil || price <= 0 {
			continue
		}
		if overrides == nil {
			overrides = make(map[string]float64)
		}
		overrides[itemID] = price
	}
	return overrides
}

// parseSelections reads prefix[ITEM_ID]=value form fields into a map.
func parseSelections(values url.Values, prefix string) map[string]string {
	var selections map[string]string
	for key, entries := range values {
		itemID, ok := fieldItemID(key, prefix)
		if !ok || len(entries) == 0 {
			continue
		}
		if selections == nil {
			selections = make(map[string]string)
		}
		selections[itemID] = entries[0]
	}
	return selections
}

// fieldItemID extracts ITEM_ID from a prefix[ITEM_ID] form field name.
func fieldItemID(key, prefix string) (string, bool) {
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, "]") {
		return "", false
	}
	itemID := strings.TrimSuffix(strings.TrimPrefix(key, prefix), "]")
	return itemID, itemID != ""
}

// pruneSelection drops entries that are invalid or already the default, so the
// cookie only carries the deviations.
func pruneSelection(selections map[string]string, options []Option, fallback string) map[string]string {
	for itemID, value := range selections {
		if !hasOption(options, value) || value == fallback {
			delete(selections, itemID)
		}
	}
	if len(selections) == 0 {
		return nil
	}
	return selections
}

// CityFor returns the hub a material is bought in.
func (s State) CityFor(itemID string) string {
	if city, ok := s.BuyCities[itemID]; ok {
		return city
	}
	return DefaultCity
}

// ModeFor returns the trade a material is bought with.
func (s State) ModeFor(itemID string) string {
	if mode, ok := s.BuyModes[itemID]; ok {
		return mode
	}
	return ModeInstant
}

// Encode serializes state for use as a cookie value.
func (s State) Encode() (string, error) {
	payload, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

// DecodeState restores state from a cookie value.
func DecodeState(value string) (State, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(payload, &s); err != nil {
		return State{}, err
	}
	return s.Normalize(), nil
}

// Summary is the data shown in the crafting plan table.
type Summary struct {
	Item            Item
	Found           bool
	Tier            int
	QualityLabel    string
	LocationLabel   string
	DailyBonusLabel string
	StationFeeTotal float64
}

// Summarize resolves labels and derived values for the current state.
func (s State) Summarize() Summary {
	item, found := Lookup(s.Item)
	return Summary{
		Item:            item,
		Found:           found,
		Tier:            Tier(item.UniqueName),
		QualityLabel:    labelFor(Qualities, strconv.Itoa(s.Quality)),
		LocationLabel:   labelFor(Locations, s.Location),
		DailyBonusLabel: labelFor(DailyBonuses, s.DailyBonus),
		StationFeeTotal: s.StationFee * float64(s.Quantity),
	}
}

func labelFor(options []Option, value string) string {
	for _, option := range options {
		if option.Value == value {
			return option.Label
		}
	}
	return value
}

func hasOption(options []Option, value string) bool {
	for _, option := range options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func atoi(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func atof(value string, fallback float64) float64 {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func truthy(value string) bool {
	return value == "1" || value == "on" || value == "true"
}
