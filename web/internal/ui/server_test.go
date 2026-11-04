package ui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// newTestServer builds a Server whose prices come from a fake market.
func newTestServer(t *testing.T, prices map[string]int) *Server {
	t.Helper()

	server, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	server.prices = newPriceCache(newFakeMarket(t, prices).client(), "Caerleon", time.Minute)
	return server
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
	server := newTestServer(t, map[string]int{"T4_METALBAR": 1234, "T4_LEATHER": 567})

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
		"Steel Bar &times;16",
		"Worked Leather &times;8",
		"2026-11-03 02:10",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}

	if strings.Contains(body, "<th>Item</th>") {
		t.Error("body still renders the item name column")
	}
}

func TestCraftingPageRendersOverrides(t *testing.T) {
	server := newTestServer(t, map[string]int{"T4_METALBAR": 1234, "T4_LEATHER": 567})

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
	server := newTestServer(t, nil)

	recorder := postForm(server, url.Values{"item": {"T8_GVGSEASONREWARD_FAMEBUFF"}})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "No recipe data") {
		t.Error("expected the empty-state row")
	}
}
