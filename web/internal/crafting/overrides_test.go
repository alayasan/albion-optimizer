package crafting

import (
	"reflect"
	"testing"
)

func TestParseOverrides(t *testing.T) {
	values := map[string][]string{
		"price[T4_METALBAR]":     {"1200"},
		"price[T4_LEATHER]":      {" 850.5 "},
		"price[T4_PLANKS]":       {""},
		"price[T4_CLOTH]":        {"0"},
		"price[T4_ARTEFACT_XYZ]": {"-4"},
		"price[]":                {"100"},
		"unrelated":              {"1"},
	}

	got := FromValues(values).Overrides
	want := map[string]float64{
		"T4_METALBAR": 1200,
		"T4_LEATHER":  850.5,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("overrides = %v, want %v", got, want)
	}
}

func TestOverridesSurviveCookieRoundTrip(t *testing.T) {
	state := DefaultState()
	state.Overrides = map[string]float64{"T4_METALBAR": 1500, "T7_SOUL_LEVEL3": 99999}

	encoded, err := state.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeState(encoded)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !reflect.DeepEqual(decoded.Overrides, state.Overrides) {
		t.Errorf("overrides = %v, want %v", decoded.Overrides, state.Overrides)
	}
}

func TestNormalizeDropsEmptyOverrides(t *testing.T) {
	got := State{
		Overrides: map[string]float64{"": 100, "T4_METALBAR": 0, "T4_LEATHER": 50},
	}.Normalize()

	want := map[string]float64{"T4_LEATHER": 50}
	if !reflect.DeepEqual(got.Overrides, want) {
		t.Errorf("overrides = %v, want %v", got.Overrides, want)
	}

	cleared := State{Overrides: map[string]float64{"T4_METALBAR": 0}}.Normalize()
	if cleared.Overrides != nil {
		t.Errorf("overrides = %v, want nil", cleared.Overrides)
	}
}

func TestDefaultStateHasNoOverrides(t *testing.T) {
	if got := DefaultState().Overrides; got != nil {
		t.Errorf("default overrides = %v, want nil", got)
	}
}
