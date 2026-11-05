package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// newTestServer builds a Server whose market data comes from a fake market.
// The fake is returned so a test can add trade history to it.
func newTestServer(t *testing.T, prices map[string]int) (*Server, *fakeMarket) {
	t.Helper()

	server, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	fake := newFakeMarket(t, prices)
	client := fake.client()
	server.prices = newPriceCache(client, time.Minute)
	server.demands = newDemandCache(client, 1, time.Minute)
	return server, fake
}

func postForm(server *Server, form url.Values) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/crafting", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("HX-Request", "true")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	return recorder
}

func TestCraftingPageRendersIngredientColumns(t *testing.T) {
	server, fake := newTestServer(t, map[string]int{
		"T4_METALBAR":   1234,
		"T4_LEATHER":    567,
		"T4_MAIN_SWORD": 3000,
	})
	fake.setHistory(map[string]int{"T4_MAIN_SWORD": 12})

	recorder := postForm(server, url.Values{"item": {"T8_MAIN_SWORD"}})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		"<th>Metal Bar</th>",
		"<th>Leather</th>",
		`name="price[T4_METALBAR]"`,
		`placeholder="1234"`,
		`class="age age-`,
		`name="price[T4_MAIN_SWORD]"`,
		`name="buyCity[T4_METALBAR]"`,
		`name="buyMode[T4_METALBAR]"`,
		`name="sellCity"`,
		`name="sellMode"`,
		"<th class=\"num\">Demand (7d)</th>",
		"<th class=\"num\">Bought / day</th>",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}

	for _, gone := range []string{"<th>Quality</th>", "<th>Location</th>", "<th>Zone</th>",
		"<th>Hideout</th>", "<th>Daily Bonus</th>", "<th>Station Fee</th>", "<th>Focus</th>",
		"<th>Item</th>"} {
		if strings.Contains(body, gone) {
			t.Errorf("body still renders %q", gone)
		}
	}
}

func TestCraftingPageRendersDemandWindow(t *testing.T) {
	server, fake := newTestServer(t, map[string]int{"T4_METALBAR": 1234, "T4_LEATHER": 567})
	fake.setHistory(map[string]int{"T4_MAIN_SWORD": 12})

	recorder := postForm(server, url.Values{"item": {"T8_MAIN_SWORD"}, "demandDays": {"14"}})
	body := recorder.Body.String()

	if !strings.Contains(body, "<th class=\"num\">Demand (14d)</th>") {
		t.Error("demand window did not reach the table header")
	}
	// Twelve units a day over fourteen days is 168 units.
	if !strings.Contains(body, ">168<") {
		t.Error("demand sum for the chosen window missing")
	}
}

func TestCraftingPageRendersOverrides(t *testing.T) {
	server, _ := newTestServer(t, map[string]int{"T4_METALBAR": 1234, "T4_LEATHER": 567})

	recorder := postForm(server, url.Values{
		"item":               {"T8_MAIN_SWORD"},
		"price[T4_METALBAR]": {"777"},
	})
	body := recorder.Body.String()

	if !strings.Contains(body, `name="price[T4_METALBAR]"`) {
		t.Fatal("override input missing")
	}
	if !strings.Contains(body, `value="777"`) {
		t.Error("override value not rendered")
	}
}

func TestCraftingPageWithoutRecipe(t *testing.T) {
	server, _ := newTestServer(t, nil)

	recorder := postForm(server, url.Values{"item": {"T8_GVGSEASONREWARD_FAMEBUFF"}})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "No recipe data") {
		t.Error("expected the empty-state row")
	}
}

func TestCraftingPageMergesUnenchantedIngredients(t *testing.T) {
	server, _ := newTestServer(t, nil)

	recorder := postForm(server, url.Values{"item": {"T8_2H_ARCANESTAFF_CRYSTAL"}})
	body := recorder.Body.String()

	if !strings.Contains(body, `rowspan="5"`) {
		t.Error("expected artefact cells merged across a tier")
	}
	// One input per tier for the artefact, five tiers in total.
	if got := strings.Count(body, `name="price[T4_ARTEFACT_2H_ARCANESTAFF_CRYSTAL]"`); got != 1 {
		t.Errorf("tier 4 artefact inputs = %d, want 1", got)
	}
	if got := strings.Count(body, `name="price[T8_ARTEFACT_2H_ARCANESTAFF_CRYSTAL]"`); got != 1 {
		t.Errorf("tier 8 artefact inputs = %d, want 1", got)
	}
}

func TestCraftingPageKeepsTradeSelections(t *testing.T) {
	server, _ := newTestServer(t, map[string]int{"T4_METALBAR": 1000, "T4_LEATHER": 500})

	recorder := postForm(server, url.Values{
		"item":                 {"T8_MAIN_SWORD"},
		"sellCity":             {"Caerleon"},
		"sellMode":             {"instant"},
		"buyCity[T4_METALBAR]": {"Thetford"},
		"buyMode[T4_METALBAR]": {"order"},
	})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	body := recorder.Body.String()
	for _, want := range []string{
		`<option value="Caerleon" selected>Caerleon</option>`,
		`<option value="instant" selected>Instant sell</option>`,
		`<option value="Thetford" selected>Thetford</option>`,
		`<option value="order" selected>Buy order</option>`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
}
