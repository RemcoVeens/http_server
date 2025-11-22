package database

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadDB() (*Queries, string, string, string) {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	secret := os.Getenv("SECRET")
	platform := os.Getenv("PLATFORM")
	Polka := os.Getenv("POLKA_KEY")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Println("Could not open db: %w", err)
		os.Exit(1)
	}
	return New(db), platform, secret, Polka
}
