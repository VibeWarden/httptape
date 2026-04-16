# Frontend-first development with httptape proxy mode

Working example of [httptape](https://github.com/VibeWarden/httptape) used as a fallback proxy in front of a backend API. The frontend never breaks, even when the backend goes down — it transparently falls back to cached data, and the UI reflects the current source **live, without any user action**.

## Architecture

```
Browser ──► proxy (:3001) ──► upstream (:8081)
              │ │                  │
              │ └─ on failure ◄─┐  └─ serves "real" data
              │                 │     from mocks/upstream-fixtures/
              │                 │
              │  L1 (memory)  ──┤
              │  L2 (disk)    ──┘
              │
              └─ /__httptape/health/stream  (SSE — live state updates)
```

Three services in docker-compose:

| Service | Role |
|---|---|
| `upstream` | Simulated real backend (`httptape serve` from `mocks/upstream-fixtures/`) |
| `proxy` | `httptape proxy` with L1+L2 caching, CORS, `--fallback-on-5xx`, and the `/__httptape/health` endpoint enabled |
| `frontend` | React + Vite UI (this example's source) |

## How cache fallback works

| Upstream state | Source served | Badge |
|---|---|---|
| Reachable | Live response from upstream, cached in L1 (raw) and L2 (redacted) | green **Live** |
| Down, L1 has the entry | L1 cached response (raw, current session) | yellow **L1 Cache** |
| Down, L1 empty (fresh proxy start) | L2 cache (disk, redacted seed) | red **L2 Cache** |

The upstream and L2 fixtures contain visibly different data so you can tell them apart at a glance:

| | Upstream (live) | L2 cache (seed) |
|---|---|---|
| Prices | Precise (`$49.95`) | Rounded (`$50.00`) |
| Descriptions | Detailed | Shorter, suffixed `(cached)` |

## Live status updates

The proxy exposes `GET /__httptape/health/stream` (Server-Sent Events) when started with `--health-endpoint`. The frontend opens an `EventSource` on that URL and:

1. Updates the source badge whenever the proxy's perceived upstream state changes.
2. Re-fetches data so the page reflects the new source.

No polling, no manual refresh. See [`src/useHealthStream.ts`](src/useHealthStream.ts).

## Quick start

```bash
docker compose up --build
```

The first time, this builds `httptape:demo` from this repository's root `Dockerfile` (the version in this example needs the health endpoint, which ships in v0.9.0+). Subsequent runs are fast.

Open [http://localhost:3000](http://localhost:3000).

## Try it

The badge in the sidebar shows the current data source. Run any of these and watch it flip live (~2 second debounce on the upstream probe):

```bash
docker compose stop upstream    # → yellow "L1 Cache"
docker compose restart proxy    # → red "L2 Cache" (L1 cleared on restart)
docker compose start upstream   # → green "Live"
```

The `./scripts/toggle-upstream.sh` script does the stop/start dance.

## Local development without Docker

```bash
# Build the httptape CLI (needs Go 1.26+)
( cd ../.. && go build -o /tmp/httptape ./cmd/httptape )

# Terminal 1: upstream
/tmp/httptape serve --fixtures ./mocks/upstream-fixtures --cors --addr :8082

# Terminal 2: proxy with health endpoint
/tmp/httptape proxy \
  --upstream http://localhost:8082 \
  --fixtures ./mocks/fixtures \
  --config ./mocks/sanitize.json \
  --port 3001 \
  --cors --fallback-on-5xx \
  --health-endpoint --upstream-probe-interval=2s

# Terminal 3: Vite dev server
VITE_API_URL=http://localhost:3001 npm run dev
```

## Adding new endpoints

For each new endpoint, drop a JSON fixture into both:

- `mocks/upstream-fixtures/` — the "real" backend response
- `mocks/fixtures/` — the L2 seed cache (committed alongside source — should be already-redacted)

Each file is a single httptape `Tape` (request/response pair). httptape picks up new fixtures without restart.

## Project layout

```
ts-frontend-first/
  src/
    api.ts                       # fetch wrapper
    App.tsx                      # main app: profile + product grid + add-to-cart
    useHealthStream.ts           # SSE hook driving the badge + refetch
    components/
      ProfileCard.tsx            # identity + credit-card visualization (shows redaction)
      ArchitectureDiagram.tsx    # live diagram of upstream/cache state
      Instructions.tsx           # the "Try it" copy-button list
  mocks/
    upstream-fixtures/           # "real" responses (precise prices)
    fixtures/                    # L2 seed cache (rounded prices, "(cached)" suffix)
    sanitize.json                # httptape redaction config (typed fakers)
  scripts/
    toggle-upstream.sh           # one-liner to flip upstream up/down
  docker-compose.yml             # 3 services, builds httptape:demo from ../../
  Dockerfile                     # multi-stage build for the React frontend
```
