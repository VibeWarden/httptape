package httptape

import (
	"context"
	"errors"
)

// ErrNotFound is returned when a requested tape does not exist in the store.
var ErrNotFound = errors.New("httptape: tape not found")

// Filter controls which tapes are returned by Store.List.
type Filter struct {
	// Route filters tapes by route. Empty string means no filter (return all).
	Route string

	// Method filters tapes by HTTP method. Empty string means no filter.
	Method string
}

// Store persists and retrieves recorded HTTP interactions.
// It is the primary hexagonal port for persistence.
//
// All methods accept a context.Context for cancellation and deadline support.
// Implementations must respect context cancellation.
type Store interface {
	// Save persists a tape. If a tape with the same ID already exists,
	// it is overwritten (upsert semantics).
	Save(ctx context.Context, tape Tape) error

	// Load retrieves a single tape by ID.
	// Returns a non-nil error wrapping ErrNotFound if the tape does not exist.
	Load(ctx context.Context, id string) (Tape, error)

	// List returns all tapes matching the given filter.
	// An empty filter returns all tapes. Returns an empty slice (not nil) if
	// no tapes match.
	List(ctx context.Context, filter Filter) ([]Tape, error)

	// Delete removes a tape by ID.
	// Returns a non-nil error wrapping ErrNotFound if the tape does not exist.
	Delete(ctx context.Context, id string) error
}
