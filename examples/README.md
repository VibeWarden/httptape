# httptape examples

Each subdirectory is a self-contained, runnable example of httptape used in a real scenario. They're built and tested against the version of httptape in this repository — when a feature lands here, the relevant example is updated alongside it.

| Example | What it shows |
|---|---|
| [`ts-frontend-first/`](ts-frontend-first/) | Vite + React frontend talking to an httptape proxy, with live source-state updates over SSE. Demonstrates fallback-to-cache (live → L1 → L2) and per-event redaction in `mocks/sanitize.json`. |

> More examples coming: Go-embedded library use, Kotlin testcontainers, Python CI fixtures, Kotlin proxy integration. Tracked in `httptape-demos/` for now; will land here as each is polished.

## Conventions

- Each example owns its own dependencies — Go examples have their own `go.mod` so they don't pollute the library's stdlib-only constraint.
- `docker-compose.yml`, where present, pins to a published httptape image (`ghcr.io/vibewarden/httptape:<version>`) so examples Just Run on a fresh clone — no local build of httptape required. Bumped per release.
- Examples are kept opinionated and minimal — they showcase httptape behavior, not framework taste.
