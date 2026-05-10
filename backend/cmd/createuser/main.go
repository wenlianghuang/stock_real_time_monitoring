package main

import (
	"flag"
	"log"
	"os"

	"github.com/joho/godotenv"
	"stock_record/backend/internal/authsrv"
)

func main() {
	_ = godotenv.Load()
	defaultDSN := envOr("DATABASE_URL", "postgres://stockmon_app:devpass@localhost:5432/stockmon?sslmode=disable")

	username := flag.String("username", "", "login username (required)")
	password := flag.String("password", "", "plain password (required)")
	dsn := flag.String("dsn", defaultDSN, "PostgreSQL DSN (default: DATABASE_URL)")
	flag.Parse()

	if *username == "" || *password == "" {
		log.Fatal("usage: createuser -username NAME -password SECRET [-dsn DATABASE_URL]")
	}

	st, err := authsrv.OpenMigrate(*dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer st.Close()

	if err := st.CreateUser(*username, *password); err != nil {
		log.Fatalf("create user: %v", err)
	}
	log.Printf("created user %q", *username)
}

func envOr(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}
