package authsrv

import (
	"database/sql"
	"embed"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

const (
	maxUsernameLen = 128
	maxPasswordLen = 512
)

// ErrUsernameTaken is returned when INSERT violates the unique username constraint.
var ErrUsernameTaken = errors.New("username taken")

// ErrUserNotFound is returned when UPDATE affects no row (unknown username).
var ErrUserNotFound = errors.New("user not found")

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Store executes auth-related queries against the shared PostgreSQL DB.
type Store struct {
	db *sql.DB
}

func OpenMigrate(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		b, err := migrationsFS.ReadFile("migrations/" + e.Name())
		if err != nil {
			_ = db.Close()
			return nil, err
		}
		if _, err := db.Exec(string(b)); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) LookupPasswordHash(username string) (string, error) {
	var hash string
	err := s.db.QueryRow(`SELECT password_hash FROM users WHERE username = $1`, username).Scan(&hash)
	return hash, err
}

func (s *Store) LookupUserID(username string) (int64, error) {
	var id int64
	err := s.db.QueryRow(`SELECT id FROM users WHERE username = $1`, username).Scan(&id)
	return id, err
}

// CreateUser inserts a new user with a bcrypt-hashed password (same cost as login verification).
// Returns ErrUsernameTaken when username already exists.
func (s *Store) CreateUser(username, passwordPlain string) error {
	u := strings.TrimSpace(username)
	if u == "" || passwordPlain == "" {
		return errors.New("username and password required")
	}
	if len(u) > maxUsernameLen || len(passwordPlain) > maxPasswordLen {
		return errors.New("username or password too long")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(passwordPlain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT INTO users (username, password_hash) VALUES ($1, $2)`, u, string(hash))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrUsernameTaken
		}
		return err
	}
	return nil
}

// UpdatePassword sets a new bcrypt hash for username. Validation mirrors CreateUser (length, non-empty).
func (s *Store) UpdatePassword(username, passwordPlain string) error {
	u := strings.TrimSpace(username)
	if u == "" || passwordPlain == "" {
		return errors.New("username and password required")
	}
	if len(u) > maxUsernameLen || len(passwordPlain) > maxPasswordLen {
		return errors.New("username or password too long")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(passwordPlain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`UPDATE users SET password_hash = $1 WHERE username = $2`, string(hash), u)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (s *Store) ListWatchlist(userID int64) ([]string, error) {
	rows, err := s.db.Query(`SELECT symbol FROM user_stocks WHERE user_id = $1 ORDER BY created_at DESC, symbol ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var sym string
		if err := rows.Scan(&sym); err != nil {
			return nil, err
		}
		out = append(out, sym)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) AddToWatchlist(userID int64, symbol string) error {
	sym := strings.TrimSpace(symbol)
	if sym == "" {
		return errors.New("symbol required")
	}
	_, err := s.db.Exec(`INSERT INTO user_stocks (user_id, symbol) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, sym)
	return err
}

func (s *Store) RemoveFromWatchlist(userID int64, symbol string) error {
	sym := strings.TrimSpace(symbol)
	if sym == "" {
		return errors.New("symbol required")
	}
	_, err := s.db.Exec(`DELETE FROM user_stocks WHERE user_id = $1 AND symbol = $2`, userID, sym)
	return err
}
