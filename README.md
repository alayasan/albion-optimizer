# Albion-Optimizer

A one-stop webapp for your Albion Online optimization needs.

## Status

The web app is being migrated from React + Vite to **Go + htmx**
(server-rendered HTML, no client-side framework).

## Layout

```
web/     Go server, htmx templates, embedded item data
```

## Run the web app

Requires Go 1.22+.

```bash
cd web
go run ./cmd/server
```

Open http://localhost:8080 (override with `ADDR`, e.g. `ADDR=:3000`).

More details in [web/README.md](web/README.md).

## Development notes

- Item data is preprocessed from the Albion Online game files into
  `web/internal/crafting/data/items.json` (craftable gear, highest tier only)
  and embedded into the binary at build time.
- UI state (selected item, quality, fees, …) is stored in an HttpOnly cookie.
- All interactive updates are htmx partial swaps against `/crafting`.
