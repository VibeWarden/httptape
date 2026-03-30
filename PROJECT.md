# httptape

**HTTP traffic recording, sanitization, and replay for Go.**

An embeddable Go library that captures HTTP request/response pairs, sanitizes sensitive data on write, and replays them as a mock server. Think WireMock, but native Go, embeddable, and with sanitization built into the core.

## Why

- **WireMock requires Java** — separate process, 200MB+ memory, can't embed in a Go binary
- **Go test mocking libraries** (`gock`, `httpmock`) only work inside test code — no standalone server, no recording, no fixture management
- **Nobody does sanitization** — existing tools record raw traffic including secrets and PII. Sanitization is always an afterthought. httptape sanitizes on write — sensitive data never hits disk.

## Use cases

- **Integration test fixtures** — record real API interactions once, replay forever. Deterministic CI without live API credentials.
- **Local development** — mock external APIs without internet access or API keys.
- **Production traffic capture** — safely record a sample of live traffic for debugging, with PII automatically redacted.
- **Environment portability** — record in prod, sanitize, export, import in dev. Same API behavior, no sensitive data.

## Core principles

- **Embeddable** — import as a Go package, not a separate process. Works in any Go application.
- **Sanitize on write** — sensitive data is redacted before it touches disk. There is no "raw" recording.
- **Deterministic faking** — same input always produces the same fake output (HMAC with project seed). Preserves relational consistency across fixtures.
- **Zero or minimal dependencies** — stdlib-first. No frameworks, no heavy abstractions.
- **Progressive matching** — start with exact match, grow to fuzzy/regex. Simple cases stay simple.

## Architecture

### Core types

```go
// Tape is a recorded HTTP interaction
type Tape struct {
    ID         string         `json:"id"`
    Request    RecordedReq    `json:"request"`
    Response   RecordedResp   `json:"response"`
    Route      string         `json:"route,omitempty"`
    RecordedAt time.Time      `json:"recorded_at"`
    Metadata   map[string]any `json:"metadata,omitempty"`
}

type RecordedReq struct {
    Method   string              `json:"method"`
    URL      string              `json:"url"`
    Headers  map[string][]string `json:"headers"`
    Body     json.RawMessage     `json:"body,omitempty"`
    BodyHash string              `json:"body_hash,omitempty"`
}

type RecordedResp struct {
    Status  int                 `json:"status"`
    Headers map[string][]string `json:"headers"`
    Body    json.RawMessage     `json:"body,omitempty"`
}
```

### API surface

```go
// Recording — wraps http.RoundTripper
recorder := httptape.NewRecorder(http.DefaultTransport,
    httptape.WithSanitizer(sanitizer),
    httptape.WithStorage(store),
    httptape.WithSampling(0.01),       // 1% sampling
    httptape.WithAsync(true),           // non-blocking writes
)
client := &http.Client{Transport: recorder}

// Sanitization — configured declaratively
sanitizer := httptape.NewSanitizer(
    httptape.RedactHeaders("Authorization", "Cookie", "X-Api-Key"),
    httptape.RedactBodyPaths("$.card.number", "$.ssn"),
    httptape.FakeFields("$.email", "$.user_id"),  // deterministic faking
    httptape.WithSeed("project-specific-seed"),     // HMAC seed for deterministic output
)

// Mock server — serves recorded fixtures
server := httptape.NewServer(store,
    httptape.WithFallbackStatus(502),
    httptape.WithMatcher(matcher),
)
http.ListenAndServe(":8081", server)

// Matching — progressive complexity
matcher := httptape.NewMatcher(
    httptape.MatchRoute(),         // by route name
    httptape.MatchMethod(),        // by HTTP method
    httptape.MatchPath(),          // by URL path
    httptape.MatchQueryParams(),   // by query parameters
    httptape.MatchBodyHash(),      // by body content hash
)

// Storage — pluggable
store := httptape.NewFileStore("./fixtures/")
// or
store := httptape.NewMemoryStore()

// Import/Export
bundle, _ := store.Export()         // returns tar.gz bytes
store.Import(bundle)                // loads from tar.gz
```

### Package structure

```
httptape/
  tape.go              # Core Tape type and related types
  recorder.go          # RoundTripper wrapper for recording
  recorder_test.go
  sanitizer.go         # Redaction and deterministic faking
  sanitizer_test.go
  server.go            # Mock HTTP server (http.Handler)
  server_test.go
  matcher.go           # Request-to-fixture matching
  matcher_test.go
  store.go             # Storage interface
  store_file.go        # Filesystem storage
  store_file_test.go
  store_memory.go      # In-memory storage (for tests)
  store_memory_test.go
  bundle.go            # Import/export (tar.gz)
  bundle_test.go
  options.go           # Functional options
  doc.go               # Package documentation
```

## Design decisions

| Decision | Choice | Reason |
|---|---|---|
| Language | Go | Embeddable in Go applications, single binary |
| License | Apache 2.0 | Permissive, compatible with most projects |
| Dependencies | stdlib only (v1) | Easy to embed, no transitive dependency issues |
| Sanitization | On write, not on export | Sensitive data never touches disk |
| Deterministic faking | HMAC-based | Same input → same fake, preserves relational consistency |
| Fixture format | JSON | Human-readable, easy to inspect and edit |
| Storage interface | Pluggable | Filesystem for production, memory for tests |
| Matching | Progressive | Exact match first, fuzzy/regex as opt-in |
| Recording | Async by default | Non-blocking, minimal hot-path overhead |
| Body handling | Store hash + full body | Hash for matching, full body for replay |

## Non-goals (v1)

- **Standalone CLI** — v1 is a library. A CLI wrapper can come later.
- **HTTP/2, gRPC, WebSocket** — HTTP/1.1 first. Protocol support can be added.
- **Templated responses** — WireMock-style response templating is a v2 feature.
- **Stateful behaviors** — scenario-based state machines are a v2 feature.
- **GUI** — no web UI. Fixtures are JSON files, use your editor.

## Integration with VibeWarden

httptape is designed as a standalone library but will be used by VibeWarden's egress proxy:

- **VibeWarden wraps the egress `http.Client` with `httptape.Recorder`** when recording mode is enabled
- **VibeWarden runs `httptape.Server`** as the egress backend when mock/replay mode is enabled
- **VibeWarden adds CLI commands** (`vibew egress record/export/import`) that delegate to httptape's storage and bundle APIs
- **VibeWarden adds OTel instrumentation** on top of httptape operations

The library itself has no knowledge of VibeWarden, Caddy, OTel, or any sidecar concepts.
