package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"stock_record/backend/internal/agg"
	"stock_record/backend/internal/provider/mockprovider"
	"stock_record/backend/internal/provider/wsprovider/fugle"
	"stock_record/backend/internal/state"
	"stock_record/backend/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Load local config for developer convenience (ignore if missing).
	// NOTE: Keep real keys in backend/.env and do not commit it.
	_ = godotenv.Load()

	addr := envOr("ADDR", ":8080")
	snapshotEvery := envDurationOr("SNAPSHOT_EVERY", 1*time.Second)
	sparkWindow := envDurationOr("SPARK_WINDOW", 30*time.Minute)
	providerName := envOr("PROVIDER", "mock")
	fugleKey := os.Getenv("FUGLE_API_KEY")
	fugleSymbols := envOr("FUGLE_SYMBOLS", "2330,2308,2317")
	log.Printf("config: PROVIDER=%s FUGLE_SYMBOLS=%s", providerName, fugleSymbols)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := state.NewStore()
	hub := ws.NewHub()
	go hub.Run()

	var runProvider func(ctx context.Context, st *state.Store) error
	switch providerName {
	case "fugle":
		p := fugle.New(fugle.Config{
			APIKey:   fugleKey,
			Symbols:  splitCSV(fugleSymbols),
			Channel:  "aggregates",
		})
		// On any client subscribe, bootstrap via REST (for fast snapshot) and also
		// request streaming subscription so the symbol starts updating in-session.
		go func() {
			for syms := range hub.SubscribeEvents() {
				p.BootstrapMissing(ctx, st, syms)
				p.Subscribe(syms)
			}
		}()
		runProvider = p.Run
	case "mock":
		mock := mockprovider.New(mockprovider.Config{
			Symbols: []mockprovider.SymbolSpec{
				{Symbol: "2330", Name: "台積電"},
				{Symbol: "2317", Name: "鴻海"},
				{Symbol: "2454", Name: "聯發科"},
				{Symbol: "2881", Name: "富邦金"},
			},
			EventEvery: 200 * time.Millisecond,
		})
		runProvider = mock.Run
	default:
		log.Printf("unknown PROVIDER=%q, fallback to mock", providerName)
		mock := mockprovider.New(mockprovider.Config{
			Symbols: []mockprovider.SymbolSpec{
				{Symbol: "2330", Name: "台積電"},
				{Symbol: "2317", Name: "鴻海"},
			},
			EventEvery: 200 * time.Millisecond,
		})
		runProvider = mock.Run
	}

	go func() {
		if err := runProvider(ctx, st); err != nil {
			log.Printf("provider stopped: %v", err)
		}
	}()

	aggregator := agg.NewAggregator(st, hub, agg.Config{
		Every:       snapshotEvery,
		SparkWindow: sparkWindow,
	})
	go aggregator.Run(ctx)

	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/ws", hub.HandleWS)

	srv := &http.Server{Addr: addr, Handler: r}

	go func() {
		log.Printf("server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func envOr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func envDurationOr(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

