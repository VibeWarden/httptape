# httptape — Backlog

## Milestone 1: Core (v0.1.0)

Foundation — record, replay, and store HTTP interactions.

### 1.1 Core types and storage interface

**Priority:** high

- [ ] `Tape`, `RecordedReq`, `RecordedResp` types in `tape.go`
- [ ] `Store` interface: `Save(Tape)`, `Load(id) -> Tape`, `List(filter) -> []Tape`, `Delete(id)`
- [ ] `MemoryStore` implementation (for tests and embedding)
- [ ] `FileStore` implementation (JSON files on disk, organized by route)
- [ ] Directory structure: `fixtures/<route>/<method>_<path_hash>.json`
- [ ] Unit tests for both store implementations

### 1.2 Recorder (RoundTripper wrapper)

**Priority:** high

- [ ] `Recorder` implements `http.RoundTripper`
- [ ] Wraps an inner `http.RoundTripper`, captures request and response
- [ ] Writes `Tape` to `Store` after each round-trip
- [ ] Functional options: `WithStorage`, `WithSanitizer`, `WithSampling`, `WithAsync`
- [ ] Async mode: write to a buffered channel, background goroutine drains to store
- [ ] Sampling: probabilistic (0.0-1.0), only record a fraction of traffic
- [ ] Graceful shutdown: flush pending recordings on close
- [ ] Does not modify request or response — transparent to the caller
- [ ] Unit tests with a test HTTP server

### 1.3 Mock server (http.Handler)

**Priority:** high

- [ ] `Server` implements `http.Handler`
- [ ] Receives HTTP requests, matches against stored `Tape` fixtures
- [ ] Returns the matched fixture's response (status, headers, body)
- [ ] Configurable fallback status when no fixture matches (default: 502)
- [ ] Unit tests with recorded fixtures

### 1.4 Matcher (request-to-fixture matching)

**Priority:** high

- [ ] `Matcher` interface: `Match(request, []Tape) -> (Tape, bool)`
- [ ] Composable match criteria via functional options:
  - `MatchMethod()` — HTTP method must match
  - `MatchPath()` — URL path must match
  - `MatchRoute()` — route name must match (if set)
  - `MatchQueryParams()` — query parameters must match
  - `MatchBodyHash()` — request body hash must match
- [ ] Match priority: more specific matches win (e.g., body hash match > path-only match)
- [ ] `DefaultMatcher` — method + path (covers 90% of cases)
- [ ] Unit tests for each match criteria and priority resolution

---

## Milestone 2: Sanitization (v0.2.0)

The differentiator — redaction and deterministic faking.

### 2.1 Header redaction

**Priority:** high

- [ ] `Sanitizer` type with configurable rules
- [ ] `RedactHeaders(names ...string)` — replace header values with `[REDACTED]`
- [ ] Applied to both request and response headers
- [ ] Common defaults available: `DefaultSensitiveHeaders` (Authorization, Cookie, Set-Cookie, X-Api-Key)
- [ ] Unit tests

### 2.2 Body path redaction

**Priority:** high

- [ ] `RedactBodyPaths(paths ...string)` — redact fields in JSON bodies using JSONPath-like syntax
- [ ] Supported syntax: `$.field`, `$.nested.field`, `$.array[*].field`
- [ ] Redacted values replaced with `"[REDACTED]"` (string), `0` (number), `false` (bool)
- [ ] Non-JSON bodies: skip body redaction, log a warning
- [ ] Unit tests with nested and array JSON structures

### 2.3 Deterministic faking

**Priority:** high

- [ ] `FakeFields(paths ...string)` — replace field values with deterministic fakes
- [ ] HMAC-SHA256 with a configurable project seed
- [ ] Same input value always produces the same fake output (cross-fixture consistency)
- [ ] Faking strategies by detected type:
  - Email: `user_<hash>@example.com`
  - UUID: deterministic UUID v5 from input
  - Numeric ID: deterministic integer from hash
  - String: `fake_<hash_prefix>`
- [ ] `WithSeed(seed string)` — project-level seed for HMAC
- [ ] Unit tests verifying determinism and consistency

### 2.4 Sanitizer integration with Recorder

**Priority:** high

- [ ] Recorder applies Sanitizer before writing to Store
- [ ] Sanitization is mandatory — no way to record without at least a no-op sanitizer
- [ ] Sanitizer is applied to the Tape (not the live request/response — caller is unaffected)
- [ ] Integration test: record with sanitizer, verify fixtures are redacted

---

## Milestone 3: Import/Export (v0.3.0)

Move fixtures between environments safely.

### 3.1 Bundle export

**Priority:** medium

- [ ] `Store.Export() -> (io.Reader, error)` — produces a tar.gz bundle
- [ ] Bundle contains:
  - `manifest.json` — fixture count, route list, export timestamp, sanitizer config used
  - `fixtures/` — all fixture JSON files preserving directory structure
- [ ] Export from any Store implementation (FileStore, MemoryStore)
- [ ] Unit tests

### 3.2 Bundle import

**Priority:** medium

- [ ] `Store.Import(io.Reader) -> error` — loads a tar.gz bundle
- [ ] Validates manifest format and fixture schema before importing
- [ ] Merge strategy: overwrite existing fixtures with same ID, keep others
- [ ] Import into any Store implementation
- [ ] Unit tests

### 3.3 Selective export

**Priority:** low

- [ ] Export filtered subsets: by route, by time range, by method
- [ ] `Store.Export(WithRoutes("stripe", "s3"), WithSince(time.Time))`
- [ ] Unit tests

---

## Milestone 4: Advanced matching (v0.4.0)

Progressive matching complexity for power users.

### 4.1 Regex path matching

**Priority:** medium
**Label:** to-refine

- [ ] `MatchPathRegex(pattern)` — match URL path against regex
- [ ] Useful for parameterized paths (`/users/\d+/orders`)
- [ ] Unit tests

### 4.2 Header matching

**Priority:** medium
**Label:** to-refine

- [ ] `MatchHeaders(key, value)` — require specific header values
- [ ] Useful for API version matching (`Accept: application/vnd.api.v2+json`)
- [ ] Unit tests

### 4.3 Fuzzy body matching

**Priority:** low
**Label:** to-refine

- [ ] `MatchBodyFuzzy(paths ...string)` — match only specific fields in request body
- [ ] Ignore fields that vary per request (timestamps, nonces, request IDs)
- [ ] Unit tests

---

## Milestone 5: Production readiness (v0.5.0)

### 5.1 Concurrent safety audit

**Priority:** high

- [ ] All types safe for concurrent use (Recorder, Server, Store, Sanitizer)
- [ ] Race detector passes on all tests
- [ ] Document concurrency guarantees in godoc

### 5.2 Performance benchmarks

**Priority:** medium

- [ ] Benchmark: Recorder overhead per request (target: <50us for async mode)
- [ ] Benchmark: Server response latency (target: <1ms for exact match)
- [ ] Benchmark: Sanitizer throughput for large JSON bodies
- [ ] Benchmark: FileStore write throughput
- [ ] Results documented in README

### 5.3 Error handling and edge cases

**Priority:** medium

- [ ] Binary response bodies (images, protobuf) — store as base64
- [ ] Large bodies — configurable max body size, truncate with warning
- [ ] Malformed JSON in body — store raw, skip body-level sanitization
- [ ] Empty responses (204 No Content) — handled correctly
- [ ] Redirect chains — record each hop or final response only (configurable)
- [ ] Unit tests for all edge cases

---

## Future (v1.0.0+)

These are ideas for after the core is stable. Not yet refined.

- **Standalone CLI** — `httptape record`, `httptape serve`, `httptape export/import`
- **Response templating** — dynamic responses using request data (WireMock-style)
- **Stateful scenarios** — state machine for multi-step API interactions
- **HTTP/2 support**
- **gRPC support** — record/replay gRPC calls
- **WebSocket support** — record/replay WebSocket message streams
- **Remote storage adapters** — S3, GCS for team fixture sharing
- **Fixture diffing** — detect when live API behavior diverges from recordings
- **Web UI** — browse and edit fixtures in a browser
