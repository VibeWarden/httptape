package httptape

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

// Tape represents a single recorded HTTP interaction (request + response pair).
// It is a pure value type with no I/O.
type Tape struct {
	// ID uniquely identifies this tape. Generated as a UUID v4 string on creation.
	ID string `json:"id"`

	// Route is a logical grouping label (e.g., "users-api", "auth-service").
	// Used by FileStore for directory partitioning and by matchers for scoping.
	Route string `json:"route"`

	// RecordedAt is the UTC timestamp when the interaction was captured.
	RecordedAt time.Time `json:"recorded_at"`

	// Request is the recorded HTTP request.
	Request RecordedReq `json:"request"`

	// Response is the recorded HTTP response.
	Response RecordedResp `json:"response"`
}

// RecordedReq captures the essential parts of an HTTP request for matching and replay.
type RecordedReq struct {
	// Method is the HTTP method (GET, POST, etc.).
	Method string `json:"method"`

	// URL is the full request URL as a string.
	URL string `json:"url"`

	// Headers contains the request headers. Only non-sensitive headers are stored
	// after sanitization (handled by the sanitizer, not by this type).
	Headers http.Header `json:"headers"`

	// Body is the full request body bytes. May be nil for bodiless requests.
	Body []byte `json:"body"`

	// BodyHash is a hex-encoded SHA-256 hash of the original request body.
	// Used for matching without comparing full bodies.
	BodyHash string `json:"body_hash"`
}

// RecordedResp captures the essential parts of an HTTP response for replay.
type RecordedResp struct {
	// StatusCode is the HTTP response status code.
	StatusCode int `json:"status_code"`

	// Headers contains the response headers.
	Headers http.Header `json:"headers"`

	// Body is the full response body bytes.
	Body []byte `json:"body"`
}

// NewTape creates a new Tape with a generated ID and the current UTC timestamp.
func NewTape(route string, req RecordedReq, resp RecordedResp) Tape {
	return Tape{
		ID:         newUUID(),
		Route:      route,
		RecordedAt: time.Now().UTC(),
		Request:    req,
		Response:   resp,
	}
}

// BodyHashFromBytes computes the hex-encoded SHA-256 hash of the given bytes.
// Returns an empty string if b is nil or empty.
func BodyHashFromBytes(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// newUUID generates a UUID v4 string using crypto/rand.
func newUUID() string {
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		// crypto/rand.Read should never fail on supported platforms.
		// If it does, return a zero UUID rather than panicking (library rule: no panics).
		return "00000000-0000-4000-8000-000000000000"
	}
	// Set version to 4 (bits 12-15 of time_hi_and_version).
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Set variant to RFC 4122 (bits 6-7 of clock_seq_hi_and_reserved).
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
