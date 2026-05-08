package provider

import (
	"context"

	"stock_record/backend/internal/state"
)

// MarketDataProvider writes updates into the shared Store.
// In real integrations, this is implemented by a vendor-specific adapter.
type MarketDataProvider interface {
	Run(ctx context.Context, st *state.Store) error
}

