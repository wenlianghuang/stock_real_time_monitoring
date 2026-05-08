package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"stock_record/backend/internal/domain"

	"github.com/gorilla/websocket"
)

type client struct {
	conn *websocket.Conn

	mu      sync.RWMutex
	symbols map[string]struct{}

	send chan []byte
}

type Hub struct {
	upgrader websocket.Upgrader

	register   chan *client
	unregister chan *client

	clientsMu sync.RWMutex
	clients   map[*client]struct{}

	subscribeEvents chan []string
}

func NewHub() *Hub {
	return &Hub{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		register:   make(chan *client),
		unregister: make(chan *client),
		clients:    make(map[*client]struct{}),
		subscribeEvents: make(chan []string, 128),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clientsMu.Lock()
			h.clients[c] = struct{}{}
			h.clientsMu.Unlock()
			h.sendStatus(c, "connected")
		case c := <-h.unregister:
			h.clientsMu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.clientsMu.Unlock()
		}
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{
		conn:    conn,
		symbols: make(map[string]struct{}),
		send:    make(chan []byte, 64),
	}
	h.register <- c

	go h.writePump(c)
	go h.readPump(c)
}

// SubscribeEvents emits symbols when any client subscribes.
// It is best-effort (drops if the buffer is full).
func (h *Hub) SubscribeEvents() <-chan []string {
	return h.subscribeEvents
}

type clientMsg struct {
	Type    string   `json:"type"`
	Symbols []string `json:"symbols,omitempty"`
	Ts      int64    `json:"ts,omitempty"`
}

func (h *Hub) readPump(c *client) {
	defer func() {
		_ = c.conn.Close()
		h.unregister <- c
	}()
	c.conn.SetReadLimit(1 << 20)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var m clientMsg
		if err := json.Unmarshal(data, &m); err != nil {
			continue
		}
		switch m.Type {
		case "subscribe":
			c.mu.Lock()
			for _, s := range m.Symbols {
				if s != "" {
					c.symbols[s] = struct{}{}
				}
			}
			c.mu.Unlock()
			// notify backend to optionally bootstrap missing quotes
			if len(m.Symbols) > 0 {
				select {
				case h.subscribeEvents <- m.Symbols:
				default:
				}
			}
		case "unsubscribe":
			c.mu.Lock()
			for _, s := range m.Symbols {
				delete(c.symbols, s)
			}
			c.mu.Unlock()
		case "ping":
			h.sendStatus(c, "pong")
		}
	}
}

func (h *Hub) writePump(c *client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (h *Hub) SubscribedSymbols(c *client) []string {
	c.mu.RLock()
	out := make([]string, 0, len(c.symbols))
	for s := range c.symbols {
		out = append(out, s)
	}
	c.mu.RUnlock()
	return out
}

func (h *Hub) BroadcastSnapshot(snapshot domain.Snapshot) {
	data, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	h.clientsMu.RLock()
	for c := range h.clients {
		select {
		case c.send <- data:
		default:
			// drop if slow
		}
	}
	h.clientsMu.RUnlock()
}

func (h *Hub) BroadcastPerClient(build func(symbols []string) (domain.Snapshot, bool)) {
	h.clientsMu.RLock()
	for c := range h.clients {
		syms := h.SubscribedSymbols(c)
		snap, ok := build(syms)
		if !ok {
			continue
		}
		data, err := json.Marshal(snap)
		if err != nil {
			continue
		}
		select {
		case c.send <- data:
		default:
		}
	}
	h.clientsMu.RUnlock()
}

func (h *Hub) sendStatus(c *client, msg string) {
	payload := map[string]any{"type": "status", "message": msg}
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("status marshal error: %v", err)
		return
	}
	select {
	case c.send <- data:
	default:
	}
}

