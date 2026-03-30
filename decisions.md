# httptape — Decisions Log

This file is the living record of all architectural decisions.
Updated by the architect agent (ADRs).
Never delete entries — mark superseded decisions as `Superseded by ADR-N`.

---

## Locked decisions (from project inception)

| # | Decision | Status |
|---|---|---|
| L-01 | Language: Go | Locked |
| L-02 | License: Apache 2.0 | Locked |
| L-03 | Architecture: Hexagonal (ports & adapters) within single package | Locked |
| L-04 | Dependencies: stdlib only for v1 | Locked |
| L-05 | Sanitization: on write, not on export | Locked |
| L-06 | Deterministic faking: HMAC-SHA256 with configurable seed | Locked |
| L-07 | Fixture format: JSON | Locked |
| L-08 | Storage: pluggable interface (Store) | Locked |
| L-09 | Matching: progressive (exact first, fuzzy/regex later) | Locked |
| L-10 | Recording: async by default via buffered channel | Locked |
| L-11 | No init(), no package-level mutable state, no panics | Locked |
| L-12 | Functional options pattern for all public constructors | Locked |
| L-13 | 90% test coverage target, stdlib testing only | Locked |

---

## ADRs

(Architect agent appends ADRs here as issues are designed.)
