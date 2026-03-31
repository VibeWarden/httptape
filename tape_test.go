package httptape

import (
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
