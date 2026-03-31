package httptape

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// makeTape creates a minimal valid tape for testing.
func makeTape(route, method, url string) Tape {
	return NewTape(route, RecordedReq{
		Method:   method,
		URL:      url,
		Headers:  http.Header{"Content-Type": {"application/json"}},
		Body:     []byte(`{"key":"value"}`),
		BodyHash: BodyHashFromBytes([]byte(`{"key":"value"}`)),
	}, RecordedResp{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": {"application/json"}},
		Body:       []byte(`{"result":"ok"}`),
	})
}

func TestNewMemoryStore_WithOption(t *testing.T) {
	called := false
	opt := func(ms *MemoryStore) { called = true }
	_ = NewMemoryStore(opt)
	if !called {
		t.Error("MemoryStoreOption was not applied")
	}
}

func TestMemoryStore_Save(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	tape := makeTape("users-api", "GET", "http://example.com/users")

	if err := store.Save(ctx, tape); err != nil {
		t.Fatalf("Save() error = %v, want nil", err)
	}

	loaded, err := store.Load(ctx, tape.ID)
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if loaded.ID != tape.ID {
		t.Errorf("Load().ID = %q, want %q", loaded.ID, tape.ID)
	}
	if loaded.Route != tape.Route {
		t.Errorf("Load().Route = %q, want %q", loaded.Route, tape.Route)
	}
	if loaded.Request.Method != tape.Request.Method {
		t.Errorf("Load().Request.Method = %q, want %q", loaded.Request.Method, tape.Request.Method)
	}
	if loaded.Request.URL != tape.Request.URL {
		t.Errorf("Load().Request.URL = %q, want %q", loaded.Request.URL, tape.Request.URL)
	}
	if loaded.Response.StatusCode != tape.Response.StatusCode {
		t.Errorf("Load().Response.StatusCode = %d, want %d", loaded.Response.StatusCode, tape.Response.StatusCode)
	}
}

func TestMemoryStore_Load_NotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	_, err := store.Load(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("Load() error = nil, want ErrNotFound")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load() error = %v, want error wrapping ErrNotFound", err)
	}
}

func TestMemoryStore_List_All(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tape1 := makeTape("route-a", "GET", "http://example.com/a")
	tape2 := makeTape("route-b", "POST", "http://example.com/b")
	tape3 := makeTape("route-a", "DELETE", "http://example.com/c")

	for _, tape := range []Tape{tape1, tape2, tape3} {
		if err := store.Save(ctx, tape); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	tapes, err := store.List(ctx, Filter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tapes) != 3 {
		t.Errorf("List() returned %d tapes, want 3", len(tapes))
	}
}

func TestMemoryStore_List_ByRoute(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tape1 := makeTape("route-a", "GET", "http://example.com/a")
	tape2 := makeTape("route-b", "POST", "http://example.com/b")
	tape3 := makeTape("route-a", "DELETE", "http://example.com/c")

	for _, tape := range []Tape{tape1, tape2, tape3} {
		if err := store.Save(ctx, tape); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	tapes, err := store.List(ctx, Filter{Route: "route-a"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tapes) != 2 {
		t.Errorf("List(Route=route-a) returned %d tapes, want 2", len(tapes))
	}
	for _, tape := range tapes {
		if tape.Route != "route-a" {
			t.Errorf("List(Route=route-a) returned tape with route %q", tape.Route)
		}
	}
}

func TestMemoryStore_List_ByMethod(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tape1 := makeTape("route-a", "GET", "http://example.com/a")
	tape2 := makeTape("route-b", "POST", "http://example.com/b")
	tape3 := makeTape("route-a", "GET", "http://example.com/c")

	for _, tape := range []Tape{tape1, tape2, tape3} {
		if err := store.Save(ctx, tape); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	tapes, err := store.List(ctx, Filter{Method: "GET"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tapes) != 2 {
		t.Errorf("List(Method=GET) returned %d tapes, want 2", len(tapes))
	}
	for _, tape := range tapes {
		if tape.Request.Method != "GET" {
			t.Errorf("List(Method=GET) returned tape with method %q", tape.Request.Method)
		}
	}
}

func TestMemoryStore_List_Empty(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tapes, err := store.List(ctx, Filter{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if tapes == nil {
		t.Error("List() returned nil, want empty slice")
	}
	if len(tapes) != 0 {
		t.Errorf("List() returned %d tapes, want 0", len(tapes))
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tape := makeTape("users-api", "GET", "http://example.com/users")
	if err := store.Save(ctx, tape); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := store.Delete(ctx, tape.ID); err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}

	_, err := store.Load(ctx, tape.ID)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Load() after Delete() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryStore_Delete_NotFound(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	err := store.Delete(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("Delete() error = nil, want ErrNotFound")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete() error = %v, want error wrapping ErrNotFound", err)
	}
}

func TestMemoryStore_Save_Overwrite(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tape := makeTape("users-api", "GET", "http://example.com/users")
	if err := store.Save(ctx, tape); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Overwrite with updated route.
	tape.Route = "updated-route"
	if err := store.Save(ctx, tape); err != nil {
		t.Fatalf("Save() overwrite error = %v", err)
	}

	loaded, err := store.Load(ctx, tape.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Route != "updated-route" {
		t.Errorf("Load().Route = %q, want %q", loaded.Route, "updated-route")
	}
}

func TestMemoryStore_Isolation(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tape := makeTape("users-api", "GET", "http://example.com/users")
	if err := store.Save(ctx, tape); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Mutate the original tape after saving.
	tape.Route = "mutated-route"
	tape.Request.Headers.Set("X-Mutated", "true")
	tape.Request.Body[0] = 'X'

	// Load and verify the stored copy is unchanged.
	loaded, err := store.Load(ctx, tape.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Route == "mutated-route" {
		t.Error("Store did not deep-copy: route was mutated")
	}
	if loaded.Request.Headers.Get("X-Mutated") == "true" {
		t.Error("Store did not deep-copy: request headers were mutated")
	}
	if loaded.Request.Body[0] == 'X' {
		t.Error("Store did not deep-copy: request body was mutated")
	}

	// Also verify that mutating the loaded copy does not affect the store.
	loaded.Route = "loaded-mutated"
	loaded2, err := store.Load(ctx, tape.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded2.Route == "loaded-mutated" {
		t.Error("Store did not deep-copy on Load: route was mutated through loaded copy")
	}
}

func TestMemoryStore_NilBodyAndHeaders(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tape := NewTape("route", RecordedReq{
		Method:  "GET",
		URL:     "http://example.com",
		Headers: nil,
		Body:    nil,
	}, RecordedResp{
		StatusCode: 200,
		Headers:    nil,
		Body:       nil,
	})

	if err := store.Save(ctx, tape); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load(ctx, tape.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Request.Headers != nil {
		t.Error("Expected nil request headers after round-trip")
	}
	if loaded.Request.Body != nil {
		t.Error("Expected nil request body after round-trip")
	}
	if loaded.Response.Headers != nil {
		t.Error("Expected nil response headers after round-trip")
	}
	if loaded.Response.Body != nil {
		t.Error("Expected nil response body after round-trip")
	}
}

func TestMemoryStore_ListRouteAndMethodFilter(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	tape1 := makeTape("route-a", "GET", "http://example.com/a")
	tape2 := makeTape("route-a", "POST", "http://example.com/b")
	tape3 := makeTape("route-b", "GET", "http://example.com/c")

	for _, tape := range []Tape{tape1, tape2, tape3} {
		if err := store.Save(ctx, tape); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	tapes, err := store.List(ctx, Filter{Route: "route-a", Method: "GET"})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tapes) != 1 {
		t.Errorf("List(Route=route-a, Method=GET) returned %d tapes, want 1", len(tapes))
	}
}

func TestMemoryStore_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	store := NewMemoryStore()
	tape := makeTape("users-api", "GET", "http://example.com/users")

	tests := []struct {
		name string
		fn   func() error
	}{
		{"Save", func() error { return store.Save(ctx, tape) }},
		{"Load", func() error { _, err := store.Load(ctx, "any"); return err }},
		{"List", func() error { _, err := store.List(ctx, Filter{}); return err }},
		{"Delete", func() error { return store.Delete(ctx, "any") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fn()
			if err == nil {
				t.Errorf("%s() with cancelled context: error = nil, want non-nil", tt.name)
			}
			if !errors.Is(err, context.Canceled) {
				t.Errorf("%s() with cancelled context: error = %v, want context.Canceled", tt.name, err)
			}
		})
	}
}
