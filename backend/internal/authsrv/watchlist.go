package authsrv

import (
	"encoding/json"
	"net/http"
	"strings"
)

type watchlistResponse struct {
	Username string   `json:"username"`
	Symbols  []string `json:"symbols"`
}

type watchlistMutateRequest struct {
	Username string `json:"username"`
	Symbol   string `json:"symbol"`
}

type watchlistErrResponse struct {
	Error string `json:"error"`
}

func (s *Store) GetWatchlist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	u := strings.TrimSpace(r.URL.Query().Get("username"))
	if u == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "username required"})
		return
	}
	userID, err := s.LookupUserID(u)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "unknown user"})
		return
	}

	syms, err := s.ListWatchlist(userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "failed to load watchlist"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(watchlistResponse{Username: u, Symbols: syms})
}

func (s *Store) AddWatchlist(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}
	w.Header().Set("Content-Type", "application/json")

	var req watchlistMutateRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "invalid request"})
		return
	}

	u := strings.TrimSpace(req.Username)
	sym := strings.TrimSpace(req.Symbol)
	if u == "" || sym == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "username and symbol required"})
		return
	}
	userID, err := s.LookupUserID(u)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "unknown user"})
		return
	}

	if err := s.AddToWatchlist(userID, sym); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "failed to add"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(watchlistResponse{Username: u, Symbols: []string{sym}})
}

func (s *Store) RemoveWatchlist(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}
	w.Header().Set("Content-Type", "application/json")

	var req watchlistMutateRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "invalid request"})
		return
	}

	u := strings.TrimSpace(req.Username)
	sym := strings.TrimSpace(req.Symbol)
	if u == "" || sym == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "username and symbol required"})
		return
	}
	userID, err := s.LookupUserID(u)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "unknown user"})
		return
	}

	if err := s.RemoveFromWatchlist(userID, sym); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(watchlistErrResponse{Error: "failed to remove"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(watchlistResponse{Username: u, Symbols: []string{sym}})
}

