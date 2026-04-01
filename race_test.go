package httptape

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestMemoryStore_ConcurrentSaveLoad exercises concurrent Save and Load
// operations on MemoryStore under the race detector.
func TestMemoryStore_ConcurrentSaveLoad(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	const n = 100

	var wg sync.WaitGroup

	// Concurrent saves.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tape := makeTape("route", "GET", fmt.Sprintf("http://example.com/%d", i))
			if err := store.Save(ctx, tape); err != nil {
				t.Errorf("Save(%d) error: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent loads.
	tapes, err := store.List(ctx, Filter{})
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	for _, tape := range tapes {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if _, err := store.Load(ctx, id); err != nil {
				t.Errorf("Load(%s) error: %v", id, err)
			}
		}(tape.ID)
	}
	wg.Wait()

	// Concurrent mixed reads and writes.
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			tape := makeTape("mixed", "POST", fmt.Sprintf("http://example.com/mixed/%d", i))
			_ = store.Save(ctx, tape)
		}(i)
		go func() {
			defer wg.Done()
			_, _ = store.List(ctx, Filter{})
		}()
	}
	wg.Wait()
}

// TestFileStore_ConcurrentSaveLoad exercises concurrent Save and Load
// operations on FileStore under the race detector.
func TestFileStore_ConcurrentSaveLoad(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "httptape-race-filestore-"+t.Name())
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	store, err := NewFileStore(WithDirectory(dir))
	if err != nil {
		t.Fatalf("NewFileStore() error: %v", err)
	}

	ctx := context.Background()
	const n = 50

	var wg sync.WaitGroup

	// Concurrent saves.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tape := makeTape("route", "GET", fmt.Sprintf("http://example.com/%d", i))
			if err := store.Save(ctx, tape); err != nil {
				t.Errorf("Save(%d) error: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	// Concurrent loads.
	tapes, listErr := store.List(ctx, Filter{})
	if listErr != nil {
		t.Fatalf("List() error: %v", listErr)
	}
	for _, tape := range tapes {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			if _, err := store.Load(ctx, id); err != nil {
				t.Errorf("Load(%s) error: %v", id, err)
			}
		}(tape.ID)
	}
	wg.Wait()

	// Concurrent mixed reads and writes.
	for i := 0; i < n; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			tape := makeTape("mixed", "POST", fmt.Sprintf("http://example.com/mixed/%d", i))
			_ = store.Save(ctx, tape)
		}(i)
		go func() {
			defer wg.Done()
			_, _ = store.List(ctx, Filter{})
		}()
	}
	wg.Wait()
}

// TestRecorder_ConcurrentRoundTrip exercises concurrent RoundTrip calls
// followed by Close under the race detector, verifying the TOCTOU fix.
func TestRecorder_ConcurrentRoundTrip(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	store := NewMemoryStore()
	rec := NewRecorder(store,
		WithTransport(backend.Client().Transport),
		WithRoute("race-test"),
		WithBufferSize(256),
	)

	const n = 100
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req, err := http.NewRequest("GET", backend.URL+fmt.Sprintf("/path/%d", i), nil)
			if err != nil {
				t.Errorf("NewRequest error: %v", err)
				return
			}
			resp, err := rec.RoundTrip(req)
			if err != nil {
				t.Errorf("RoundTrip(%d) error: %v", i, err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}
	wg.Wait()

	if err := rec.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

// TestRecorder_ConcurrentRoundTripAndClose exercises the specific race between
// RoundTrip and Close that the sendMu fix addresses.
func TestRecorder_ConcurrentRoundTripAndClose(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer backend.Close()

	store := NewMemoryStore()
	rec := NewRecorder(store,
		WithTransport(backend.Client().Transport),
		WithRoute("close-race"),
		WithBufferSize(16),
	)

	const n = 50
	var wg sync.WaitGroup

	// Launch many concurrent RoundTrips.
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req, err := http.NewRequest("POST", backend.URL+"/api",
				strings.NewReader(fmt.Sprintf(`{"i":%d}`, i)))
			if err != nil {
				t.Errorf("NewRequest error: %v", err)
				return
			}
			resp, err := rec.RoundTrip(req)
			if err != nil {
				// Errors after close are acceptable.
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}

	// Close concurrently while RoundTrips are in flight.
	wg.Add(1)
	go func() {
		defer wg.Done()
		rec.Close()
	}()

	wg.Wait()
}

// TestServer_ConcurrentServeHTTP exercises concurrent ServeHTTP calls
// under the race detector.
func TestServer_ConcurrentServeHTTP(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Pre-load some tapes.
	for i := 0; i < 10; i++ {
		tape := makeTape("server-race", "GET", fmt.Sprintf("/items/%d", i))
		if err := store.Save(ctx, tape); err != nil {
			t.Fatalf("Save() error: %v", err)
		}
	}

	srv := NewServer(store)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	const n = 100
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			path := fmt.Sprintf("/items/%d", i%10)
			resp, err := http.Get(ts.URL + path)
			if err != nil {
				t.Errorf("GET %s error: %v", path, err)
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}(i)
	}
	wg.Wait()
}
