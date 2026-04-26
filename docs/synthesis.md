# Synthesis Mode

Synthesis mode enables httptape to generate responses for URL patterns using
exemplar tapes. Instead of requiring an exact recorded tape for every URL,
you author a single exemplar with a URL pattern (e.g., `/users/:id`) and
httptape synthesizes responses for any matching request.

## When to use synthesis

Synthesis is designed for **frontend-first development workflows** where:

- The backend is partially specified and the UI is being built against a mock.
- You want realistic, deterministic responses for a family of URLs (e.g.,
  `/users/1`, `/users/2`, `/users/999`) without recording each one.
- You need the mock server to behave like a lightweight contract-driven stub.

Synthesis is **not** intended for integration tests where unexpected requests
should fail fast. In that case, leave synthesis disabled (the default).

## How it differs from replay

| Feature | Replay (default) | Synthesis |
|---|---|---|
| Matching | Exact URL | URL pattern (e.g., `/users/:id`) |
| Response source | Recorded tape | Exemplar tape with template expressions |
| Opt-in | Always on | Requires `WithSynthesis()` or `--synthesize` |
| Template resolution | Optional (bodies with `{{...}}`) | Always (exemplar bodies are templates) |
| Type coercion | Not applicable | `| int`, `| float`, `| bool` in JSON bodies |

## Opt-in gating (two levels)

Synthesis requires two levels of opt-in:

1. **Per-tape**: the `"exemplar": true` field marks a tape as an exemplar.
2. **Per-server**: the `WithSynthesis()` option (or `--synthesize` CLI flag)
   enables the exemplar fallback path.

Without both, exemplar tapes are loaded but never consulted.

## Authoring exemplar tapes

An exemplar tape is a JSON fixture with:

- `"exemplar": true` at the tape level.
- `"url_pattern"` on the request (instead of `"url"`).
- Template expressions in the response body.

Example (`fixtures/users-exemplar.json`):

```json
{
  "id": "users-exemplar",
  "route": "",
  "recorded_at": "2026-01-01T00:00:00Z",
  "exemplar": true,
  "request": {
    "method": "GET",
    "url_pattern": "/users/:id",
    "headers": {},
    "body": null,
    "body_hash": ""
  },
  "response": {
    "status_code": 200,
    "headers": {
      "Content-Type": ["application/json"]
    },
    "body": {
      "id": "{{pathParam.id | int}}",
      "name": "{{faker.name seed=user-{{pathParam.id}}}}",
      "email": "{{faker.email seed=user-{{pathParam.id}}}}"
    }
  }
}
```

### Validation rules

- `exemplar: true` requires `url_pattern` to be set.
- `url_pattern` requires `exemplar: true`.
- `url` and `url_pattern` are mutually exclusive.
- SSE exemplar tapes are not supported.

Invalid tapes produce a startup error when loading fixtures.

## Type coercion

In JSON response bodies, template expressions can include a type coercion
pipe to emit native JSON types instead of strings:

- `{{pathParam.id | int}}` -- emits a JSON number (e.g., `42`).
- `{{request.query.price | float}}` -- emits a JSON float (e.g., `19.99`).
- `{{request.query.active | bool}}` -- emits a JSON boolean (e.g., `true`).

Coercion is only meaningful in JSON bodies. In text bodies, everything is
a string and coercion is ignored.

If coercion fails (e.g., `"abc" | int`):
- **Strict mode**: returns HTTP 500 with `X-Httptape-Error: template`.
- **Lenient mode** (default): emits the uncoerced string value.

## Match flow

When synthesis is enabled, the server uses a two-phase match flow:

1. **Phase 1 -- exact match**: all non-exemplar tapes are checked first.
   If an exact match is found, synthesis is not consulted.
2. **Phase 2 -- exemplar fallback**: if no exact match is found, exemplar
   tapes are checked. The request path is tested against each exemplar's
   `url_pattern`. The exemplar must also pass all other configured criteria
   (method, headers, body_fuzzy, etc.).

Among matching exemplars, the highest-scoring one wins. Ties are broken by
declaration order.

## Seed stability

Faker seeds in exemplar tapes are content-derived. For example, the seed
`user-{{pathParam.id}}` evaluates to `user-42` for `/users/42`. This means:

- Renaming the tape file has no effect on generated data.
- Changing the tape ID has no effect.
- The same request always produces the same fake data (deterministic).

## CLI usage

```bash
httptape serve --fixtures ./fixtures --synthesize
```

The `--synthesize` flag enables synthesis mode. Without it, exemplar tapes
in the fixture directory are loaded but ignored.

## Library usage

```go
srv, err := httptape.NewServer(store, httptape.WithSynthesis())
```

## Known limitations

- SSE exemplar tapes are not supported. A follow-up issue can add support.
- Type coercion is limited to `| int`, `| float`, `| bool`.
- Coercion is only effective in JSON response bodies, not text bodies.
