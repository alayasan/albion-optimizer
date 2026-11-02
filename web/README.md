# Albion Optimizer — Web

Server-rendered web app built with **Go** and **htmx**.

## Requirements

- Go 1.22+

## Run

```bash
cd web
go run ./cmd/server
```

Then open http://localhost:8080

Set `ADDR` to change the listen address (default `:8080`).

## Structure

```
web/
├── cmd/server/main.go        # entrypoint, graceful shutdown
├── internal/crafting/        # domain: items, options, form state
│   └── data/items.json       # craftable gear, highest tier only
└── internal/ui/              # HTTP layer
    ├── server.go             # handlers, cookie state, template wiring
    ├── templates/            # html/template partials
    └── static/app.css        # Albion dark theme
```

## How it works

- The full page is a `layout.html` shell.
- Every control lives in one `<form>` that posts to `/crafting` via htmx
  (`hx-trigger="change delay:150ms"`), and the `#app` fragment is swapped
  in place — no client-side state library.
- Crafting state is persisted in an HttpOnly cookie so it survives reloads.
- Item data is embedded into the Go binary at build time.

## Notes

- Item art is loaded from `render.albiononline.com`.
- htmx is currently loaded from the unpkg CDN; swap it for a vendored copy
  under `static/` if you need fully offline builds.
