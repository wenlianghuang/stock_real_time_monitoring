package authsrv

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginOKResponse struct {
	OK       bool   `json:"ok"`
	Username string `json:"username"`
}

type loginErrResponse struct {
	Error string `json:"error"`
}

// Login verifies credentials against the users table (bcrypt hashes).
func (s *Store) Login(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}
	w.Header().Set("Content-Type", "application/json")

	var req loginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(loginErrResponse{Error: "invalid request"})
		return
	}

	u := strings.TrimSpace(req.Username)
	p := req.Password

	if u == "" || p == "" {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(loginErrResponse{Error: "invalid username or password"})
		return
	}

	hash, err := s.LookupPasswordHash(u)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(loginErrResponse{Error: "invalid username or password"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(p)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(loginErrResponse{Error: "invalid username or password"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(loginOKResponse{OK: true, Username: u})
}

type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register creates a new account (POST JSON {username, password}).
func (s *Store) Register(w http.ResponseWriter, r *http.Request) {
	if r.Body != nil {
		defer r.Body.Close()
	}
	w.Header().Set("Content-Type", "application/json")

	var req registerRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(loginErrResponse{Error: "invalid request"})
		return
	}

	u := strings.TrimSpace(req.Username)
	p := req.Password

	if u == "" || p == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(loginErrResponse{Error: "username and password required"})
		return
	}

	if err := s.CreateUser(u, p); err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(loginErrResponse{Error: "username taken"})
			return
		}
		if err.Error() == "username or password too long" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(loginErrResponse{Error: "username or password too long"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(loginErrResponse{Error: "registration failed"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(loginOKResponse{OK: true, Username: u})
}
