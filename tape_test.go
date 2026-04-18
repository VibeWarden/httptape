package httptape

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestNewTape(t *testing.T) {
	req := RecordedReq{
		Method: "GET",
		URL:    "http://example.com",
	}
	resp := RecordedResp{
		StatusCode: 200,
	}

	tape := NewTape("test-route", req, resp)

	if tape.ID == "" {
		t.Error("NewTape().ID is empty, want UUID")
	}
	if tape.Route != "test-route" {
		t.Errorf("NewTape().Route = %q, want %q", tape.Route, "test-route")
	}
	if tape.RecordedAt.IsZero() {
		t.Error("NewTape().RecordedAt is zero, want current time")
	}
	if tape.Request.Method != "GET" {
		t.Errorf("NewTape().Request.Method = %q, want %q", tape.Request.Method, "GET")
	}
	if tape.Response.StatusCode != 200 {
		t.Errorf("NewTape().Response.StatusCode = %d, want %d", tape.Response.StatusCode, 200)
	}
}

func TestNewTape_UniqueIDs(t *testing.T) {
	req := RecordedReq{Method: "GET", URL: "http://example.com"}
	resp := RecordedResp{StatusCode: 200}

	tape1 := NewTape("route", req, resp)
	tape2 := NewTape("route", req, resp)

	if tape1.ID == tape2.ID {
		t.Errorf("NewTape() produced duplicate IDs: %q", tape1.ID)
	}
}

func TestNewUUID_Format(t *testing.T) {
	uuid := newUUID()

	// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		t.Fatalf("UUID %q has %d parts, want 5", uuid, len(parts))
	}

	expectedLens := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLens[i] {
			t.Errorf("UUID part %d = %q (len %d), want len %d", i, part, len(part), expectedLens[i])
		}
	}

	// Version nibble must be '4'.
	if parts[2][0] != '4' {
		t.Errorf("UUID version nibble = %c, want '4'", parts[2][0])
	}

	// Variant nibble must be 8, 9, a, or b.
	variant := parts[3][0]
	if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
		t.Errorf("UUID variant nibble = %c, want one of 8, 9, a, b", variant)
	}
}

func TestBodyHashFromBytes(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "nil input",
			input: nil,
			want:  "",
		},
		{
			name:  "empty input",
			input: []byte{},
			want:  "",
		},
		{
			name:  "non-empty input",
			input: []byte("hello world"),
			// SHA-256 of "hello world"
			want: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BodyHashFromBytes(tt.input)
			if got != tt.want {
				t.Errorf("BodyHashFromBytes() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBodyHashFromBytes_Deterministic(t *testing.T) {
	input := []byte("deterministic test input")
	hash1 := BodyHashFromBytes(input)
	hash2 := BodyHashFromBytes(input)

	if hash1 != hash2 {
		t.Errorf("BodyHashFromBytes() not deterministic: %q != %q", hash1, hash2)
	}
}

// --- ADR-41: Body marshal/unmarshal tests ---

func TestRecordedReq_MarshalJSON_JSONBody(t *testing.T) {
	r := RecordedReq{
		Method:  "POST",
		URL:     "http://example.com/api",
		Headers: http.Header{"Content-Type": {"application/json"}},
		Body:    []byte(`{"key":"value"}`),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// The body should appear as native JSON, not a string.
	if !strings.Contains(string(data), `"body":{"key":"value"}`) {
		t.Errorf("expected native JSON body, got: %s", data)
	}
}

func TestRecordedReq_MarshalJSON_TextBody(t *testing.T) {
	r := RecordedReq{
		Method:  "POST",
		URL:     "http://example.com/api",
		Headers: http.Header{"Content-Type": {"text/plain"}},
		Body:    []byte("hello world"),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"body":"hello world"`) {
		t.Errorf("expected text string body, got: %s", data)
	}
}

func TestRecordedReq_MarshalJSON_BinaryBody(t *testing.T) {
	r := RecordedReq{
		Method:  "GET",
		URL:     "http://example.com/image",
		Headers: http.Header{"Content-Type": {"image/png"}},
		Body:    []byte{0x89, 0x50, 0x4E, 0x47},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Should be base64 string "iVBORw==" for {0x89, 0x50, 0x4E, 0x47}.
	if !strings.Contains(string(data), `"body":"iVBORw=="`) {
		t.Errorf("expected base64 body, got: %s", data)
	}
}

func TestRecordedReq_MarshalJSON_NilBody(t *testing.T) {
	r := RecordedReq{
		Method:  "GET",
		URL:     "http://example.com/api",
		Headers: http.Header{"Content-Type": {"application/json"}},
		Body:    nil,
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"body":null`) {
		t.Errorf("expected null body, got: %s", data)
	}
}

func TestRecordedReq_MarshalJSON_EmptyBody(t *testing.T) {
	r := RecordedReq{
		Method:  "GET",
		URL:     "http://example.com/api",
		Headers: http.Header{"Content-Type": {"application/json"}},
		Body:    []byte{},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"body":null`) {
		t.Errorf("expected null body for empty slice, got: %s", data)
	}
}

func TestRecordedReq_MarshalJSON_InvalidJSONWithJSONCT(t *testing.T) {
	r := RecordedReq{
		Method:  "POST",
		URL:     "http://example.com/api",
		Headers: http.Header{"Content-Type": {"application/json"}},
		Body:    []byte("not valid json {{{"),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Should fall back to base64 since the body is not valid JSON.
	if strings.Contains(string(data), `"body":"not valid json`) {
		t.Errorf("expected base64 fallback, but got raw text: %s", data)
	}
	// Should still have a "body" field.
	if !strings.Contains(string(data), `"body":`) {
		t.Errorf("missing body field: %s", data)
	}
}

func TestRecordedReq_MarshalJSON_VendorJSON(t *testing.T) {
	r := RecordedReq{
		Method:  "POST",
		URL:     "http://example.com/api",
		Headers: http.Header{"Content-Type": {"application/vnd.api+json"}},
		Body:    []byte(`{"data":{"type":"users","id":"1"}}`),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Should emit native JSON for vendor +json types.
	if !strings.Contains(string(data), `"body":{"data":{"type":"users","id":"1"}}`) {
		t.Errorf("expected native JSON body for vendor +json, got: %s", data)
	}
}

func TestRecordedResp_MarshalJSON_JSONBody(t *testing.T) {
	r := RecordedResp{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": {"application/json"}},
		Body:       []byte(`[1,2,3]`),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"body":[1,2,3]`) {
		t.Errorf("expected native JSON array body, got: %s", data)
	}
}

func TestRecordedResp_MarshalJSON_NilBodyWithSSE(t *testing.T) {
	r := RecordedResp{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": {"text/event-stream"}},
		Body:       nil,
		SSEEvents:  []SSEEvent{{Data: "hello"}},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if !strings.Contains(string(data), `"body":null`) {
		t.Errorf("expected null body for SSE tape, got: %s", data)
	}
	if !strings.Contains(string(data), `"sse_events"`) {
		t.Errorf("expected sse_events field, got: %s", data)
	}
}

func TestRecordedReq_MarshalJSON_MissingCT(t *testing.T) {
	r := RecordedReq{
		Method: "POST",
		URL:    "http://example.com/api",
		Body:   []byte("some data"),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// No Content-Type -> binary -> base64.
	if strings.Contains(string(data), `"body":"some data"`) {
		t.Errorf("expected base64, got raw text: %s", data)
	}
}

// --- Unmarshal tests ---

func TestRecordedReq_UnmarshalJSON_NativeJSON(t *testing.T) {
	input := `{
		"method": "POST",
		"url": "http://example.com/api",
		"headers": {"Content-Type": ["application/json"]},
		"body": {"key": "value"},
		"body_hash": ""
	}`

	var r RecordedReq
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	// Body should be compact JSON bytes.
	want := `{"key":"value"}`
	if string(r.Body) != want {
		t.Errorf("Body = %q, want %q", string(r.Body), want)
	}
}

func TestRecordedReq_UnmarshalJSON_StringText(t *testing.T) {
	input := `{
		"method": "POST",
		"url": "http://example.com/api",
		"headers": {"Content-Type": ["text/plain"]},
		"body": "hello world",
		"body_hash": ""
	}`

	var r RecordedReq
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if string(r.Body) != "hello world" {
		t.Errorf("Body = %q, want %q", string(r.Body), "hello world")
	}
}

func TestRecordedReq_UnmarshalJSON_Base64Binary(t *testing.T) {
	// AQID is base64 for []byte{1, 2, 3}
	input := `{
		"method": "GET",
		"url": "http://example.com/image",
		"headers": {"Content-Type": ["image/png"]},
		"body": "AQID",
		"body_hash": ""
	}`

	var r RecordedReq
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	want := []byte{1, 2, 3}
	if !bytes.Equal(r.Body, want) {
		t.Errorf("Body = %v, want %v", r.Body, want)
	}
}

func TestRecordedReq_UnmarshalJSON_Null(t *testing.T) {
	input := `{
		"method": "GET",
		"url": "http://example.com/api",
		"headers": {},
		"body": null,
		"body_hash": ""
	}`

	var r RecordedReq
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if r.Body != nil {
		t.Errorf("Body = %v, want nil", r.Body)
	}
}

func TestRecordedReq_UnmarshalJSON_LegacyBase64JSON(t *testing.T) {
	// eyJrIjoidiJ9 is base64 for {"k":"v"}
	input := `{
		"method": "POST",
		"url": "http://example.com/api",
		"headers": {"Content-Type": ["application/json"]},
		"body": "eyJrIjoidiJ9",
		"body_hash": ""
	}`

	var r RecordedReq
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	want := `{"k":"v"}`
	if string(r.Body) != want {
		t.Errorf("Body = %q, want %q", string(r.Body), want)
	}
}

func TestRecordedReq_UnmarshalJSON_LegacyBodyEncoding(t *testing.T) {
	// Legacy fixtures may have body_encoding field -- it should be silently ignored.
	input := `{
		"method": "POST",
		"url": "http://example.com/api",
		"headers": {"Content-Type": ["application/json"]},
		"body": {"ok": true},
		"body_hash": "",
		"body_encoding": "identity"
	}`

	var r RecordedReq
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	want := `{"ok":true}`
	if string(r.Body) != want {
		t.Errorf("Body = %q, want %q", string(r.Body), want)
	}
}

func TestRecordedReq_UnmarshalJSON_NativeJSONArray(t *testing.T) {
	input := `{
		"method": "POST",
		"url": "http://example.com/api",
		"headers": {"Content-Type": ["application/json"]},
		"body": [1, 2, 3],
		"body_hash": ""
	}`

	var r RecordedReq
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	want := `[1,2,3]`
	if string(r.Body) != want {
		t.Errorf("Body = %q, want %q", string(r.Body), want)
	}
}

// --- Round-trip tests ---

func TestRoundTrip_JSONBody(t *testing.T) {
	r := RecordedReq{
		Method:  "POST",
		URL:     "http://example.com/api",
		Headers: http.Header{"Content-Type": {"application/json"}},
		Body:    []byte(`{"key":"value","nested":{"a":1}}`),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var r2 RecordedReq
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !bytes.Equal(r.Body, r2.Body) {
		t.Errorf("round-trip body mismatch:\n  original: %q\n  result:   %q", string(r.Body), string(r2.Body))
	}
}

func TestRoundTrip_TextBody(t *testing.T) {
	r := RecordedReq{
		Method:  "POST",
		URL:     "http://example.com/api",
		Headers: http.Header{"Content-Type": {"text/plain"}},
		Body:    []byte("hello world"),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var r2 RecordedReq
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !bytes.Equal(r.Body, r2.Body) {
		t.Errorf("round-trip body mismatch:\n  original: %q\n  result:   %q", string(r.Body), string(r2.Body))
	}
}

func TestRoundTrip_BinaryBody(t *testing.T) {
	r := RecordedReq{
		Method:  "GET",
		URL:     "http://example.com/image",
		Headers: http.Header{"Content-Type": {"image/png"}},
		Body:    []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var r2 RecordedReq
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !bytes.Equal(r.Body, r2.Body) {
		t.Errorf("round-trip body mismatch:\n  original: %v\n  result:   %v", r.Body, r2.Body)
	}
}

func TestRoundTrip_NilBody(t *testing.T) {
	r := RecordedReq{
		Method:  "GET",
		URL:     "http://example.com/api",
		Headers: http.Header{},
		Body:    nil,
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var r2 RecordedReq
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if r2.Body != nil {
		t.Errorf("round-trip nil body: got %v, want nil", r2.Body)
	}
}

func TestRoundTrip_Resp_JSONBody(t *testing.T) {
	r := RecordedResp{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": {"application/json"}},
		Body:       []byte(`{"users":[{"id":1,"name":"Alice"}]}`),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var r2 RecordedResp
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if !bytes.Equal(r.Body, r2.Body) {
		t.Errorf("round-trip body mismatch:\n  original: %q\n  result:   %q", string(r.Body), string(r2.Body))
	}
}

func TestRecordedReq_UnmarshalJSON_FormUrlencoded(t *testing.T) {
	input := `{
		"method": "POST",
		"url": "http://example.com/login",
		"headers": {"Content-Type": ["application/x-www-form-urlencoded"]},
		"body": "username=alice&password=secret",
		"body_hash": ""
	}`

	var r RecordedReq
	if err := json.Unmarshal([]byte(input), &r); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	want := "username=alice&password=secret"
	if string(r.Body) != want {
		t.Errorf("Body = %q, want %q", string(r.Body), want)
	}
}

func TestRecordedReq_MarshalJSON_FormUrlencoded(t *testing.T) {
	r := RecordedReq{
		Method:  "POST",
		URL:     "http://example.com/login",
		Headers: http.Header{"Content-Type": {"application/x-www-form-urlencoded"}},
		Body:    []byte("username=alice&password=secret"),
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Form data is classified as text, should be a JSON string.
	if !strings.Contains(string(data), `"body":"username=alice\u0026password=secret"`) &&
		!strings.Contains(string(data), `"body":"username=alice&password=secret"`) {
		t.Errorf("expected text string body for form data, got: %s", data)
	}
}
