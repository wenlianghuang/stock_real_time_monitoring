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
