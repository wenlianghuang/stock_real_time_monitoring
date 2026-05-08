package agg

import (
	"context"
	"time"

	"stock_record/backend/internal/domain"
	"stock_record/backend/internal/state"
	"stock_record/backend/internal/ws"
)

type Config struct {
	Every       time.Duration
	SparkWindow time.Duration
}

type Aggregator struct {
	st  *state.Store
	hub *ws.Hub
	cfg Config

	series *seriesStore
}

func NewAggregator(st *state.Store, hub *ws.Hub, cfg Config) *Aggregator {
	if cfg.Every <= 0 {
		cfg.Every = time.Second
	}
	if cfg.SparkWindow <= 0 {
		cfg.SparkWindow = 30 * time.Minute
	}
	return &Aggregator{
		st:     st,
		hub:    hub,
		cfg:    cfg,
		series: newSeriesStore(cfg.SparkWindow),
	}
}

func (a *Aggregator) Run(ctx context.Context) {
	ticker := time.NewTicker(a.cfg.Every)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ts := domain.NowUnixMs()
			// per-client snapshots, only for subscribed symbols
			a.hub.BroadcastPerClient(func(symbols []string) (domain.Snapshot, bool) {
				if len(symbols) == 0 {
					return domain.Snapshot{Type: "snapshot", Ts: ts, Symbols: map[string]domain.QuoteState{}}, true
				}
				snap := a.st.Snapshot(symbols)
				for _, q := range snap {
					a.series.add(q.Symbol, ts, q.LastPrice)
				}
				return domain.Snapshot{
					Type:    "snapshot",
					Ts:      ts,
					Symbols: snap,
					Spark:   a.series.getMany(symbols),
				}, true
			})
		}
	}
}

