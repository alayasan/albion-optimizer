package ui

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alayasan/albion-optimizer/web/internal/crafting"
	"github.com/alayasan/albion-optimizer/web/internal/market"
)

//go:embed templates/*.html static/*
var assets embed.FS

const (
	cookieName = "albion_state"
	cookieAge  = 60 * 60 * 24 * 365
)

// View is the data exposed to the HTML templates.
type View struct {
	State         crafting.State
	Items         []crafting.Item
	Selected      *crafting.Item
	Summary       crafting.Summary
	Qualities     []crafting.Option
	ZoneQualities []crafting.Option
	HideoutPowers []crafting.Option
	DailyBonuses  []crafting.Option
	Locations     []crafting.Option
	DemandWindows []crafting.Option
	Plan          PlanTable
}

// Server serves the htmx-powered crafting UI and its static assets.
type Server struct {
	templates *template.Template
	mux       *http.ServeMux
	prices    *priceCache
	demands   *demandCache
}

// New parses the embedded templates and wires up the routes.
func New() (*Server, error) {
	funcs := template.FuncMap{
		"selected": func(option string, current any) bool {
			return option == fmt.Sprint(current)
		},
		"money": func(value float64) string {
			return fmt.Sprintf("%.0f", value)
		},
		"number": func(value float64) string {
			return strconv.FormatFloat(value, 'f', -1, 64)
		},
	}

	templates, err := template.New("root").Funcs(funcs).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	staticFS, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, fmt.Errorf("static assets: %w", err)
	}

	client := market.New(priceRegion)
	s := &Server{
		templates: templates,
		mux:       http.NewServeMux(),
		prices:    newPriceCache(client, priceCity, priceTTL),
		demands:   newDemandCache(client, priceCity, priceQuality, priceTTL),
	}

	s.mux.HandleFunc("GET /", s.handleIndex)
	s.mux.HandleFunc("POST /crafting", s.handleCrafting)
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	state := s.readState(r)
	s.render(w, r, state)
}

func (s *Server) handleCrafting(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	state := crafting.FromValues(r.PostForm)
	s.writeState(w, state)
	s.render(w, r, state)
}

func (s *Server) render(w http.ResponseWriter, r *http.Request, state crafting.State) {
	view := newView(r.Context(), s.prices, s.demands, state)

	name := "layout"
	if isHTMX(r) {
		name = "app"
	}

	var body bytes.Buffer
	if err := s.templates.ExecuteTemplate(&body, name, view); err != nil {
		log.Printf("render %s: %v", name, err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Vary", "HX-Request")
	_, _ = body.WriteTo(w)
}

func newView(ctx context.Context, prices *priceCache, demands *demandCache, state crafting.State) View {
	view := View{
		State:         state,
		Items:         crafting.Items(),
		Qualities:     crafting.Qualities,
		ZoneQualities: crafting.ZoneQualities,
		HideoutPowers: crafting.HideoutPowers,
		DailyBonuses:  crafting.DailyBonuses,
		Locations:     crafting.Locations,
		DemandWindows: crafting.DemandWindows,
		Summary:       state.Summarize(),
	}

	if item, ok := crafting.Lookup(state.Item); ok {
		view.Selected = &item
		view.Plan = buildPlan(ctx, prices, demands, state, view.Selected, time.Now())
	}

	return view
}

func (s *Server) readState(r *http.Request) crafting.State {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return crafting.DefaultState()
	}
	state, err := crafting.DecodeState(cookie.Value)
	if err != nil {
		return crafting.DefaultState()
	}
	return state
}

func (s *Server) writeState(w http.ResponseWriter, state crafting.State) {
	value, err := state.Encode()
	if err != nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Path:     "/",
		MaxAge:   cookieAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func isHTMX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}
