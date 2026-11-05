package crafting

import (
	"net/url"
	"reflect"
	"testing"
)

func TestItemsLoadedAndSorted(t *testing.T) {
	list := Items()
	if len(list) == 0 {
		t.Fatal("expected embedded items")
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].Label > list[i].Label {
			t.Fatalf("items not sorted: %q before %q", list[i-1].Label, list[i].Label)
		}
	}
	if _, ok := Lookup(list[0].UniqueName); !ok {
		t.Fatalf("lookup failed for %q", list[0].UniqueName)
	}
}

func TestNormalizeClampsInvalidValues(t *testing.T) {
	got := State{
		Item:         "not-a-real-item",
		Quality:      99,
		Location:     "somewhere",
		Quantity:     0,
		StationFee:   -5,
		ZoneQuality:  0,
		HideoutPower: 42,
		DailyBonus:   "50",
	}.Normalize()

	if got.Quality != 5 {
		t.Errorf("quality = %d, want 5", got.Quality)
	}
	if got.ZoneQuality != 1 {
		t.Errorf("zoneQuality = %d, want 1", got.ZoneQuality)
	}
	if got.HideoutPower != 9 {
		t.Errorf("hideoutPower = %d, want 9", got.HideoutPower)
	}
	if got.Quantity != 1 {
		t.Errorf("quantity = %d, want 1", got.Quantity)
	}
	if got.StationFee != 0 {
		t.Errorf("stationFee = %v, want 0", got.StationFee)
	}
	if got.Location != "bonus-city" {
		t.Errorf("location = %q, want bonus-city", got.Location)
	}
	if got.DailyBonus != "none" {
		t.Errorf("dailyBonus = %q, want none", got.DailyBonus)
	}
	if got.Item == "not-a-real-item" {
		t.Error("invalid item was not replaced")
	}
}

func TestFromValuesParsesForm(t *testing.T) {
	values := map[string][]string{
		"item":         {Items()[0].UniqueName},
		"quality":      {"4"},
		"location":     {"hideout"},
		"quantity":     {"25"},
		"stationFee":   {"1200"},
		"refreshData":  {"1"},
		"useFocus":     {"on"},
		"zoneQuality":  {"6"},
		"hideoutPower": {"9"},
		"dailyBonus":   {"20"},
	}

	got := FromValues(values)

	if got.Quality != 4 {
		t.Errorf("quality = %d, want 4", got.Quality)
	}
	if got.Location != "hideout" {
		t.Errorf("location = %q, want hideout", got.Location)
	}
	if got.Quantity != 25 {
		t.Errorf("quantity = %d, want 25", got.Quantity)
	}
	if got.StationFee != 1200 {
		t.Errorf("stationFee = %v, want 1200", got.StationFee)
	}
	if !got.RefreshData || !got.UseFocus {
		t.Error("expected refreshData and useFocus to be true")
	}
	if got.ZoneQuality != 6 || got.HideoutPower != 9 || got.DailyBonus != "20" {
		t.Errorf("unexpected bonuses: %+v", got)
	}
}

func TestStateCookieRoundTrip(t *testing.T) {
	original := DefaultState()
	original.Quality = 5
	original.Quantity = 10

	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	decoded, err := DecodeState(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !reflect.DeepEqual(decoded, original) {
		t.Errorf("round trip mismatch: got %+v want %+v", decoded, original)
	}
}

func TestSummaryLabels(t *testing.T) {
	state := DefaultState()
	state.Quality = 3
	state.DailyBonus = "20"

	summary := state.Summarize()
	if !summary.Found {
		t.Fatal("expected default item to resolve")
	}
	if summary.QualityLabel != "Outstanding" {
		t.Errorf("qualityLabel = %q, want Outstanding", summary.QualityLabel)
	}
	if summary.DailyBonusLabel != "20%" {
		t.Errorf("dailyBonusLabel = %q, want 20%%", summary.DailyBonusLabel)
	}
	if summary.Tier <= 0 {
		t.Errorf("tier = %d, want > 0", summary.Tier)
	}
}

func TestTierExtraction(t *testing.T) {
	cases := map[string]int{
		"T8_MAIN_SWORD": 8,
		"T3_2H_BOW":     3,
		"NO_TIER":       0,
	}
	for name, want := range cases {
		if got := Tier(name); got != want {
			t.Errorf("Tier(%q) = %d, want %d", name, got, want)
		}
	}
}

func TestDemandDaysDefaultsAndNormalizes(t *testing.T) {
	if got := DefaultState().DemandDays; got != 7 {
		t.Errorf("default demand days = %d, want 7", got)
	}

	state := DefaultState()
	state.DemandDays = 9
	if got := state.Normalize().DemandDays; got != 7 {
		t.Errorf("normalized demand days = %d, want the 7 fallback", got)
	}

	state.DemandDays = 30
	if got := state.Normalize().DemandDays; got != 30 {
		t.Errorf("normalized demand days = %d, want 30 kept", got)
	}

	form := url.Values{"demandDays": {"14"}}
	if got := FromValues(form).DemandDays; got != 14 {
		t.Errorf("form demand days = %d, want 14", got)
	}
}

func TestMaterialsDefaultToInstantBuyInTheDefaultCity(t *testing.T) {
	state := DefaultState()
	if got := state.CityFor("T4_METALBAR"); got != DefaultCity {
		t.Errorf("city = %q, want %q", got, DefaultCity)
	}
	if got := state.ModeFor("T4_METALBAR"); got != ModeInstant {
		t.Errorf("mode = %q, want %q", got, ModeInstant)
	}
}

func TestTradeSelectionsFromForm(t *testing.T) {
	state := FromValues(url.Values{
		"item":                 {"T8_MAIN_SWORD"},
		"sellCity":             {"Caerleon"},
		"sellMode":             {ModeInstant},
		"buyCity[T4_METALBAR]": {"Thetford"},
		"buyMode[T4_METALBAR]": {ModeOrder},
	})

	if state.SellCity != "Caerleon" || state.SellMode != ModeInstant {
		t.Errorf("sell at %q/%q, want Caerleon/instant", state.SellCity, state.SellMode)
	}
	if got := state.CityFor("T4_METALBAR"); got != "Thetford" {
		t.Errorf("material city = %q, want Thetford", got)
	}
	if got := state.ModeFor("T4_METALBAR"); got != ModeOrder {
		t.Errorf("material mode = %q, want order", got)
	}
}

func TestTradeSelectionsDropUnknownValues(t *testing.T) {
	state := FromValues(url.Values{
		"item":                 {"T8_MAIN_SWORD"},
		"sellCity":             {"Nowhere"},
		"sellMode":             {"bogus"},
		"buyCity[T4_METALBAR]": {"Nowhere"},
		"buyMode[T4_METALBAR]": {"bogus"},
	})

	if state.SellCity != DefaultCity || state.SellMode != ModeOrder {
		t.Errorf("sell at %q/%q, want the defaults", state.SellCity, state.SellMode)
	}
	if state.BuyCities != nil || state.BuyModes != nil {
		t.Errorf("kept invalid selections: %v/%v", state.BuyCities, state.BuyModes)
	}
}

func TestTradeSelectionsDropDefaults(t *testing.T) {
	state := FromValues(url.Values{
		"item":                 {"T8_MAIN_SWORD"},
		"buyCity[T4_METALBAR]": {DefaultCity},
		"buyMode[T4_METALBAR]": {ModeInstant},
	})

	if len(state.BuyCities) != 0 || len(state.BuyModes) != 0 {
		t.Errorf("kept entries equal to the defaults: %v/%v", state.BuyCities, state.BuyModes)
	}
}
