package mockprovider

import (
	"context"
	"math"
	"math/rand"
	"time"

	"stock_record/backend/internal/domain"
	"stock_record/backend/internal/state"
)

type SymbolSpec struct {
	Symbol string
	Name   string
}

type Config struct {
	Symbols    []SymbolSpec
	EventEvery time.Duration
}

type Provider struct {
	cfg Config
	rng *rand.Rand

	base map[string]float64
}

func New(cfg Config) *Provider {
	if cfg.EventEvery <= 0 {
		cfg.EventEvery = 200 * time.Millisecond
	}
	p := &Provider{
		cfg:  cfg,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
		base: make(map[string]float64),
	}
	for _, s := range cfg.Symbols {
		// somewhat reasonable starting prices for demo
		start := 100 + p.rng.Float64()*900
		p.base[s.Symbol] = math.Round(start*10) / 10
	}
	return p
}

func (p *Provider) Run(ctx context.Context, st *state.Store) error {
	ticker := time.NewTicker(p.cfg.EventEvery)
	defer ticker.Stop()

	// initialize
	for _, s := range p.cfg.Symbols {
		prev := p.base[s.Symbol]
		q := domain.QuoteState{
			Symbol: s.Symbol,
			Name:   s.Name,
			LastPrice: prev,
			PrevClose: prev,
			Open:      prev,
			High:      prev,
			Low:       prev,
			Volume:    0,
			Change:    0,
			ChangePct: 0,
			Book:      genBook(prev, p.rng),
			LastUpdateTs: domain.NowUnixMs(),
		}
		st.UpsertQuote(q)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			for _, s := range p.cfg.Symbols {
				p.stepSymbol(st, s)
			}
		}
	}
}

func (p *Provider) stepSymbol(st *state.Store, s SymbolSpec) {
	q, ok := st.GetQuote(s.Symbol)
	if !ok {
		return
	}

	// random walk
	move := (p.rng.Float64() - 0.5) * 2.0 // [-1,1]
	move *= q.LastPrice * 0.001          // ~0.1%
	newPrice := q.LastPrice + move
	if newPrice <= 0 {
		newPrice = q.LastPrice
	}
	newPrice = math.Round(newPrice*10) / 10

	q.LastPrice = newPrice
	if q.Volume == 0 {
		q.Open = newPrice
	}
	if newPrice > q.High {
		q.High = newPrice
	}
	if newPrice < q.Low {
		q.Low = newPrice
	}

	q.Volume += int64(10 + p.rng.Intn(200))
	q.Change = q.LastPrice - q.PrevClose
	if q.PrevClose != 0 {
		q.ChangePct = q.Change / q.PrevClose
	}

	q.Book = genBook(q.LastPrice, p.rng)
	q.LastUpdateTs = domain.NowUnixMs()
	st.UpsertQuote(q)
}

func genBook(mid float64, rng *rand.Rand) domain.OrderBook5 {
	levels := 5
	tick := 0.1
	bids := make([]domain.BookLevel, 0, levels)
	asks := make([]domain.BookLevel, 0, levels)

	for i := 0; i < levels; i++ {
		bp := mid - float64(i+1)*tick
		ap := mid + float64(i+1)*tick
		bids = append(bids, domain.BookLevel{Price: math.Round(bp*10) / 10, Size: int64(10 + rng.Intn(500))})
		asks = append(asks, domain.BookLevel{Price: math.Round(ap*10) / 10, Size: int64(10 + rng.Intn(500))})
	}
	return domain.OrderBook5{Bids: bids, Asks: asks}
}

