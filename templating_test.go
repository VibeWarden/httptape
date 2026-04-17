package httptape

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestScanTemplateExprs(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []templateExpr
	}{
		{
			name: "no expressions",
			src:  "hello world",
			want: nil,
		},
		{
			name: "single expression",
			src:  "Hello {{request.method}}!",
			want: []templateExpr{
				{raw: "request.method", start: 6, end: 24},
			},
		},
		{
			name: "multiple expressions",
			src:  "{{request.method}} {{request.path}}",
			want: []templateExpr{
				{raw: "request.method", start: 0, end: 18},
				{raw: "request.path", start: 19, end: 35},
			},
		},
		{
			name: "expression with whitespace",
			src:  "{{ request.method }}",
			want: []templateExpr{
				{raw: "request.method", start: 0, end: 20},
			},
		},
		{
			name: "unclosed expression",
			src:  "hello {{request.method",
			want: nil,
		},
		{
			name: "empty expression",
			src:  "hello {{}} world",
			want: []templateExpr{
				{raw: "", start: 6, end: 10},
			},
		},
		{
			name: "nested braces not treated as nested",
			src:  "{{request.headers.X-Key}}",
			want: []templateExpr{
				{raw: "request.headers.X-Key", start: 0, end: 25},
			},
		},
		{
			name: "expression at end of string",
			src:  "prefix {{request.path}}",
			want: []templateExpr{
				{raw: "request.path", start: 7, end: 23},
			},
		},
		{
			name: "expression at start and end",
			src:  "{{request.method}} and {{request.path}}",
			want: []templateExpr{
				{raw: "request.method", start: 0, end: 18},
				{raw: "request.path", start: 23, end: 39},
			},
		},
		{
			name: "only opening braces",
			src:  "{{ no close",
			want: nil,
		},
		{
			name: "only closing braces",
			src:  "no open }}",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanTemplateExprs([]byte(tt.src))
			if len(got) != len(tt.want) {
				t.Fatalf("scanTemplateExprs(%q) returned %d expressions, want %d", tt.src, len(got), len(tt.want))
			}
			for i := range got {
				if got[i].raw != tt.want[i].raw {
					t.Errorf("expr[%d].raw = %q, want %q", i, got[i].raw, tt.want[i].raw)
				}
				if got[i].start != tt.want[i].start {
					t.Errorf("expr[%d].start = %d, want %d", i, got[i].start, tt.want[i].start)
				}
				if got[i].end != tt.want[i].end {
					t.Errorf("expr[%d].end = %d, want %d", i, got[i].end, tt.want[i].end)
				}
			}
		})
	}
}

func TestResolveExpr(t *testing.T) {
	reqBody := []byte(`{"user":{"email":"alice@example.com","age":30},"active":true,"score":99.5}`)
	req := httptest.NewRequest("POST", "/api/users?page=2&sort=name", bytes.NewReader(reqBody))
	req.Header.Set("X-Request-Id", "req-abc-123")
	req.Header.Set("Content-Type", "application/json")

	tests := []struct {
		name    string
		expr    string
		wantVal string
		wantOk  bool
	}{
		{
			name:    "request.method",
			expr:    "request.method",
			wantVal: "POST",
			wantOk:  true,
		},
		{
			name:    "request.path",
			expr:    "request.path",
			wantVal: "/api/users",
			wantOk:  true,
		},
		{
			name:    "request.url",
			expr:    "request.url",
			wantVal: "/api/users?page=2&sort=name",
			wantOk:  true,
		},
		{
			name:    "request.headers existing",
			expr:    "request.headers.X-Request-Id",
			wantVal: "req-abc-123",
			wantOk:  true,
		},
		{
			name:    "request.headers case insensitive",
			expr:    "request.headers.x-request-id",
			wantVal: "req-abc-123",
			wantOk:  true,
		},
		{
			name:    "request.headers missing",
			expr:    "request.headers.X-Missing",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "request.query existing",
			expr:    "request.query.page",
			wantVal: "2",
			wantOk:  true,
		},
		{
			name:    "request.query missing",
			expr:    "request.query.missing",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "request.body string field",
			expr:    "request.body.user.email",
			wantVal: "alice@example.com",
			wantOk:  true,
		},
		{
			name:    "request.body number field",
			expr:    "request.body.user.age",
			wantVal: "30",
			wantOk:  true,
		},
		{
			name:    "request.body boolean field",
			expr:    "request.body.active",
			wantVal: "true",
			wantOk:  true,
		},
		{
			name:    "request.body fractional number",
			expr:    "request.body.score",
			wantVal: "99.5",
			wantOk:  true,
		},
		{
			name:    "request.body non-scalar (object)",
			expr:    "request.body.user",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "request.body missing field",
			expr:    "request.body.nonexistent",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "non-request namespace",
			expr:    "state.counter",
			wantVal: "{{state.counter}}",
			wantOk:  true,
		},
		{
			name:    "request.path.param (unresolvable today)",
			expr:    "request.path.id",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "unknown request sub-key",
			expr:    "request.unknown",
			wantVal: "",
			wantOk:  false,
		},
		{
			name:    "empty expression",
			expr:    "",
			wantVal: "{{}}",
			wantOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotOk := resolveExpr(tt.expr, req, reqBody)
			if gotOk != tt.wantOk {
				t.Errorf("resolveExpr(%q) ok = %v, want %v", tt.expr, gotOk, tt.wantOk)
			}
			if gotVal != tt.wantVal {
				t.Errorf("resolveExpr(%q) = %q, want %q", tt.expr, gotVal, tt.wantVal)
			}
		})
	}
}

func TestResolveExpr_NilBody(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	val, ok := resolveExpr("request.body.field", req, nil)
	if ok {
		t.Errorf("expected unresolvable for nil body, got ok=true, val=%q", val)
	}
}

func TestScalarToString(t *testing.T) {
	tests := []struct {
		name   string
		input  any
		want   string
		wantOk bool
	}{
		{"string", "hello", "hello", true},
		{"integer float", float64(42), "42", true},
		{"fractional float", float64(3.14), "3.14", true},
		{"negative integer", float64(-7), "-7", true},
		{"zero", float64(0), "0", true},
		{"true", true, "true", true},
		{"false", false, "false", true},
		{"nil", nil, "", false},
		{"map", map[string]any{"k": "v"}, "", false},
		{"slice", []any{1, 2}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := scalarToString(tt.input)
			if ok != tt.wantOk {
				t.Errorf("scalarToString(%v) ok = %v, want %v", tt.input, ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("scalarToString(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveTemplateBody(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		method  string
		path    string
		headers http.Header
		reqBody string
		strict  bool
		want    string
		wantErr bool
	}{
		{
			name:   "no templates",
			body:   `{"id": "pay_123"}`,
			method: "GET",
			path:   "/test",
			want:   `{"id": "pay_123"}`,
		},
		{
			name:   "method substitution",
			body:   `{"method": "{{request.method}}"}`,
			method: "POST",
			path:   "/test",
			want:   `{"method": "POST"}`,
		},
		{
			name:   "path substitution",
			body:   `{"path": "{{request.path}}"}`,
			method: "GET",
			path:   "/users/42",
			want:   `{"path": "/users/42"}`,
		},
		{
			name:   "url substitution",
			body:   `{"url": "{{request.url}}"}`,
			method: "GET",
			path:   "/users?page=1",
			want:   `{"url": "/users?page=1"}`,
		},
		{
			name:    "header substitution",
			body:    `{"key": "{{request.headers.X-Request-Id}}"}`,
			method:  "GET",
			path:    "/test",
			headers: http.Header{"X-Request-Id": {"req-123"}},
			want:    `{"key": "req-123"}`,
		},
		{
			name:   "query substitution",
			body:   `{"page": "{{request.query.page}}"}`,
			method: "GET",
			path:   "/test?page=5",
			want:   `{"page": "5"}`,
		},
		{
			name:    "body field substitution",
			body:    `{"echo_email": "{{request.body.user.email}}"}`,
			method:  "POST",
			path:    "/test",
			reqBody: `{"user":{"email":"bob@example.com"}}`,
			want:    `{"echo_email": "bob@example.com"}`,
		},
		{
			name:   "multiple substitutions",
			body:   `{"method":"{{request.method}}","path":"{{request.path}}"}`,
			method: "PUT",
			path:   "/api/resource",
			want:   `{"method":"PUT","path":"/api/resource"}`,
		},
		{
			name:   "lenient unresolvable replaces with empty",
			body:   `{"val": "{{request.headers.Missing}}"}`,
			method: "GET",
			path:   "/test",
			strict: false,
			want:   `{"val": ""}`,
		},
		{
			name:    "strict unresolvable returns error",
			body:    `{"val": "{{request.headers.Missing}}"}`,
			method:  "GET",
			path:    "/test",
			strict:  true,
			wantErr: true,
		},
		{
			name:   "non-request namespace left as literal",
			body:   `{"counter": "{{state.counter}}"}`,
			method: "GET",
			path:   "/test",
			want:   `{"counter": "{{state.counter}}"}`,
		},
		{
			name:   "mixed resolvable and literal namespace",
			body:   `{{request.method}} {{state.x}}`,
			method: "GET",
			path:   "/test",
			want:   `GET {{state.x}}`,
		},
		{
			name:    "idempotency key echo (headline example)",
			body:    `{"id": "pay_123", "idempotency_key": "{{request.headers.Idempotency-Key}}"}`,
			method:  "POST",
			path:    "/payments",
			headers: http.Header{"Idempotency-Key": {"idem-abc-789"}},
			want:    `{"id": "pay_123", "idempotency_key": "idem-abc-789"}`,
		},
		{
			name:   "nil body returns nil",
			body:   "",
			method: "GET",
			path:   "/test",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tt.reqBody != "" {
				bodyReader = strings.NewReader(tt.reqBody)
			}
			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			for k, vs := range tt.headers {
				for _, v := range vs {
					req.Header.Add(k, v)
				}
			}

			var bodyBytes []byte
			if tt.body != "" {
				bodyBytes = []byte(tt.body)
			}

			got, err := ResolveTemplateBody(bodyBytes, req, tt.strict)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestResolveTemplateBody_NilBody(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	got, err := ResolveTemplateBody(nil, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %q", string(got))
	}
}

func TestResolveTemplateBody_FastPath_NoDelimiters(t *testing.T) {
	body := []byte(`{"no":"templates","here":true}`)
	req := httptest.NewRequest("GET", "/test", nil)
	got, err := ResolveTemplateBody(body, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Fast path should return the exact same slice.
	if &got[0] != &body[0] {
		t.Error("fast path should return the same slice, got a copy")
	}
}

func TestResolveTemplateHeaders(t *testing.T) {
	tests := []struct {
		name       string
		headers    http.Header
		method     string
		path       string
		reqHeaders http.Header
		reqBody    string
		strict     bool
		want       http.Header
		wantErr    bool
	}{
		{
			name:    "no templates in headers",
			headers: http.Header{"Content-Type": {"application/json"}},
			method:  "GET",
			path:    "/test",
			want:    http.Header{"Content-Type": {"application/json"}},
		},
		{
			name:       "header with template",
			headers:    http.Header{"X-Echo": {"{{request.headers.X-Request-Id}}"}},
			method:     "GET",
			path:       "/test",
			reqHeaders: http.Header{"X-Request-Id": {"id-42"}},
			want:       http.Header{"X-Echo": {"id-42"}},
		},
		{
			name:    "header with method template",
			headers: http.Header{"X-Method": {"{{request.method}}"}},
			method:  "DELETE",
			path:    "/test",
			want:    http.Header{"X-Method": {"DELETE"}},
		},
		{
			name:    "multiple header values",
			headers: http.Header{"X-Multi": {"{{request.method}}", "static"}},
			method:  "GET",
			path:    "/test",
			want:    http.Header{"X-Multi": {"GET", "static"}},
		},
		{
			name:    "lenient mode missing ref",
			headers: http.Header{"X-Key": {"{{request.headers.Missing}}"}},
			method:  "GET",
			path:    "/test",
			strict:  false,
			want:    http.Header{"X-Key": {""}},
		},
		{
			name:    "strict mode missing ref",
			headers: http.Header{"X-Key": {"{{request.headers.Missing}}"}},
			method:  "GET",
			path:    "/test",
			strict:  true,
			wantErr: true,
		},
		{
			name:   "nil headers returns nil",
			method: "GET",
			path:   "/test",
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tt.reqBody != "" {
				bodyReader = strings.NewReader(tt.reqBody)
			}
			req := httptest.NewRequest(tt.method, tt.path, bodyReader)
			for k, vs := range tt.reqHeaders {
				for _, v := range vs {
					req.Header.Add(k, v)
				}
			}

			got, err := ResolveTemplateHeaders(tt.headers, req, tt.strict)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}

			for key, wantVals := range tt.want {
				gotVals := got[key]
				if len(gotVals) != len(wantVals) {
					t.Errorf("header %s: got %d values, want %d", key, len(gotVals), len(wantVals))
					continue
				}
				for i := range wantVals {
					if gotVals[i] != wantVals[i] {
						t.Errorf("header %s[%d] = %q, want %q", key, i, gotVals[i], wantVals[i])
					}
				}
			}
		})
	}
}

func TestResolveTemplateBody_StrictErrorMessage(t *testing.T) {
	body := []byte(`prefix {{request.headers.Missing}} suffix`)
	req := httptest.NewRequest("GET", "/test", nil)

	_, err := ResolveTemplateBody(body, req, true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "request.headers.Missing") {
		t.Errorf("error message %q should mention the failed expression", err.Error())
	}
	if !strings.Contains(err.Error(), "httptape") {
		t.Errorf("error message %q should include package prefix", err.Error())
	}
}

func TestResolveTemplateBody_BodyFieldTypes(t *testing.T) {
	reqBody := `{"s":"hello","n":42,"f":3.14,"b":true,"null":null,"obj":{"k":"v"},"arr":[1,2]}`

	tests := []struct {
		name   string
		expr   string
		want   string
		strict bool
	}{
		{"string field", "{{request.body.s}}", "hello", false},
		{"integer field", "{{request.body.n}}", "42", false},
		{"float field", "{{request.body.f}}", "3.14", false},
		{"boolean field", "{{request.body.b}}", "true", false},
		{"null field lenient", "{{request.body.null}}", "", false},
		{"object field lenient", "{{request.body.obj}}", "", false},
		{"array field lenient", "{{request.body.arr}}", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
			got, err := ResolveTemplateBody([]byte(tt.expr), req, tt.strict)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("got %q, want %q", string(got), tt.want)
			}
		})
	}
}

func TestResolveTemplateBody_NonScalarStrict(t *testing.T) {
	reqBody := `{"obj":{"k":"v"},"arr":[1,2],"null":null}`

	tests := []struct {
		name string
		expr string
	}{
		{"object in strict", "{{request.body.obj}}"},
		{"array in strict", "{{request.body.arr}}"},
		{"null in strict", "{{request.body.null}}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", strings.NewReader(reqBody))
			_, err := ResolveTemplateBody([]byte(tt.expr), req, true)
			if err == nil {
				t.Fatal("expected error for non-scalar in strict mode")
			}
		})
	}
}

func TestResolveTemplateBody_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest("POST", "/test", strings.NewReader("not json"))
	body := []byte(`echo: {{request.body.field}}`)

	// Lenient mode: invalid JSON body -> unresolvable -> empty string.
	got, err := ResolveTemplateBody(body, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "echo: " {
		t.Errorf("got %q, want %q", string(got), "echo: ")
	}
}

func TestResolveTemplateBody_PreservesRequestBody(t *testing.T) {
	originalBody := `{"key":"value"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(originalBody))

	body := []byte(`{{request.body.key}}`)
	_, err := ResolveTemplateBody(body, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Request body should still be readable.
	remaining, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("reading body after templating: %v", err)
	}
	if string(remaining) != originalBody {
		t.Errorf("request body after templating = %q, want %q", string(remaining), originalBody)
	}
}

func TestResolveBodyField(t *testing.T) {
	body := []byte(`{"user":{"email":"a@b.com","name":"Alice"},"count":3}`)

	tests := []struct {
		name   string
		path   string
		want   string
		wantOk bool
	}{
		{"top level string", "count", "3", true},
		{"nested string", "user.email", "a@b.com", true},
		{"nested name", "user.name", "Alice", true},
		{"missing field", "user.phone", "", false},
		{"non-scalar object", "user", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveBodyField(body, tt.path)
			if ok != tt.wantOk {
				t.Errorf("resolveBodyField(%q) ok = %v, want %v", tt.path, ok, tt.wantOk)
			}
			if got != tt.want {
				t.Errorf("resolveBodyField(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestResolveBodyField_EmptyBody(t *testing.T) {
	got, ok := resolveBodyField(nil, "field")
	if ok {
		t.Errorf("expected false for nil body, got true with value %q", got)
	}

	got2, ok2 := resolveBodyField([]byte{}, "field")
	if ok2 {
		t.Errorf("expected false for empty body, got true with value %q", got2)
	}
}

func TestResolveBodyField_InvalidJSON(t *testing.T) {
	got, ok := resolveBodyField([]byte("not-json"), "field")
	if ok {
		t.Errorf("expected false for invalid JSON, got true with value %q", got)
	}
}

func TestResolveBodyField_InvalidPath(t *testing.T) {
	// parsePath rejects paths without the $. prefix -- we prepend it,
	// but a completely empty path segment should fail.
	got, ok := resolveBodyField([]byte(`{"a":"b"}`), "")
	if ok {
		t.Errorf("expected false for empty path, got true with value %q", got)
	}
}

func TestReadRequestBody(t *testing.T) {
	t.Run("truly nil body", func(t *testing.T) {
		req := &http.Request{Body: nil}
		got := readRequestBody(req)
		if got != nil {
			t.Errorf("expected nil for truly nil body, got %q", string(got))
		}
	})

	t.Run("httptest nil body", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		got := readRequestBody(req)
		// httptest.NewRequest with nil body sets Body to http.NoBody,
		// which reads as empty. readRequestBody returns an empty slice.
		if len(got) != 0 {
			t.Errorf("expected empty, got %q", string(got))
		}
	})

	t.Run("non-nil body", func(t *testing.T) {
		original := "hello body"
		req := httptest.NewRequest("POST", "/test", strings.NewReader(original))
		got := readRequestBody(req)
		if string(got) != original {
			t.Errorf("got %q, want %q", string(got), original)
		}
		// Verify body is restored.
		restored, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("re-read error: %v", err)
		}
		if string(restored) != original {
			t.Errorf("restored body = %q, want %q", string(restored), original)
		}
	})
}

func TestResolveTemplateHeaders_EmptyHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	got, err := ResolveTemplateHeaders(http.Header{}, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty header map, got %v", got)
	}
}

func TestResolveTemplateBody_QueryMultipleValues(t *testing.T) {
	// Query().Get returns the first value.
	req := httptest.NewRequest("GET", "/test?color=red&color=blue", nil)
	body := []byte(`{{request.query.color}}`)
	got, err := ResolveTemplateBody(body, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "red" {
		t.Errorf("got %q, want %q (first value)", string(got), "red")
	}
}

func TestResolveTemplateBody_HeaderCaseInsensitive(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Custom-Header", "value-123")

	// Template uses different casing.
	body := []byte(`{{request.headers.x-custom-header}}`)
	got, err := ResolveTemplateBody(body, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "value-123" {
		t.Errorf("got %q, want %q", string(got), "value-123")
	}
}

func TestResolveTemplateBody_AdjacentExpressions(t *testing.T) {
	req := httptest.NewRequest("GET", "/path", nil)
	body := []byte(`{{request.method}}{{request.path}}`)
	got, err := ResolveTemplateBody(body, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "GET/path" {
		t.Errorf("got %q, want %q", string(got), "GET/path")
	}
}

func TestResolveTemplateBody_MixedResolvableAndUnresolvable(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	body := []byte(`a={{request.method}},b={{request.headers.Missing}},c={{request.path}}`)

	// Lenient mode.
	got, err := ResolveTemplateBody(body, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "a=GET,b=,c=/test" {
		t.Errorf("got %q, want %q", string(got), "a=GET,b=,c=/test")
	}
}

func TestResolveTemplateBody_UnclosedDelimiterTreatedAsLiteral(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	body := []byte(`hello {{request.method}} and {{ unclosed`)
	got, err := ResolveTemplateBody(body, req, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The {{ unclosed part stays as literal since there's no closing }}.
	if string(got) != "hello GET and {{ unclosed" {
		t.Errorf("got %q, want %q", string(got), "hello GET and {{ unclosed")
	}
}
