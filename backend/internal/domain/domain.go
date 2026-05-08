package domain

import "time"

type BookLevel struct {
	Price float64 `json:"price"`
	Size  int64   `json:"size"`
}

type OrderBook5 struct {
	Bids []BookLevel `json:"bids"`
	Asks []BookLevel `json:"asks"`
}

type QuoteState struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`

	LastPrice float64 `json:"lastPrice"`
	PrevClose float64 `json:"prevClose"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    int64   `json:"volume"`

	Change    float64 `json:"change"`
	ChangePct float64 `json:"changePct"`

	Book         OrderBook5 `json:"book"`
	LastUpdateTs int64      `json:"lastUpdateTs"`
}

type SparkPoint struct {
	T int64   `json:"t"`
	P float64 `json:"p"`
}

type Snapshot struct {
	Type    string                 `json:"type"`
	Ts      int64                  `json:"ts"`
	Symbols map[string]QuoteState  `json:"symbols"`
	Spark   map[string][]SparkPoint `json:"spark,omitempty"`
}

func NowUnixMs() int64 { return time.Now().UnixMilli() }

