package fugle

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"io"
	"strings"
	"sync"
	"time"

	"stock_record/backend/internal/domain"
	"stock_record/backend/internal/state"

	"github.com/gorilla/websocket"
)

const defaultURL = "wss://api.fugle.tw/marketdata/v1.0/stock/streaming"
const defaultRESTBase = "https://api.fugle.tw/marketdata/v1.0/stock"

type Config struct {
	URL    string
	APIKey string
	// Symbols are TW stock ids, e.g. "2330".
	Symbols []string
	// Subscribe aggregates channel (recommended).
	Channel string
}

type Provider struct {
	cfg Config
	restBase string
	http     *http.Client

	desiredMu sync.RWMutex
	desired   map[string]struct{}
	subReq    chan []string
}

func New(cfg Config) *Provider {
	if cfg.URL == "" {
		cfg.URL = defaultURL
	}
	if cfg.Channel == "" {
		cfg.Channel = "aggregates"
	}
	// API keys should not contain whitespace; strip defensively to avoid common copy/paste issues.
	if cfg.APIKey != "" {
		cfg.APIKey = strings.Join(strings.Fields(cfg.APIKey), "")
	}
	cfg.Symbols = normalizeSymbols(cfg.Symbols)
	p := &Provider{
		cfg:      cfg,
		restBase: defaultRESTBase,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
	p.desired = make(map[string]struct{}, len(cfg.Symbols)+16)
	for _, s := range cfg.Symbols {
		p.desired[s] = struct{}{}
	}
	p.subReq = make(chan []string, 256)
	return p
}

// Subscribe requests streaming updates for additional symbols.
// It is safe to call at any time; the provider will subscribe when connected/in-session.
func (p *Provider) Subscribe(symbols []string) {
	symbols = normalizeSymbols(symbols)
	if len(symbols) == 0 {
		return
	}

	p.desiredMu.Lock()
	for _, s := range symbols {
		p.desired[s] = struct{}{}
	}
	p.desiredMu.Unlock()

	select {
	case p.subReq <- symbols:
	default:
		// best-effort: drop if busy; desired set is still updated
	}
}

func (p *Provider) desiredSymbols() []string {
	p.desiredMu.RLock()
	out := make([]string, 0, len(p.desired))
	for s := range p.desired {
		out = append(out, s)
	}
	p.desiredMu.RUnlock()
	return out
}

func (p *Provider) Run(ctx context.Context, st *state.Store) error {
	if p.cfg.APIKey == "" {
		return errors.New("Fugle API key is empty (set FUGLE_API_KEY)")
	}
	if len(p.cfg.Symbols) == 0 {
		return errors.New("no symbols configured (set FUGLE_SYMBOLS like \"2330,2308\")")
	}

	log.Printf("fugle: starting url=%s channel=%s symbols=%v", p.cfg.URL, p.cfg.Channel, p.cfg.Symbols)

	loc, _ := time.LoadLocation("Asia/Taipei")
	var lastBootstrapDay string

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		now := time.Now().In(loc)
		inSession, nextBoundary := twseInSessionAndNextBoundary(now)

		if !inSession {
			day := now.Format("2006-01-02")
			if lastBootstrapDay != day {
				if err := p.bootstrapQuotes(ctx, st); err != nil {
					log.Printf("fugle: rest bootstrap error: %v", err)
				} else {
					log.Printf("fugle: rest bootstrap ok (day=%s symbols=%d)", day, len(p.cfg.Symbols))
					lastBootstrapDay = day
				}
			}

			sleep := time.Until(nextBoundary)
			if sleep < 5*time.Second {
				sleep = 5 * time.Second
			}
			log.Printf("fugle: out of session, sleep %s until %s", sleep.Round(time.Second), nextBoundary.Format(time.RFC3339))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(sleep):
				continue
			}
		}

		// In session: run WS streaming with reconnect backoff.
		backoff := 500 * time.Millisecond
		for {
			if err := p.runOnce(ctx, st); err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("fugle provider error: %v (reconnect in %s)", err, backoff)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
				backoff *= 2
				if backoff > 10*time.Second {
					backoff = 10 * time.Second
				}

				// If session ended while reconnecting, break to outer loop.
				now = time.Now().In(loc)
				inSession, _ = twseInSessionAndNextBoundary(now)
				if !inSession {
					break
				}
				continue
			}

			// normal exit shouldn't happen; reconnect defensively
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}
}

// BootstrapMissing fetches and upserts quotes for symbols not present in Store.
// This is intended for out-of-session (or first-subscribe) "last snapshot" UX.
func (p *Provider) BootstrapMissing(ctx context.Context, st *state.Store, symbols []string) {
	symbols = normalizeSymbols(symbols)
	for _, sym := range symbols {
		if _, ok := st.GetQuote(sym); ok {
			continue
		}
		q, err := p.fetchQuote(ctx, sym)
		if err != nil {
			log.Printf("fugle: rest bootstrap symbol=%s error=%v", sym, err)
			continue
		}
		if q.Name == "" {
			if name, ok := knownNames[q.Symbol]; ok {
				q.Name = name
			}
		}
		q.LastUpdateTs = domain.NowUnixMs()
		st.UpsertQuote(q)
		log.Printf("fugle: rest bootstrap symbol=%s ok last=%.2f", q.Symbol, q.LastPrice)
	}
}

func (p *Provider) runOnce(ctx context.Context, st *state.Store) error {
	dialer := websocket.Dialer{
		// Prefer direct connections; proxies can break WS frames in some environments.
		Proxy:            nil,
		HandshakeTimeout: 10 * time.Second,
		// Some servers / clients (including common JS stacks) negotiate permessage-deflate by default.
		// Enabling it improves interoperability.
		EnableCompression: true,
	}
	conn, _, err := dialer.DialContext(ctx, p.cfg.URL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()
	log.Printf("fugle: ws connected")

	// Diagnostics (kept lightweight):
	// - If JSON decoding fails, log a truncated raw message.
	// - Log the first few events to confirm liveness after auth.
	loggedRaw := 0
	loggedEvents := 0

	readTimeout := 90 * time.Second
	_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
	var pongCount int64
	conn.SetPongHandler(func(appData string) error {
		pongCount++
		// keep quiet; pongs are expected
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
		return nil
	})

	var writeMu sync.Mutex
	writeJSON := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteJSON(v)
	}

	// Tracks what we've already subscribed on this WS connection.
	subscribed := make(map[string]struct{}, 256)
	subscribeSymbols := func(symbols []string) error {
		symbols = normalizeSymbols(symbols)
		if len(symbols) == 0 {
			return nil
		}
		newSyms := make([]string, 0, len(symbols))
		for _, s := range symbols {
			if _, ok := subscribed[s]; ok {
				continue
			}
			subscribed[s] = struct{}{}
			newSyms = append(newSyms, s)
		}
		if len(newSyms) == 0 {
			return nil
		}
		log.Printf("fugle: ws subscribe add symbols=%v", newSyms)
		// subscribe channel for multiple symbols (incremental)
		return writeJSON(map[string]any{
			"event": "subscribe",
			"data": map[string]any{
				"channel": p.cfg.Channel,
				"symbols": newSyms,
			},
		})
	}

	// auth
	if err := writeJSON(map[string]any{
		"event": "auth",
		"data":  map[string]any{"apikey": p.cfg.APIKey},
	}); err != nil {
		return err
	}
	log.Printf("fugle: auth sent")

	authOK := false
	authedCh := make(chan struct{})

	pingTicker := time.NewTicker(25 * time.Second)
	defer pingTicker.Stop()

	// log counters to help debug (without leaking key)
	var dataCount int64

	// write pump: keepalive ping + WS ping control frames
	done := make(chan struct{})
	defer close(done)
	go func() {
		wsPing := time.NewTicker(30 * time.Second)
		defer wsPing.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-pingTicker.C:
				// Fugle JSON ping/pong
				_ = writeJSON(map[string]any{"event": "ping", "data": map[string]any{"state": "keepalive"}})
			case <-wsPing.C:
				// WebSocket protocol ping to trigger pong handler / keep NAT alive
				writeMu.Lock()
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				_ = conn.WriteMessage(websocket.PingMessage, nil)
				writeMu.Unlock()
			}
		}
	}()

	// Read pump: parse WS messages and update store.
	errCh := make(chan error, 1)
	go func() {
		defer func() { errCh <- errors.New("fugle ws read loop exited") }()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			_ = conn.SetReadDeadline(time.Now().Add(readTimeout))

			var env envelope
			if err := json.Unmarshal(msg, &env); err != nil {
				if loggedRaw < 5 {
					raw := msg
					if len(raw) > 512 {
						raw = raw[:512]
					}
					log.Printf("fugle: ws json unmarshal error: %v raw=%q", err, string(raw))
					loggedRaw++
				}
				continue
			}

			if loggedEvents < 10 && env.Event != "" {
				log.Printf("fugle: ws recv event=%s channel=%s id=%s", env.Event, env.Channel, env.ID)
				loggedEvents++
			}

			switch env.Event {
			case "authenticated":
				if !authOK {
					log.Printf("fugle: authenticated")
					authOK = true
					close(authedCh)
				}
			case "error":
				errCh <- errors.New("fugle ws error: " + env.DataMessage())
				return
			case "data":
				if !authOK {
					continue
				}
				if env.Channel != p.cfg.Channel {
					continue
				}
				q, ok := mapAggregateToQuote(env.Data)
				if ok {
					dataCount++
					if dataCount == 1 || dataCount%200 == 0 {
						log.Printf("fugle: data received count=%d last=%s %.2f", dataCount, q.Symbol, q.LastPrice)
					}
					if q.Name == "" {
						if name, ok := knownNames[q.Symbol]; ok {
							q.Name = name
						}
					}
					q.LastUpdateTs = domain.NowUnixMs()
					st.UpsertQuote(q)
				}
			case "subscribed":
				log.Printf("fugle: subscribed ok")
			case "heartbeat":
				// ignore
			case "snapshot", "pong":
				// ignore
			default:
				// ignore
			}
		}
	}()

	// Wait for authenticated (or fail fast) before subscribing.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	case <-time.After(10 * time.Second):
		return errors.New("fugle ws auth timeout (no authenticated event)")
	case <-authedCh:
	}

	// Initial subscribe: desired symbols (config + any already requested via Subscribe()).
	if err := subscribeSymbols(p.desiredSymbols()); err != nil {
		return err
	}
	log.Printf("fugle: subscribe channel=%s symbols=%v", p.cfg.Channel, p.desiredSymbols())

	// Dynamic subscribe loop: when UI subscribes to more symbols, add them incrementally.
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			return err
		case syms := <-p.subReq:
			if !authOK {
				continue
			}
			if err := subscribeSymbols(syms); err != nil {
				return err
			}
		}
	}
}

type envelope struct {
	Event   string          `json:"event"`
	Data    json.RawMessage `json:"data"`
	ID      string          `json:"id"`
	Channel string          `json:"channel"`
}

func (e envelope) DataMessage() string {
	var m struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(e.Data, &m)
	return m.Message
}

type aggregateMsg struct {
	Symbol          string  `json:"symbol"`
	Name            string  `json:"name"`
	PreviousClose   float64 `json:"previousClose"`
	OpenPrice       float64 `json:"openPrice"`
	HighPrice       float64 `json:"highPrice"`
	LowPrice        float64 `json:"lowPrice"`
	LastPrice       float64 `json:"lastPrice"`
	Change          float64 `json:"change"`
	ChangePercent   float64 `json:"changePercent"` // percentage number, e.g. 0.35 means 0.35%
	Bids            []level `json:"bids"`
	Asks            []level `json:"asks"`
	Total           total   `json:"total"`
	LastUpdatedTime int64   `json:"lastUpdated"`
}

type level struct {
	Price float64 `json:"price"`
	Size  int64   `json:"size"`
}

type total struct {
	TradeVolume int64 `json:"tradeVolume"`
}

func mapAggregateToQuote(raw json.RawMessage) (domain.QuoteState, bool) {
	var a aggregateMsg
	if err := json.Unmarshal(raw, &a); err != nil {
		return domain.QuoteState{}, false
	}
	if a.Symbol == "" {
		return domain.QuoteState{}, false
	}

	q := domain.QuoteState{
		Symbol:    a.Symbol,
		Name:      a.Name,
		LastPrice: a.LastPrice,
		PrevClose: a.PreviousClose,
		Open:      a.OpenPrice,
		High:      a.HighPrice,
		Low:       a.LowPrice,
		Volume:    a.Total.TradeVolume,
		Change:    a.Change,
		// our frontend expects decimal ratio (e.g. 0.008), fugle uses percent number (e.g. 0.35)
		ChangePct: a.ChangePercent / 100.0,
		Book: domain.OrderBook5{
			Bids: mapLevels(a.Bids),
			Asks: mapLevels(a.Asks),
		},
	}
	return q, true
}

func mapLevels(in []level) []domain.BookLevel {
	out := make([]domain.BookLevel, 0, 5)
	for i := 0; i < len(in) && i < 5; i++ {
		out = append(out, domain.BookLevel{Price: in[i].Price, Size: in[i].Size})
	}
	return out
}

func normalizeSymbols(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

var knownNames = map[string]string{
	"2330": "台積電",
	"2308": "台達電",
	"2317": "鴻海",
	"2454": "聯發科",
	"2881": "富邦金",
}

// twseInSessionAndNextBoundary uses Taiwan time and a simplified session window:
// Mon-Fri 09:00–13:30.
// It returns whether "now" is in-session and the next boundary time (open or close).
func twseInSessionAndNextBoundary(now time.Time) (bool, time.Time) {
	wd := now.Weekday()
	// weekend: next open is next Monday 09:00
	if wd == time.Saturday || wd == time.Sunday {
		days := (time.Monday - wd + 7) % 7
		if days == 0 {
			days = 7
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()).AddDate(0, 0, int(days))
		return false, next
	}

	open := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location())
	close := time.Date(now.Year(), now.Month(), now.Day(), 13, 30, 0, 0, now.Location())

	if now.Before(open) {
		return false, open
	}
	if now.After(close) || now.Equal(close) {
		// next weekday open
		nextDay := now.AddDate(0, 0, 1)
		for nextDay.Weekday() == time.Saturday || nextDay.Weekday() == time.Sunday {
			nextDay = nextDay.AddDate(0, 0, 1)
		}
		nextOpen := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 9, 0, 0, 0, now.Location())
		return false, nextOpen
	}
	return true, close
}

func (p *Provider) bootstrapQuotes(ctx context.Context, st *state.Store) error {
	for _, sym := range p.cfg.Symbols {
		q, err := p.fetchQuote(ctx, sym)
		if err != nil {
			return err
		}
		if q.Name == "" {
			if name, ok := knownNames[q.Symbol]; ok {
				q.Name = name
			}
		}
		q.LastUpdateTs = domain.NowUnixMs()
		st.UpsertQuote(q)
	}
	return nil
}

func (p *Provider) fetchQuote(ctx context.Context, symbol string) (domain.QuoteState, error) {
	u, _ := url.JoinPath(p.restBase, "intraday", "quote", symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return domain.QuoteState{}, err
	}
	req.Header.Set("X-API-KEY", p.cfg.APIKey)

	resp, err := p.http.Do(req)
	if err != nil {
		return domain.QuoteState{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.QuoteState{}, errors.New("fugle rest status " + resp.Status + ": " + string(body))
	}

	// REST quote response schema is compatible with aggregateMsg for our mapping fields.
	q, ok := mapAggregateToQuote(json.RawMessage(body))
	if !ok {
		return domain.QuoteState{}, errors.New("failed to parse fugle quote response")
	}
	return q, nil
}

