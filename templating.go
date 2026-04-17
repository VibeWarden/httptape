package httptape

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// templateExpr represents a parsed template expression extracted from a
// Mustache-style {{...}} placeholder. It stores the raw expression text
// and its byte offsets within the source.
type templateExpr struct {
	// raw is the expression text between {{ and }}, trimmed of whitespace.
	raw string
	// start is the byte offset of the opening "{{" in the source.
	start int
	// end is the byte offset just past the closing "}}" in the source.
	end int
}

// scanTemplateExprs scans src for all {{...}} expressions and returns them
// in order of appearance. Nested or malformed delimiters ({{ without }})
// are left as literal text. This is the "hand-rolled scanner" approach:
// a single bytes.Index loop, ~80 lines, no regex.
func scanTemplateExprs(src []byte) []templateExpr {
	var exprs []templateExpr
	pos := 0
	for pos < len(src) {
		openIdx := bytes.Index(src[pos:], []byte("{{"))
		if openIdx < 0 {
			break
		}
		openIdx += pos // absolute offset

		closeIdx := bytes.Index(src[openIdx+2:], []byte("}}"))
		if closeIdx < 0 {
			break // unclosed {{ — treat rest as literal
		}
		closeIdx += openIdx + 2 // absolute offset of first char of "}}"

		raw := string(bytes.TrimSpace(src[openIdx+2 : closeIdx]))
		exprs = append(exprs, templateExpr{
			raw:   raw,
			start: openIdx,
			end:   closeIdx + 2, // past the "}}"
		})
		pos = closeIdx + 2
	}
	return exprs
}

// resolveExpr resolves a single template expression against the incoming
// HTTP request. It returns the resolved string value and true, or ("", false)
// if the expression is unresolvable.
//
// Supported expression prefixes:
//   - request.method        -> r.Method
//   - request.path          -> r.URL.Path
//   - request.url           -> r.URL.String()
//   - request.headers.<N>   -> r.Header.Get(http.CanonicalHeaderKey(N))
//   - request.query.<N>     -> r.URL.Query().Get(N)
//   - request.body.<path>   -> JSON field at $.path (dot-separated)
//
// Non-"request." prefixed expressions are left as literal text (returned
// as-is with ok=true). This is forward-compatible with future namespaces
// like "state.*" (#46).
func resolveExpr(expr string, r *http.Request, reqBody []byte) (string, bool) {
	// Non-request namespace: leave as literal (forward-compat).
	if len(expr) < 8 || expr[:8] != "request." {
		return "{{" + expr + "}}", true
	}

	rest := expr[8:] // strip "request."
	switch {
	case rest == "method":
		return r.Method, true
	case rest == "path":
		return r.URL.Path, true
	case rest == "url":
		return r.URL.String(), true
	}

	// request.headers.<Name>
	if len(rest) > 8 && rest[:8] == "headers." {
		name := rest[8:]
		val := r.Header.Get(http.CanonicalHeaderKey(name))
		if val == "" {
			// Header not present — unresolvable.
			return "", false
		}
		return val, true
	}

	// request.query.<name>
	if len(rest) > 6 && rest[:6] == "query." {
		name := rest[6:]
		val := r.URL.Query().Get(name)
		if val == "" {
			return "", false
		}
		return val, true
	}

	// request.body.<json.path>
	if len(rest) > 5 && rest[:5] == "body." {
		jsonPath := rest[5:]
		return resolveBodyField(reqBody, jsonPath)
	}

	// request.path.<param> — unresolvable today (no path-template matcher).
	// Any other request.* sub-key is also unresolvable.
	return "", false
}

// resolveBodyField extracts a scalar value from a JSON body using dot-notation.
// The dotPath is prepended with "$." and parsed via parsePath from sanitizer.go,
// then extracted via extractAtPath from matcher.go.
//
// Only scalar values (string, number, bool) are converted to strings. Objects
// and arrays return ("", false) as they cannot be meaningfully interpolated.
func resolveBodyField(body []byte, dotPath string) (string, bool) {
	if len(body) == 0 {
		return "", false
	}

	pp, ok := parsePath("$." + dotPath)
	if !ok {
		return "", false
	}

	var data any
	if err := json.Unmarshal(body, &data); err != nil {
		return "", false
	}

	val, ok := extractAtPath(data, pp.segments)
	if !ok {
		return "", false
	}

	return scalarToString(val)
}

// scalarToString converts a JSON scalar value to its string representation.
// Returns ("", false) for non-scalar types (nil, objects, arrays).
func scalarToString(v any) (string, bool) {
	switch val := v.(type) {
	case string:
		return val, true
	case float64:
		// Use strconv for clean formatting (no trailing zeros).
		if val == float64(int64(val)) {
			return strconv.FormatInt(int64(val), 10), true
		}
		return strconv.FormatFloat(val, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(val), true
	default:
		// nil, map[string]any, []any — not scalar.
		return "", false
	}
}

// ResolveTemplateBody resolves all {{request.*}} template expressions in body
// against the incoming HTTP request. It returns the resolved body bytes.
//
// If strict is true, any unresolvable expression causes an error describing
// which expression failed. If strict is false (lenient mode), unresolvable
// expressions are replaced with an empty string.
//
// Non-"request." expressions (e.g., {{state.counter}}) are left as literal
// text in both modes, providing forward-compatibility for future namespaces.
//
// If body contains no "{{" sequence, it is returned unchanged with zero
// allocations (fast path).
//
// ResolveTemplateBody is safe for concurrent use — it is a pure function
// with no shared mutable state. The request body is read once and cached
// in memory for the duration of the call.
func ResolveTemplateBody(body []byte, r *http.Request, strict bool) ([]byte, error) {
	// Fast path: no template delimiters at all.
	if !bytes.Contains(body, []byte("{{")) {
		return body, nil
	}

	reqBody := readRequestBody(r)

	exprs := scanTemplateExprs(body)
	if len(exprs) == 0 {
		return body, nil
	}

	// Build result by walking through expressions in order.
	var buf bytes.Buffer
	buf.Grow(len(body))
	prev := 0

	for _, expr := range exprs {
		// Copy literal text before this expression.
		buf.Write(body[prev:expr.start])

		resolved, ok := resolveExpr(expr.raw, r, reqBody)
		if !ok {
			if strict {
				return nil, fmt.Errorf("httptape: unresolvable template expression: {{%s}}", expr.raw)
			}
			// Lenient: replace with empty string (write nothing).
		} else {
			buf.WriteString(resolved)
		}
		prev = expr.end
	}
	// Copy trailing literal text.
	buf.Write(body[prev:])

	return buf.Bytes(), nil
}

// ResolveTemplateHeaders resolves all {{request.*}} template expressions in
// response header values against the incoming HTTP request. It returns a new
// http.Header map with resolved values.
//
// If strict is true, any unresolvable expression causes an error. If strict
// is false (lenient mode), unresolvable expressions are replaced with an
// empty string.
//
// Non-"request." expressions are left as literal text (forward-compat).
//
// Headers that contain no "{{" sequences are copied as-is (fast path per
// header value).
//
// ResolveTemplateHeaders is safe for concurrent use — it is a pure function
// with no shared mutable state.
func ResolveTemplateHeaders(h http.Header, r *http.Request, strict bool) (http.Header, error) {
	if h == nil {
		return nil, nil
	}

	reqBody := readRequestBody(r)
	result := make(http.Header, len(h))

	for key, values := range h {
		resolved := make([]string, len(values))
		for i, v := range values {
			rv, err := resolveTemplateString(v, r, reqBody, strict)
			if err != nil {
				return nil, err
			}
			resolved[i] = rv
		}
		result[key] = resolved
	}

	return result, nil
}

// resolveTemplateString resolves all {{...}} expressions in a single string.
// Used by ResolveTemplateHeaders for individual header values.
func resolveTemplateString(s string, r *http.Request, reqBody []byte, strict bool) (string, error) {
	// Fast path: no delimiters.
	if !bytes.Contains([]byte(s), []byte("{{")) {
		return s, nil
	}

	src := []byte(s)
	exprs := scanTemplateExprs(src)
	if len(exprs) == 0 {
		return s, nil
	}

	var buf bytes.Buffer
	buf.Grow(len(s))
	prev := 0

	for _, expr := range exprs {
		buf.Write(src[prev:expr.start])

		resolved, ok := resolveExpr(expr.raw, r, reqBody)
		if !ok {
			if strict {
				return "", fmt.Errorf("httptape: unresolvable template expression: {{%s}}", expr.raw)
			}
			// Lenient: empty string.
		} else {
			buf.WriteString(resolved)
		}
		prev = expr.end
	}
	buf.Write(src[prev:])

	return buf.String(), nil
}

// readRequestBody reads the full request body and restores it so the body
// remains readable by downstream handlers. Returns nil if the body is nil
// or empty.
func readRequestBody(r *http.Request) []byte {
	if r.Body == nil {
		return nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil
	}
	// Restore the body for any downstream readers.
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body
}
